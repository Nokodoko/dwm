// Package tui is the dwm-agent chat interface.
package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/Nokodoko/dwm-agent/internal/agentloop"
	"github.com/Nokodoko/dwm-agent/internal/dwmipc"
	"github.com/Nokodoko/dwm-agent/internal/llm"
	"github.com/Nokodoko/dwm-agent/internal/skills"
	"github.com/Nokodoko/dwm-agent/internal/theme"
)

type role int

const (
	roleUser role = iota
	roleAgent
	roleSystem
	roleTool
	roleError
)

// entry is one rendered line-block in the transcript.
type entry struct {
	role role
	text string

	// Cached glamour output. Rendering markdown is expensive enough that
	// redoing it for every message on every keystroke is what makes a long
	// transcript feel like it has hung; the cache makes it once per message.
	// renderWidth records the width it was built at, so a resize invalidates
	// it rather than leaving text wrapped for the old geometry.
	rendered    string
	renderWidth int
}

// Model is the bubbletea model.
type Model struct {
	vp   viewport.Model
	ta   textarea.Model
	sp   spinner.Model
	st   theme.Styles
	glam *glamour.TermRenderer

	conn   *dwmipc.Conn
	client *llm.Client
	reg    *llm.Registry
	skills *skills.Set

	history    []llm.Message
	transcript []entry

	busy   bool
	width  int
	height int
	status string

	// mode is the prompt's editing mode; pending holds a half-finished
	// operator such as the `d` of `dw` (see vim.go).
	mode    mode
	pending rune
}

// turnResult carries a completed agentic turn back to the UI.
type turnResult struct {
	steps   []entry
	history []llm.Message
	err     error
}

// bangResult carries the output of a direct "!command" run.
type bangResult struct {
	cmd string
	out string
}

// skillResult carries the output of a filesystem skill run as /<slug>.
type skillResult struct {
	label string
	out   string
}

// New builds the TUI.
func New(conn *dwmipc.Conn, client *llm.Client, reg *llm.Registry, sk *skills.Set) Model {
	st := theme.New()

	ta := textarea.New()
	ta.Placeholder = "Ask me anything…  (/help for commands)"
	ta.Prompt = "▌ "
	ta.CharLimit = 4000
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = st.Spinner

	vp := viewport.New(80, 20)

	// tokyo-night matches the palette; older glamour builds lack it.
	glam, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("tokyo-night"),
		glamour.WithWordWrap(78),
	)
	if err != nil {
		glam, _ = glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithWordWrap(78),
		)
	}

	m := Model{
		vp: vp, ta: ta, sp: sp, st: st, glam: glam,
		conn: conn, client: client, reg: reg, skills: sk,
		history: []llm.Message{{Role: llm.RoleSystem, Content: agentloop.SystemPrompt}},
		status:  "ready",
	}
	m.push(roleSystem, fmt.Sprintf("connected to dwm at %s · model %s",
		dwmipc.SocketPath(), client.Backend().Name))
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.sp.Tick)
}

func (m *Model) push(r role, text string) {
	m.transcript = append(m.transcript, entry{role: r, text: text})
}

