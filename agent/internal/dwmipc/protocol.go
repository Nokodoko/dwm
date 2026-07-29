// Package dwmipc implements a client for the mihirlad55 dwm-ipc protocol.
//
// The wire format is a 12-byte packed header followed by a JSON payload:
//
//	magic [7]byte  // "DWM-IPC"
//	size  uint32   // payload length, NATIVE endian (dwm memcpy's it raw)
//	type  uint8    // MessageType
//
// See ipc.c:129-148 for the reference reader.
package dwmipc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// Magic is the fixed header prefix on every packet (ipc.h:11).
const Magic = "DWM-IPC"

// HeaderLen is the size of a packed dwm_ipc_header_t: 7 + 4 + 1.
const HeaderLen = len(Magic) + 4 + 1

// MaxMessageSize bounds a single payload so a desynchronised stream cannot
// drive an unbounded allocation.
const MaxMessageSize = 1 << 24 // 16 MiB

// MessageType mirrors the IPCMessageType enum (ipc.h:19-27).
type MessageType uint8

const (
	MsgRunCommand   MessageType = 0
	MsgGetMonitors  MessageType = 1
	MsgGetTags      MessageType = 2
	MsgGetLayouts   MessageType = 3
	MsgGetDWMClient MessageType = 4
	MsgSubscribe    MessageType = 5
	MsgEvent        MessageType = 6
)

func (t MessageType) String() string {
	switch t {
	case MsgRunCommand:
		return "run_command"
	case MsgGetMonitors:
		return "get_monitors"
	case MsgGetTags:
		return "get_tags"
	case MsgGetLayouts:
		return "get_layouts"
	case MsgGetDWMClient:
		return "get_dwm_client"
	case MsgSubscribe:
		return "subscribe"
	case MsgEvent:
		return "event"
	default:
		return fmt.Sprintf("unknown(%d)", uint8(t))
	}
}

// Event mirrors the IPCEvent bitmask (ipc.h:29-36). The wire representation in
// a subscribe payload is the snake_case name, not the bit.
type Event string

const (
	EventTagChange          Event = "tag_change_event"
	EventClientFocusChange  Event = "client_focus_change_event"
	EventLayoutChange       Event = "layout_change_event"
	EventMonitorFocusChange Event = "monitor_focus_change_event"
	EventFocusedTitleChange Event = "focused_title_change_event"
	EventFocusedStateChange Event = "focused_state_change_event"
)

// AllEvents is every event dwm can publish, in ipc.h bit order.
var AllEvents = []Event{
	EventTagChange,
	EventClientFocusChange,
	EventLayoutChange,
	EventMonitorFocusChange,
	EventFocusedTitleChange,
	EventFocusedStateChange,
}

// ErrBadMagic is returned when a packet does not begin with Magic, which means
// the stream is desynchronised and the connection must be discarded.
var ErrBadMagic = errors.New("dwmipc: bad packet magic")

// writePacket emits one header+payload frame.
func writePacket(w io.Writer, t MessageType, payload []byte) error {
	if len(payload) > MaxMessageSize {
		return fmt.Errorf("dwmipc: payload %d exceeds max %d", len(payload), MaxMessageSize)
	}
	buf := make([]byte, HeaderLen+len(payload))
	copy(buf, Magic)
	// dwm memcpy's this field with no byte-order conversion (ipc.c:978), so it
	// must be written in the host's native order, not network order.
	binary.NativeEndian.PutUint32(buf[len(Magic):], uint32(len(payload)))
	buf[len(Magic)+4] = byte(t)
	copy(buf[HeaderLen:], payload)

	_, err := w.Write(buf)
	return err
}

// readPacket consumes exactly one header+payload frame.
func readPacket(r io.Reader) (MessageType, []byte, error) {
	var hdr [HeaderLen]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return 0, nil, err
	}
	if string(hdr[:len(Magic)]) != Magic {
		return 0, nil, fmt.Errorf("%w: got %q", ErrBadMagic, hdr[:len(Magic)])
	}

	size := binary.NativeEndian.Uint32(hdr[len(Magic):])
	if size > MaxMessageSize {
		return 0, nil, fmt.Errorf("dwmipc: reply size %d exceeds max %d", size, MaxMessageSize)
	}
	t := MessageType(hdr[len(Magic)+4])

	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return 0, nil, err
	}
	// dwm sizes the frame from yajl's NUL-terminated buffer, so the declared
	// length includes the terminator. Leaving it on breaks encoding/json with
	// "invalid character '\x00' after top-level value".
	payload = bytes.TrimRight(payload, "\x00")
	return t, payload, nil
}
