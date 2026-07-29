package dwmipc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// EventStream is a dedicated connection that receives asynchronous pushes from
// dwm. It is separate from Conn because a subscribed socket interleaves event
// frames with replies, which breaks synchronous request/reply pairing.
type EventStream struct {
	conn   net.Conn
	events chan EventMessage
	errs   chan error
}

// Subscribe opens a connection, registers for the given events, and streams
// them until ctx is cancelled or the connection fails.
//
// Passing no events subscribes to AllEvents.
func Subscribe(ctx context.Context, path string, events ...Event) (*EventStream, error) {
	if path == "" {
		path = SocketPath()
	}
	if len(events) == 0 {
		events = AllEvents
	}

	conn, err := net.DialTimeout("unix", path, DefaultTimeout)
	if err != nil {
		return nil, fmt.Errorf("dwmipc: dial %s: %w", path, err)
	}

	for _, ev := range events {
		if err := subscribeOne(conn, ev); err != nil {
			_ = conn.Close()
			return nil, err
		}
	}

	s := &EventStream{
		conn:   conn,
		events: make(chan EventMessage, 32),
		errs:   make(chan error, 1),
	}
	go s.loop(ctx)
	return s, nil
}

// subscribeOne registers a single event and consumes its acknowledgement.
func subscribeOne(conn net.Conn, ev Event) error {
	if err := conn.SetDeadline(time.Now().Add(DefaultTimeout)); err != nil {
		return err
	}
	body, err := json.Marshal(subscribeRequest{Event: ev, Action: "subscribe"})
	if err != nil {
		return err
	}
	if err := writePacket(conn, MsgSubscribe, body); err != nil {
		return fmt.Errorf("dwmipc: subscribe %s: %w", ev, err)
	}
	_, data, err := readPacket(conn)
	if err != nil {
		return fmt.Errorf("dwmipc: subscribe %s ack: %w", ev, err)
	}
	if err := checkError(data); err != nil {
		return fmt.Errorf("dwmipc: subscribe %s: %w", ev, err)
	}
	return nil
}

// Events yields decoded event pushes.
func (s *EventStream) Events() <-chan EventMessage { return s.events }

// Err yields the terminal error that ended the stream, if any.
func (s *EventStream) Err() <-chan error { return s.errs }

// Close tears down the stream.
func (s *EventStream) Close() error { return s.conn.Close() }

// loop reads event frames until the context is cancelled or the socket fails.
func (s *EventStream) loop(ctx context.Context) {
	defer close(s.events)

	go func() {
		<-ctx.Done()
		_ = s.conn.Close() // unblock the pending read
	}()

	for {
		// Events arrive whenever the user acts, which may be never; no deadline.
		if err := s.conn.SetReadDeadline(time.Time{}); err != nil {
			s.fail(ctx, err)
			return
		}
		typ, data, err := readPacket(s.conn)
		if err != nil {
			s.fail(ctx, err)
			return
		}
		if typ != MsgEvent {
			continue // ignore stray non-event frames
		}

		// dwm wraps each event as {"<event_name>": {...}}.
		var envelope map[string]json.RawMessage
		if err := json.Unmarshal(data, &envelope); err != nil {
			continue
		}
		for name, raw := range envelope {
			select {
			case s.events <- EventMessage{Type: Event(name), Raw: raw}:
			case <-ctx.Done():
				return
			}
		}
	}
}

// fail reports a terminal error unless the stream was closed deliberately.
func (s *EventStream) fail(ctx context.Context, err error) {
	if ctx.Err() != nil {
		return // cancellation, not a failure
	}
	select {
	case s.errs <- err:
	default:
	}
}