// Update handles input, resize and turn completion.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		m.renderTranscript()
		return m, nil

	case spinner.TickMsg:
		if m.busy {
			var cmd tea.Cmd
			m.sp, cmd = m.sp.Update(msg)
			return m, cmd
		}
		return m, nil

	case turnResult:
		m.busy = false
		m.status = "ready"
		if msg.err != nil {
			m.push(roleError, msg.err.Error())
		} else {
			m.history = msg.history
			m.transcript = append(m.transcript, msg.steps...)
		}
		m.renderTranscript()
		m.vp.GotoBottom()
		return m, nil

	case skillResult:
		m.busy = false
		m.status = "ready"
		m.push(roleTool, msg.label+"\n"+msg.out)
		// Recorded as conversation so a follow-up ("why did that fail?") has
		// the invocation and its output already in context.
		m.history = append(m.history, llm.Message{
			Role:    llm.RoleUser,
			Content: fmt.Sprintf("I ran the %s skill myself. Its output was:\n%s", msg.label, msg.out),
		})
		m.renderTranscript()
		m.vp.GotoBottom()
		return m, nil

	case bangResult:
		m.busy = false
		m.status = "ready"
		m.push(roleTool, "! "+msg.cmd+"\n"+msg.out)
		// Record it as conversation so a follow-up ("why did that fail?") has
		// the command and its output already in context.
		m.history = append(m.history, llm.Message{
			Role:    llm.RoleUser,
			Content: fmt.Sprintf("I ran `%s` in the shell myself. Its output was:\n%s", msg.cmd, msg.out),
		})
		m.renderTranscript()
		m.vp.GotoBottom()
		return m, nil

	case editorDone:
		if msg.path != "" {
			if b, err := os.ReadFile(msg.path); err == nil {
				// A trailing newline is an artefact of the editor saving a
				// file, not something the user typed into the prompt.
				m.ta.SetValue(strings.TrimRight(string(b), "\n"))
			}
			os.Remove(msg.path)
		}
		if msg.err != nil {
			m.push(roleError, "editor: "+msg.err.Error())
			m.renderTranscript()
		}
		// Returning from vim lands in normal mode, as :wq would.
		m.mode = modeNormal
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEsc:
			// Esc leaves insert mode rather than quitting: modal editing needs
			// it far more than the app needs a second quit key. Ctrl+C and
			// /quit both still exit.
			m.mode = modeNormal
			m.pending = 0
			return m, nil
		case tea.KeyCtrlL:
			m.transcript = nil
			m.renderTranscript()
			// The viewport keeps its scroll offset across SetContent, so after
			// clearing a transcript taller than the window it would sit past
			// the end of the new (empty) content and show nothing.
			m.vp.GotoTop()
			// tea.ClearScreen forces a full repaint. Without it, replacing a
			// frame much taller than the window -- the state a big `!ls` leaves
			// behind -- leaves the alt-screen holding stale rows that the
			// shorter frame never overwrites, which reads as a blank or frozen
			// client. It also makes Ctrl+L do what /help has always claimed:
			// clear the screen, not just the transcript.
			return m, tea.ClearScreen
		// Ctrl+G hands the prompt to $EDITOR from ANY mode. `v` does the same
		// but only in normal mode, which means the escape is unavailable
		// exactly when it is most wanted -- part-way through typing a long
		// prompt in insert mode. Handled here, above the mode dispatch, so
		// insert mode cannot swallow it as literal input.
		case tea.KeyCtrlG:
			if m.busy {
				return m, nil
			}
			return m, m.openEditor()

		case tea.KeyPgUp:
			m.vp.HalfPageUp()
			return m, nil
		case tea.KeyPgDown:
			m.vp.HalfPageDown()
			return m, nil
		case tea.KeyEnter:
			if m.busy {
				return m, nil
			}
			input := strings.TrimSpace(m.ta.Value())
			if input == "" {
				return m, nil
			}
			m.ta.Reset()
			if strings.HasPrefix(input, "/") {
				return m.command(input)
			}
			if strings.HasPrefix(input, "!") {
				return m.bang(input)
			}
			m.push(roleUser, input)
			m.history = append(m.history, llm.Message{Role: llm.RoleUser, Content: input})
			m.busy = true
			m.status = "thinking"
			m.renderTranscript()
			m.vp.GotoBottom()
			return m, tea.Batch(m.sp.Tick, m.runTurn())
		}

		// Normal mode consumes every remaining key. Falling through would let
		// the textarea insert the literal motion characters.
		if m.mode == modeNormal {
			return m, m.normalKey(msg)
		}
	}

	var cmd tea.Cmd
	m.ta, cmd = m.ta.Update(msg)
	cmds = append(cmds, cmd)
	m.vp, cmd = m.vp.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

// runTurn drives the agentic loop off the UI goroutine.
func (m Model) runTurn() tea.Cmd {
	history := append([]llm.Message(nil), m.history...)
	client, sk, conn := m.client, m.skills, m.conn

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		steps, next, err := agentloop.Run(ctx, client, sk, conn, history)
		if err != nil {
			return turnResult{err: err}
		}
		return turnResult{steps: toEntries(steps), history: next}
	}
}

// bang runs a "!command" line directly in the shell, skipping the model.
//
// The point is immediacy: when you already know the command, round-tripping it
// through an LLM is slower and can rewrite what you asked for.
// runSkill dispatches a filesystem skill invoked as a slash command.
//
// It runs off the update loop like bang() does: a display switch restarts five
// programs and sleeps between them, so running it inline would freeze the TUI
// for the duration and make a working switch look like a hang.
func (m Model) runSkill(fsk skills.FSSkill, label string, args json.RawMessage) (tea.Model, tea.Cmd) {
	label = strings.TrimSpace(label)
	m.push(roleUser, label)
	m.busy = true
	m.status = "running " + fsk.Slug
	m.renderTranscript()
	m.vp.GotoBottom()

	sk, conn, name := m.skills, m.conn, fsk.Name
	return m, tea.Batch(m.sp.Tick, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		return skillResult{label: label, out: sk.Dispatch(ctx, conn, name, args)}
	})
}

