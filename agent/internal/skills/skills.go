// Package skills exposes dwm window management to the model as callable tools.
//
// Every skill is a thin, auditable wrapper over one or two dwm IPC commands.
// Nothing here interprets free text: the model chooses a skill and supplies
// typed arguments, and this layer translates them into the fixed command set
// registered in ipccommands[].
package skills

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Nokodoko/dwm-agent/internal/dwmipc"
	"github.com/Nokodoko/dwm-agent/internal/llm"
)

// Skill is one callable capability.
type Skill struct {
	Tool llm.Tool
	Run  func(ctx context.Context, c *dwmipc.Conn, args json.RawMessage) (string, error)
}

// Set is the ordered skill registry advertised to the model.
type Set struct {
	skills map[string]Skill
	order  []string
}

// obj is shorthand for a JSON Schema fragment.
type obj = map[string]any

func schema(props obj, required ...string) obj {
	if required == nil {
		required = []string{}
	}
	return obj{"type": "object", "properties": props, "required": required}
}

func prop(typ, desc string) obj { return obj{"type": typ, "description": desc} }

// New builds the default skill set.
func New() *Set {
	s := &Set{skills: map[string]Skill{}}

	s.add("list_windows",
		"List every window dwm manages: id, title, tag bitmask, monitor, and float/fullscreen state. Call this before acting so window ids are current.",
		schema(obj{}),
		func(ctx context.Context, c *dwmipc.Conn, _ json.RawMessage) (string, error) {
			return Snapshot(c)
		})

	s.add("focus_window",
		"Focus a window by id, switching monitor and tag if needed.",
		schema(obj{"window_id": prop("integer", "X window id from list_windows")}, "window_id"),
		func(ctx context.Context, c *dwmipc.Conn, raw json.RawMessage) (string, error) {
			var a struct {
				WindowID uint32 `json:"window_id"`
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return "", err
			}
			if err := c.RunCommand("focuswin", a.WindowID); err != nil {
				return "", err
			}
			return fmt.Sprintf("focused window %d", a.WindowID), nil
		})

	s.add("close_window",
		"Close a window by id. This asks the application to quit; unsaved work may prompt.",
		schema(obj{"window_id": prop("integer", "X window id from list_windows")}, "window_id"),
		func(ctx context.Context, c *dwmipc.Conn, raw json.RawMessage) (string, error) {
			var a struct {
				WindowID uint32 `json:"window_id"`
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return "", err
			}
			// killclient acts on the focused client, so focus must move first.
			if err := c.RunCommand("focuswin", a.WindowID); err != nil {
				return "", err
			}
			if err := c.RunCommand("killclient", 0); err != nil {
				return "", err
			}
			return fmt.Sprintf("closed window %d", a.WindowID), nil
		})

	s.add("move_window_to_tag",
		"Move a window to a tag. Tags are a bitmask: tag 1 is 1, tag 2 is 2, tag 3 is 4, tag 4 is 8, and so on.",
		schema(obj{
			"window_id": prop("integer", "X window id from list_windows"),
			"tag_mask":  prop("integer", "Destination tag bitmask, e.g. 4 for tag 3"),
		}, "window_id", "tag_mask"),
		func(ctx context.Context, c *dwmipc.Conn, raw json.RawMessage) (string, error) {
			var a struct {
				WindowID uint32 `json:"window_id"`
				TagMask  uint32 `json:"tag_mask"`
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return "", err
			}
			// tag acts on the focused client, same as killclient.
			if err := c.RunCommand("focuswin", a.WindowID); err != nil {
				return "", err
			}
			if err := c.RunCommand("tag", a.TagMask); err != nil {
				return "", err
			}
			return fmt.Sprintf("moved window %d to tag mask %d", a.WindowID, a.TagMask), nil
		})

	s.add("view_tag",
		"Switch the focused monitor to view a tag.",
		schema(obj{"tag_mask": prop("integer", "Tag bitmask to view, e.g. 4 for tag 3")}, "tag_mask"),
		func(ctx context.Context, c *dwmipc.Conn, raw json.RawMessage) (string, error) {
			var a struct {
				TagMask uint32 `json:"tag_mask"`
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return "", err
			}
			if err := c.RunCommand("view", a.TagMask); err != nil {
				return "", err
			}
			return fmt.Sprintf("viewing tag mask %d", a.TagMask), nil
		})

	s.add("toggle_floating",
		"Toggle floating for a window.",
		schema(obj{"window_id": prop("integer", "X window id from list_windows")}, "window_id"),
		func(ctx context.Context, c *dwmipc.Conn, raw json.RawMessage) (string, error) {
			var a struct {
				WindowID uint32 `json:"window_id"`
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return "", err
			}
			if err := c.RunCommand("focuswin", a.WindowID); err != nil {
				return "", err
			}
			if err := c.RunCommand("togglefloating"); err != nil {
				return "", err
			}
			return fmt.Sprintf("toggled floating on window %d", a.WindowID), nil
		})

	s.add("set_layout",
		"Set the layout on the focused monitor. Symbols: \"[]=\" tiled, \"[M]\" monocle, \"><>\" floating.",
		schema(obj{"symbol": prop("string", "Layout symbol from list_windows")}, "symbol"),
		func(ctx context.Context, c *dwmipc.Conn, raw json.RawMessage) (string, error) {
			var a struct {
				Symbol string `json:"symbol"`
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return "", err
			}
			layouts, err := c.GetLayouts()
			if err != nil {
				return "", err
			}
			for _, l := range layouts {
				if l.Symbol == a.Symbol {
					// setlayoutsafe validates this address against layouts[].
					if err := c.RunCommand("setlayoutsafe", uint64(l.Address)); err != nil {
						return "", err
					}
					return fmt.Sprintf("layout set to %q", a.Symbol), nil
				}
			}
			return "", fmt.Errorf("no layout with symbol %q", a.Symbol)
		})

	s.add("move_window_to_monitor",
		"Send the focused window to another monitor by index.",
		schema(obj{"monitor": prop("integer", "Monitor number from list_windows")}, "monitor"),
		func(ctx context.Context, c *dwmipc.Conn, raw json.RawMessage) (string, error) {
			var a struct {
				Monitor uint32 `json:"monitor"`
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return "", err
			}
			if err := c.RunCommand("tagmon", a.Monitor); err != nil {
				return "", err
			}
			return fmt.Sprintf("sent focused window to monitor %d", a.Monitor), nil
		})

	s.add("set_master_factor",
		"Set the master area fraction on the focused monitor, between 0.05 and 0.95.",
		schema(obj{"factor": prop("number", "Absolute fraction, e.g. 0.6")}, "factor"),
		func(ctx context.Context, c *dwmipc.Conn, raw json.RawMessage) (string, error) {
			var a struct {
				Factor float64 `json:"factor"`
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return "", err
			}
			if a.Factor < 0.05 || a.Factor > 0.95 {
				return "", fmt.Errorf("factor %.2f out of range 0.05-0.95", a.Factor)
			}
			// dwm treats values above 1.0 as absolute; +1 selects that path.
			if err := c.RunCommand("setmfact", a.Factor+1.0); err != nil {
				return "", err
			}
			return fmt.Sprintf("master factor set to %.2f", a.Factor), nil
		})

	s.add("launch",
		"Launch a program. Only programs on dwm's spawnallow list can start; anything else is refused by the window manager.",
		schema(obj{"command": prop("string", "Command line, split on spaces. No shell: pipes and semicolons are literal.")}, "command"),
		func(ctx context.Context, c *dwmipc.Conn, raw json.RawMessage) (string, error) {
			var a struct {
				Command string `json:"command"`
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return "", err
			}
			if strings.TrimSpace(a.Command) == "" {
				return "", fmt.Errorf("empty command")
			}
			if err := c.RunCommand("spawnsafe", a.Command); err != nil {
				return "", err
			}
			// dwm forks without reporting back, and refusals are logged to its
			// stderr, so success here means "accepted", not "running".
			return fmt.Sprintf("requested launch of %q (refused silently if not in spawnallow)", a.Command), nil
		})

	s.add("shell",
		"Run a shell command with zsh and return its combined stdout and stderr plus the exit code. "+
			"Use for filesystem queries, git, package and service inspection, text processing -- anything "+
			"the window skills do not cover. Prefer this over launch for commands whose output you need; "+
			"launch is for starting GUI programs.",
		schema(obj{
			"command": prop("string", "Command line, evaluated by zsh -c. Pipes, globs and redirection work."),
			"dir":     prop("string", "Working directory. Optional; defaults to the user's home."),
		}, "command"),
		runShell)

	return s
}

