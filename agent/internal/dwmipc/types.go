package dwmipc

import "encoding/json"

// Geometry is an x/y/width/height rectangle as dumped by yajl_dumps.c.
type Geometry struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// TagState mirrors dump_tag_state: three bitmasks over the tag set.
type TagState struct {
	Selected uint `json:"selected"`
	Occupied uint `json:"occupied"`
	Urgent   uint `json:"urgent"`
}

// TagSet is the current/previous selected-tag bitmask pair.
type TagSet struct {
	Current uint `json:"current"`
	Old     uint `json:"old"`
}

// MonitorClients is the client census for one monitor. dwm reports only window
// IDs here (yajl_dumps.c:129-139) -- titles and geometry require a follow-up
// GetClient call per ID.
type MonitorClients struct {
	Selected Window   `json:"selected"`
	Stack    []Window `json:"stack"`
	All      []Window `json:"all"`
}

// Window is an X11 window XID, the identity dwm uses for every client.
type Window uint32

// LayoutSymbol is the current/previous layout glyph (e.g. "[]=", "><>").
type LayoutSymbol struct {
	Current string `json:"current"`
	Old     string `json:"old"`
}

// Layout describes a monitor's arrangement. Address is dwm's internal Layout
// pointer, required as the argument to setlayoutsafe.
type Layout struct {
	Symbol  LayoutSymbol `json:"symbol"`
	Address struct {
		Current uintptr `json:"current"`
		Old     uintptr `json:"old"`
	} `json:"address"`
}

// Bar is the status bar state for a monitor.
type Bar struct {
	Y        int  `json:"y"`
	IsShown  bool `json:"is_shown"`
	IsTop    bool `json:"is_top"`
	WindowID Window `json:"window_id"`
}

// Monitor is one dwm monitor as dumped by dump_monitor.
type Monitor struct {
	MasterFactor    float64        `json:"master_factor"`
	NumMaster       int            `json:"num_master"`
	Num             int            `json:"num"`
	IsSelected      bool           `json:"is_selected"`
	MonitorGeometry Geometry       `json:"monitor_geometry"`
	WindowGeometry  Geometry       `json:"window_geometry"`
	TagSet          TagSet         `json:"tagset"`
	TagState        TagState       `json:"tag_state"`
	Clients         MonitorClients `json:"clients"`
	Layout          Layout         `json:"layout"`
	Bar             Bar            `json:"bar"`
}

// SizeHints is a client's ICCCM size constraints.
type SizeHints struct {
	Base struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"base"`
	Step struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"step"`
	Max struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"max"`
	Min struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"min"`
}

// ClientStates are the boolean flags dwm tracks per client.
type ClientStates struct {
	IsFixed      bool `json:"is_fixed"`
	IsFloating   bool `json:"is_floating"`
	IsUrgent     bool `json:"is_urgent"`
	NeverFocus   bool `json:"never_focus"`
	OldState     bool `json:"old_state"`
	IsFullscreen bool `json:"is_fullscreen"`
}

// DWMClient is one managed window as dumped by dump_client.
//
// NOTE: dwm does not export WM_CLASS over IPC -- only Name (WM_NAME/title).
// Callers needing the class must read it from X directly.
type DWMClient struct {
	Name          string `json:"name"`
	Tags          uint   `json:"tags"`
	WindowID      Window `json:"window_id"`
	MonitorNumber int    `json:"monitor_number"`
	Geometry      struct {
		Current Geometry `json:"current"`
		Old     Geometry `json:"old"`
	} `json:"geometry"`
	SizeHints   SizeHints    `json:"size_hints"`
	BorderWidth struct {
		Current int `json:"current"`
		Old     int `json:"old"`
	} `json:"border_width"`
	States ClientStates `json:"states"`
}

// Tag is one entry from get_tags.
type Tag struct {
	BitMask uint   `json:"bit_mask"`
	Name    string `json:"name"`
	TagIdx  int    `json:"tag_index"`
}

// LayoutInfo is one entry from get_layouts.
type LayoutInfo struct {
	Symbol  string  `json:"symbol"`
	Address uintptr `json:"address"`
}

// commandRequest is the run_command payload. dwm validates argc and arg types
// against the ipccommands[] table in config.h.
type commandRequest struct {
	Command string `json:"command"`
	Args    []any  `json:"args"`
}

// subscribeRequest is the subscribe payload.
type subscribeRequest struct {
	Event  Event  `json:"event"`
	Action string `json:"action"` // "subscribe" | "unsubscribe"
}

// clientRequest is the get_dwm_client payload.
type clientRequest struct {
	ClientWindowID Window `json:"client_window_id"`
}

// reply is dwm's generic result envelope for commands and errors.
type reply struct {
	Result string `json:"result"`
	Reason string `json:"reason"`
}

// EventMessage is one asynchronous push from dwm on a subscribed connection.
// Exactly one field is populated, keyed by the event name.
type EventMessage struct {
	Type Event
	Raw  json.RawMessage
}