func (m Model) bang(line string) (tea.Model, tea.Cmd) {
	cmdline := strings.TrimSpace(strings.TrimPrefix(line, "!"))
	if cmdline == "" {
		m.push(roleError, "! needs a command, e.g. !git status")
		m.renderTranscript()
		return m, nil
	}

	m.push(roleUser, "! "+cmdline)
	m.busy = true
	m.status = "running"
	m.renderTranscript()
	m.vp.GotoBottom()

	sk, conn := m.skills, m.conn
	return m, tea.Batch(m.sp.Tick, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		args, err := json.Marshal(map[string]string{"command": cmdline})
		if err != nil {
			return bangResult{cmd: cmdline, out: "error: " + err.Error()}
		}
		return bangResult{cmd: cmdline, out: sk.Dispatch(ctx, conn, "shell", args)}
	})
}

// toEntries maps loop steps onto transcript entries.
func toEntries(steps []agentloop.Step) []entry {
	out := make([]entry, 0, len(steps))
	for _, s := range steps {
		switch s.Kind {
		case agentloop.KindTool:
			out = append(out, entry{role: roleTool, text: s.Text})
		case agentloop.KindError:
			out = append(out, entry{role: roleError, text: s.Text})
		default:
			out = append(out, entry{role: roleAgent, text: s.Text})
		}
	}
	return out
}

// command handles slash commands, which never reach the model.
func (m Model) command(line string) (tea.Model, tea.Cmd) {
	fields := strings.Fields(line)
	cmd := fields[0]
	arg := strings.TrimSpace(strings.TrimPrefix(line, cmd))

	switch cmd {
	case "/help":
		// Filesystem skills are listed live rather than hardcoded: they are
		// whatever is in .agents/skills right now, and a help text that named a
		// fixed set would go stale the first time one was added.
		skillLine := ""
		if names := m.skills.SlashNames(); len(names) > 0 {
			skillLine = strings.Join(names, ", ") + " — skills from .agents/skills (flags: -r, -m, --dry-run)\n"
		}
		m.push(roleSystem, "monty answers general questions; window management is\n"+
			"always available, just ask in plain language.\n\n"+
			"!<command> — run it in zsh immediately, without the model\n"+
			"/model [name] — list or switch backend\n"+
			"/dwm, /windows — show the live window census\n"+
			skillLine+
			"/skills [-l|-v] — list callable skills; -l adds descriptions, -v adds arguments\n"+
			"/clear — reset the conversation\n"+
			"/quit — close\n\n"+
			"Ctrl+L clears the screen · PgUp/PgDn scroll · Ctrl+C quits\n\n"+
			"modal editing — Esc leaves insert mode, it no longer quits:\n"+
			"  motions   h l j k · w b e · 0 ^ $ · gg G\n"+
			"  edits     x X · D C · s S · dw db dd d$ d0 · cw cb cc c$ · diw ciw\n"+
			"  insert    i a I A o O\n"+
			"  v         open the prompt in $EDITOR, :wq to come back\n\n"+
			"Ctrl+G opens $EDITOR from any mode, seeded with whatever is already\n"+
			"in the prompt. Save and quit and the buffer replaces the prompt.")

	case "/model":
		if arg == "" {
			ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
			defer cancel()
			m.push(roleSystem, "backends:\n"+m.reg.List(ctx, m.client.Backend().Name))
			break
		}
		b, ok := m.reg.Get(arg)
		if !ok {
			m.push(roleError, fmt.Sprintf("unknown backend %q; try: %s",
				arg, strings.Join(m.reg.Names(), ", ")))
			break
		}
		m.client.SetBackend(b)
		// Conversation history is kept: a swap changes who answers, not what
		// has been said.
		m.push(roleSystem, fmt.Sprintf("model → %s (%s)", b.Name, b.Model))

	// /dwm is the discoverable name; the skills themselves are always armed, so
	// this only prints state rather than switching the agent into a mode.
	case "/dwm", "/windows":
		snap, err := skills.Snapshot(m.conn)
		if err != nil {
			m.push(roleError, err.Error())
			break
		}
		m.push(roleTool, "window census\n"+snap)

	// Bare /skills stays a one-line roster; -l expands it with each skill's
	// description and origin, which is what you want when a markdown skill in
	// .agents/skills is not behaving and you need to know which file defined it.
	// /skill is accepted alongside /skills: the singular is what gets typed
	// about half the time, and failing it as "unknown command" while the
	// plural works is a pointless papercut.
	case "/skills", "/skill":
		switch arg {
		case "":
			m.push(roleSystem, "skills: "+strings.Join(m.skills.Names(), ", "))
		case "-l", "--list":
			m.push(roleSystem, m.skills.Describe(false))
		case "-lv", "-v", "--verbose":
			m.push(roleSystem, m.skills.Describe(true))
		default:
			m.push(roleError, fmt.Sprintf("unknown flag %q; try /skills, /skills -l, or /skills -v", arg))
		}

	case "/clear":
		m.history = []llm.Message{{Role: llm.RoleSystem, Content: agentloop.SystemPrompt}}
		m.transcript = nil
		m.push(roleSystem, "conversation cleared")

	case "/quit", "/exit":
		return m, tea.Quit

	default:
		// Filesystem skills are reachable as /<slug>, so `/edp` runs the skill
		// in .agents/skills/edp. Without this they were callable only by the
		// model choosing them from prose, which is not what a command is for.
		if fsk, ok := m.skills.LookupSlash(cmd); ok {
			args, err := skills.ParseSlashArgs(fsk, arg)
			if err != nil {
				m.push(roleError, err.Error())
				break
			}
			return m.runSkill(fsk, cmd+" "+arg, args)
		}
		known := m.skills.SlashNames()
		if len(known) > 0 {
			m.push(roleError, "unknown command "+cmd+" — /help for the list, or: "+strings.Join(known, " "))
		} else {
			m.push(roleError, "unknown command "+cmd+" — /help for the list")
		}
	}

	m.renderTranscript()
	m.vp.GotoBottom()
	return m, nil
}