// Shell execution limits. The model is not a careful operator: a runaway
// command would otherwise hang the scratchpad, and multi-megabyte output would
// blow the context window on the next turn.
const (
	shellTimeout   = 60 * time.Second
	shellMaxOutput = 32 << 10 // 32 KiB
)

// shellMarker separates .zshrc startup output from the command's own output.
//
// Running interactively is what makes the user's aliases resolve, but it also
// sources .zshrc, which prints plugin chatter (zoxide's config warning, for
// one) before the command runs. Echoing a marker first and discarding
// everything up to it strips that noise whatever its source, which beats
// pattern-matching each offending plugin. Startup always precedes the marker,
// so slicing is safe even though stdout and stderr share one pipe.
const shellMarker = "__dwm_agent_begin_9f3a1c__"

// runShell executes a command through zsh.
//
// SECURITY: this runs with the user's full privileges and deliberately bypasses
// dwm's spawnallow list, which only gates the launch skill. Window titles reach
// the model as context and are attacker-controlled, so a prompt-injected model
// can reach this. The system prompt warns about it; there is no sandbox here.
func runShell(ctx context.Context, _ *dwmipc.Conn, raw json.RawMessage) (string, error) {
	var a struct {
		Command string `json:"command"`
		Dir     string `json:"dir"`
	}
	if err := json.Unmarshal(raw, &a); err != nil {
		return "", err
	}
	if strings.TrimSpace(a.Command) == "" {
		return "", fmt.Errorf("empty command")
	}

	dir := a.Dir
	if dir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			dir = home
		}
	}

	cctx, cancel := context.WithTimeout(ctx, shellTimeout)
	defer cancel()

	// -i sources .zshrc so the user's aliases and functions resolve, which is
	// the whole point of using their shell rather than sh. The marker above
	// pays for the startup noise that comes with it.
	cmd := exec.CommandContext(cctx, zshPath(), "-ic", "echo "+shellMarker+"\n"+a.Command)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()

	body := string(out)
	if i := strings.Index(body, shellMarker); i >= 0 {
		body = body[i+len(shellMarker):]
	}
	body = strings.TrimRight(strings.TrimLeft(body, "\r\n"), "\n")
	if len(body) > shellMaxOutput {
		body = body[:shellMaxOutput] + fmt.Sprintf("\n… truncated at %d bytes", shellMaxOutput)
	}
	if body == "" {
		body = "(no output)"
	}

	if cctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("timed out after %s; partial output:\n%s", shellTimeout, body)
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// A nonzero exit is information, not a dispatch failure -- the model
		// should see the code and the output and decide what to do next.
		return fmt.Sprintf("exit %d\n%s", exitErr.ExitCode(), body), nil
	}
	if err != nil {
		return "", fmt.Errorf("could not run: %w", err)
	}
	return body, nil
}

