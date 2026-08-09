// vim.go adds modal editing to the prompt.
//
// The textarea widget stays the single source of truth for both the text and
// the cursor. Normal-mode keys are not implemented against the buffer directly;
// each one is translated into the key message the textarea already binds for
// that motion (see textarea.DefaultKeyMap). Two representations of the cursor
// would drift apart the first time a soft wrap moved one and not the other, so
// there is deliberately only one.
//
// Only motions the widget can express are supported. `e` lands where `w` does,
// and `dd` clears the line rather than removing it, because the widget offers
// no end-of-word stop and no line-delete. Counts (`3w`) and registers are not
// implemented.
package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// mode is the editing mode of the prompt.
type mode int

const (
	// modeInsert is the startup mode: the prompt is a chat box first and a vim
	// buffer second, so typing works without pressing anything.
	modeInsert mode = iota
	modeNormal
)

func (m mode) String() string {
	if m == modeNormal {
		return "NORMAL"
	}
	return "INSERT"
}

// editorDone reports that $EDITOR exited. path is read back into the prompt.
type editorDone struct {
	path string
	err  error
}

// plain, alt and altRune build the key messages textarea.DefaultKeyMap binds.
func plain(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }
func alt(t tea.KeyType) tea.KeyMsg   { return tea.KeyMsg{Type: t, Alt: true} }
func altRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}, Alt: true}
}