// layout resizes the panes to the terminal.
func (m *Model) layout() {
	inputHeight := 5 // textarea + border
	headerHeight := 1
	statusHeight := 1

	vpHeight := m.height - inputHeight - headerHeight - statusHeight - 2
	if vpHeight < 3 {
		vpHeight = 3
	}
	vpWidth := m.width - 4
	if vpWidth < 20 {
		vpWidth = 20
	}

	m.vp.Width = vpWidth
	m.vp.Height = vpHeight
	m.ta.SetWidth(m.width - 4)

	if g, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("tokyo-night"),
		glamour.WithWordWrap(vpWidth-2),
	); err == nil {
		m.glam = g
	}
}

// renderTranscript rebuilds the viewport content.
func (m *Model) renderTranscript() {
	var sb strings.Builder
	for i := range m.transcript {
		e := &m.transcript[i]
		if i > 0 {
			sb.WriteString("\n")
		}
		switch e.role {
		case roleUser:
			sb.WriteString(m.st.UserLabel.Render("❯ you") + "\n")
			sb.WriteString(m.st.UserText.Render(e.text) + "\n")
		case roleAgent:
			sb.WriteString(m.st.AgentLabel.Render("☉ agent") + "\n")
			if e.rendered == "" || e.renderWidth != m.vp.Width {
				e.rendered = m.markdown(e.text)
				e.renderWidth = m.vp.Width
			}
			sb.WriteString(e.rendered)
		case roleTool:
			sb.WriteString(m.st.Tool.Render(e.text) + "\n")
		case roleSystem:
			sb.WriteString(m.st.SysLabel.Render(e.text) + "\n")
		case roleError:
			sb.WriteString(m.st.Err.Render("✗ "+e.text) + "\n")
		}
	}
	m.vp.SetContent(sb.String())
}

// markdown renders agent prose, falling back to raw text if glamour fails.
func (m *Model) markdown(s string) string {
	if m.glam == nil {
		return s + "\n"
	}
	out, err := m.glam.Render(s)
	if err != nil {
		return s + "\n"
	}
	return out
}

// View draws the frame.
func (m Model) View() string {
	if m.width == 0 {
		return "starting…"
	}

	backend := m.client.Backend()
	title := m.st.Header.Render("☉ monty")
	model := m.st.HeaderKey.Render(backend.Name)
	host := m.st.HeaderDim.Render(backend.Model)
	header := lipgloss.JoinHorizontal(lipgloss.Top, title, model, host)

	status := m.st.Status.Render(m.status)
	if m.busy {
		status = m.sp.View() + m.st.StatusWarn.Render(" "+m.status)
	} else {
		status = m.st.StatusOK.Render("● ") + status
	}
	badge := m.st.ModeInsert.Render(m.mode.String())
	hint := m.st.Status.Render("  enter send · /help · ctrl+c quit")
	if m.mode == modeNormal {
		badge = m.st.ModeNormal.Render(m.mode.String())
		hint = m.st.Status.Render("  i insert · ctrl+g editor · enter send · ctrl+c quit")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		m.st.Viewport.Width(m.vp.Width).Render(m.vp.View()),
		m.st.Input.Width(m.width-2).Render(m.ta.View()),
		lipgloss.JoinHorizontal(lipgloss.Top, badge, status, hint),
	)
}