// zshPath resolves the zsh binary, preferring whatever is on PATH.
func zshPath() string {
	if p, err := exec.LookPath("zsh"); err == nil {
		return p
	}
	return "/bin/zsh"
}

func (s *Set) add(name, desc string, params obj, run func(context.Context, *dwmipc.Conn, json.RawMessage) (string, error)) {
	s.order = append(s.order, name)
	s.skills[name] = Skill{
		Tool: llm.Tool{
			Type: "function",
			Function: llm.ToolFunction{
				Name:        name,
				Description: desc,
				Parameters:  params,
			},
		},
		Run: run,
	}
}

// Tools returns the skill set as model-facing tool definitions.
func (s *Set) Tools() []llm.Tool {
	out := make([]llm.Tool, 0, len(s.order))
	for _, n := range s.order {
		out = append(out, s.skills[n].Tool)
	}
	return out
}

// Names lists skill names in registration order.
func (s *Set) Names() []string { return append([]string(nil), s.order...) }

// Dispatch runs a named skill. An unknown name is returned as an error string
// for the model to read rather than a Go error, so it can correct itself.
func (s *Set) Dispatch(ctx context.Context, c *dwmipc.Conn, name string, args json.RawMessage) string {
	sk, ok := s.skills[name]
	if !ok {
		return fmt.Sprintf("error: no such skill %q; available: %s", name, strings.Join(s.order, ", "))
	}
	if len(args) == 0 {
		args = json.RawMessage("{}")
	}
	out, err := sk.Run(ctx, c, args)
	if err != nil {
		return "error: " + err.Error()
	}
	return out
}

// Snapshot renders the live X session as text for the model's context.
func Snapshot(c *dwmipc.Conn) (string, error) {
	mons, err := c.GetMonitors()
	if err != nil {
		return "", err
	}
	tags, err := c.GetTags()
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("tags: ")
	for i, t := range tags {
		if i > 0 {
			sb.WriteString(" ")
		}
		name := t.Name
		if strings.TrimSpace(name) == "" {
			name = fmt.Sprintf("t%d", i+1)
		}
		fmt.Fprintf(&sb, "%s=%d", name, t.BitMask)
	}
	sb.WriteString("\n")

	for _, m := range mons {
		sel := ""
		if m.IsSelected {
			sel = " (focused)"
		}
		fmt.Fprintf(&sb, "\nmonitor %d%s %dx%d layout=%q master_factor=%.2f viewing_tags=%d\n",
			m.Num, sel, m.MonitorGeometry.Width, m.MonitorGeometry.Height,
			m.Layout.Symbol.Current, m.MasterFactor, m.TagState.Selected)

		if len(m.Clients.All) == 0 {
			sb.WriteString("  (no windows)\n")
			continue
		}
		for _, win := range m.Clients.All {
			cl, err := c.GetClient(win)
			if err != nil {
				fmt.Fprintf(&sb, "  window %d: unreadable (%v)\n", win, err)
				continue
			}
			marker := " "
			if win == m.Clients.Selected {
				marker = "*"
			}
			flags := []string{}
			if cl.States.IsFloating {
				flags = append(flags, "floating")
			}
			if cl.States.IsFullscreen {
				flags = append(flags, "fullscreen")
			}
			if cl.States.IsUrgent {
				flags = append(flags, "urgent")
			}
			extra := ""
			if len(flags) > 0 {
				extra = " [" + strings.Join(flags, ",") + "]"
			}
			fmt.Fprintf(&sb, " %s id=%d tags=%d%s title=%q\n",
				marker, cl.WindowID, cl.Tags, extra, cl.Name)
		}
	}
	return sb.String(), nil
}