// feed replays key messages into the textarea, collecting the commands it
// returns so cursor blink survives a synthesised motion.
func (m *Model) feed(msgs ...tea.KeyMsg) tea.Cmd {
	var cmds []tea.Cmd
	for _, k := range msgs {
		var cmd tea.Cmd
		m.ta, cmd = m.ta.Update(k)
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

// normalKey handles one key in normal mode. An unrecognised key is swallowed
// rather than inserted, which is what makes normal mode safe to sit in.
func (m *Model) normalKey(msg tea.KeyMsg) tea.Cmd {
	if m.pending != 0 {
		return m.operator(msg.String())
	}

	switch msg.String() {
	// Entering insert.
	case "i":
		m.mode = modeInsert
	case "a":
		m.mode = modeInsert
		return m.feed(plain(tea.KeyRight))
	case "I":
		m.mode = modeInsert
		return m.feed(plain(tea.KeyHome))
	case "A":
		m.mode = modeInsert
		return m.feed(plain(tea.KeyEnd))
	case "o":
		m.mode = modeInsert
		cmd := m.feed(plain(tea.KeyEnd))
		m.ta.InsertString("\n")
		return cmd
	case "O":
		m.mode = modeInsert
		cmd := m.feed(plain(tea.KeyHome))
		m.ta.InsertString("\n")
		m.ta.CursorUp()
		return cmd

	// Motions.
	case "h":
		return m.feed(plain(tea.KeyLeft))
	case "l":
		return m.feed(plain(tea.KeyRight))
	case "j":
		return m.feed(plain(tea.KeyDown))
	case "k":
		return m.feed(plain(tea.KeyUp))
	case "w", "e":
		return m.feed(alt(tea.KeyRight))
	case "b":
		return m.feed(alt(tea.KeyLeft))
	case "0", "^":
		return m.feed(plain(tea.KeyHome))
	case "$":
		return m.feed(plain(tea.KeyEnd))
	case "G":
		return m.feed(altRune('>'))

	// Single-key edits.
	case "x":
		return m.feed(plain(tea.KeyDelete))
	case "X":
		return m.feed(plain(tea.KeyBackspace))
	case "D":
		return m.feed(plain(tea.KeyCtrlK))
	case "C":
		m.mode = modeInsert
		return m.feed(plain(tea.KeyCtrlK))
	case "s":
		m.mode = modeInsert
		return m.feed(plain(tea.KeyDelete))
	case "S":
		m.mode = modeInsert
		return m.feed(plain(tea.KeyHome), plain(tea.KeyCtrlK))

	// Operator-pending.
	case "d", "c", "g":
		m.pending = rune(msg.String()[0])

	// Hand the buffer to a real editor.
	case "v":
		return m.openEditor()
	}
	return nil
}

// operator completes a two-key sequence. pendingInner marks the `i` of `ciw`
// and `diw`, so the operator survives one more keystroke.
func (m *Model) operator(s string) tea.Cmd {
	op := m.pending
	m.pending = 0

	switch op {
	case 'g':
		if s == "g" {
			return m.feed(altRune('<'))
		}

	case 'd':
		switch s {
		case "w":
			return m.feed(alt(tea.KeyDelete))
		case "b":
			return m.feed(alt(tea.KeyBackspace))
		case "d":
			return m.feed(plain(tea.KeyHome), plain(tea.KeyCtrlK))
		case "$":
			return m.feed(plain(tea.KeyCtrlK))
		case "0":
			return m.feed(plain(tea.KeyCtrlU))
		case "i", "a":
			m.pending = pendingDeleteInner
		}

	case 'c':
		switch s {
		case "w":
			m.mode = modeInsert
			return m.feed(alt(tea.KeyDelete))
		case "b":
			m.mode = modeInsert
			return m.feed(alt(tea.KeyBackspace))
		case "c":
			m.mode = modeInsert
			return m.feed(plain(tea.KeyHome), plain(tea.KeyCtrlK))
		case "$":
			m.mode = modeInsert
			return m.feed(plain(tea.KeyCtrlK))
		case "i", "a":
			m.pending = pendingChangeInner
		}

	// The word under the cursor is reached by stepping back to its start and
	// deleting forward; there is no text-object primitive in the widget.
	case pendingDeleteInner:
		if s == "w" {
			return m.feed(alt(tea.KeyLeft), alt(tea.KeyDelete))
		}
	case pendingChangeInner:
		if s == "w" {
			m.mode = modeInsert
			return m.feed(alt(tea.KeyLeft), alt(tea.KeyDelete))
		}
	}
	return nil
}

// Sentinels for the second half of ciw/diw. They are outside the ASCII range
// used by real operator keys so they cannot collide with one.
const (
	pendingDeleteInner rune = -1
	pendingChangeInner rune = -2
)

// openEditor suspends the TUI and opens the prompt in $EDITOR. The buffer is
// round-tripped through a temp file because that is the only contract every
// editor honours.
func (m *Model) openEditor() tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	f, err := os.CreateTemp("", "dwm-agent-prompt-*.md")
	if err != nil {
		return func() tea.Msg { return editorDone{err: err} }
	}
	path := f.Name()
	_, werr := f.WriteString(m.ta.Value())
	cerr := f.Close()
	if werr != nil || cerr != nil {
		os.Remove(path)
		if werr == nil {
			werr = cerr
		}
		return func() tea.Msg { return editorDone{err: werr} }
	}

	return tea.ExecProcess(exec.Command(editor, editorArgs(editor, path)...), func(err error) tea.Msg {
		return editorDone{path: path, err: err}
	})
}

// editorArgs builds the editor's argument list.
//
// For the vi family it prepends `--cmd 'set t_RB= t_RF='`, which clears the
// termcap entries vim uses to query the terminal's background (OSC 11) and
// foreground (OSC 10) colours. Without it the terminal's reply arrives on stdin
// after the editor exits and Bubble Tea reads it as input -- which, now that the
// editor is bound to a control key, can re-fire the binding and reopen the
// editor immediately. --cmd runs before the user's config, so it cannot be
// re-enabled behind our back, and the rest of the user's nvim setup still loads
// normally.
//
// Borrowed from icarus (internal/cli/repl/commands.go:editorArgs), which hit
// exactly this problem first.
func editorArgs(editor, path string) []string {
	base := strings.ToLower(filepath.Base(editor))
	base = strings.TrimSuffix(base, ".exe")
	switch base {
	case "nvim", "vim", "vi":
		return []string{"--cmd", "set t_RB= t_RF=", path}
	}
	return []string{path}
}
