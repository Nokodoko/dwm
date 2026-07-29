package dwmipc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

// DefaultSocket is dwm's IPC socket path as set by ipcsockpath in config.h.
const DefaultSocket = "/tmp/dwm.sock"

// DefaultTimeout bounds a single request/reply exchange. dwm answers from its
// event loop, so a hung reply means dwm is blocked and we must not block with it.
const DefaultTimeout = 3 * time.Second

// SocketPath returns the socket to dial, honouring a DWM_SOCKET override.
func SocketPath() string {
	if p := os.Getenv("DWM_SOCKET"); p != "" {
		return p
	}
	return DefaultSocket
}

// Conn is a synchronous request/reply connection to dwm.
//
// A Conn must not be used for event subscriptions: once subscribed, dwm
// interleaves unsolicited event frames with replies and the request/reply
// pairing no longer holds. Use Subscribe, which owns a separate connection.
type Conn struct {
	mu   sync.Mutex
	conn net.Conn
}

// Dial connects to the dwm IPC socket.
func Dial(path string) (*Conn, error) {
	if path == "" {
		path = SocketPath()
	}
	c, err := net.DialTimeout("unix", path, DefaultTimeout)
	if err != nil {
		return nil, fmt.Errorf("dwmipc: dial %s: %w", path, err)
	}
	return &Conn{conn: c}, nil
}

// Close releases the connection.
func (c *Conn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.Close()
}

// request sends one message and decodes the reply into out (which may be nil).
func (c *Conn) request(t MessageType, payload any, out any) error {
	body := []byte("{}")
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			return fmt.Errorf("dwmipc: marshal %s: %w", t, err)
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	deadline := time.Now().Add(DefaultTimeout)
	if err := c.conn.SetDeadline(deadline); err != nil {
		return err
	}
	defer func() { _ = c.conn.SetDeadline(time.Time{}) }()

	if err := writePacket(c.conn, t, body); err != nil {
		return fmt.Errorf("dwmipc: write %s: %w", t, err)
	}

	replyType, data, err := readPacket(c.conn)
	if err != nil {
		return fmt.Errorf("dwmipc: read reply to %s: %w", t, err)
	}
	if replyType != t {
		return fmt.Errorf("dwmipc: reply type %s does not match request %s", replyType, t)
	}
	if err := checkError(data); err != nil {
		return fmt.Errorf("dwmipc: %s: %w", t, err)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("dwmipc: decode %s reply: %w", t, err)
	}
	return nil
}

// checkError reports a dwm-side failure envelope, if the payload is one.
// Successful data replies are arrays, so only objects are inspected.
func checkError(data []byte) error {
	trimmed := bytes.TrimLeft(data, " \t\r\n")
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil
	}
	var r reply
	if err := json.Unmarshal(trimmed, &r); err != nil {
		return nil // not an envelope; let the caller decode it
	}
	if r.Result == "error" {
		if r.Reason != "" {
			return fmt.Errorf("dwm rejected request: %s", r.Reason)
		}
		return fmt.Errorf("dwm rejected request")
	}
	return nil
}

// RunCommand invokes one of the commands registered in ipccommands[]
// (config.h:289-303). Unknown commands and arity/type mismatches are rejected
// by dwm, not by this client.
func (c *Conn) RunCommand(name string, args ...any) error {
	if args == nil {
		args = []any{}
	}
	return c.request(MsgRunCommand, commandRequest{Command: name, Args: args}, nil)
}

// GetMonitors returns every monitor with its tag state and client window IDs.
func (c *Conn) GetMonitors() ([]Monitor, error) {
	var mons []Monitor
	if err := c.request(MsgGetMonitors, nil, &mons); err != nil {
		return nil, err
	}
	return mons, nil
}

// GetTags returns the configured tag names and bitmasks.
func (c *Conn) GetTags() ([]Tag, error) {
	var tags []Tag
	if err := c.request(MsgGetTags, nil, &tags); err != nil {
		return nil, err
	}
	return tags, nil
}

// GetLayouts returns the available layouts and their internal addresses, which
// setlayoutsafe requires as its argument.
func (c *Conn) GetLayouts() ([]LayoutInfo, error) {
	var layouts []LayoutInfo
	if err := c.request(MsgGetLayouts, nil, &layouts); err != nil {
		return nil, err
	}
	return layouts, nil
}

// GetClient returns full detail for one window ID. GetMonitors reports only
// IDs, so building a titled client census costs one call per window.
func (c *Conn) GetClient(win Window) (*DWMClient, error) {
	var cl DWMClient
	if err := c.request(MsgGetDWMClient, clientRequest{ClientWindowID: win}, &cl); err != nil {
		return nil, err
	}
	return &cl, nil
}
