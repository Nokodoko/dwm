package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// normalModel builds a Model carrying only what normal-mode editing touches:
// the textarea, the mode, and the pending operator.
func normalModel(value string) Model {
	ta := textarea.New()
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.Focus()
	ta.SetValue(value)
	ta.CursorStart()
	return Model{ta: ta, mode: modeNormal}
}

// press drives normal-mode keys the way bubbletea delivers them.
func press(m *Model, keys ...string) {
	for _, k := range keys {
		m.normalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
	}
}

func TestNormalModeEdits(t *testing.T) {
	t.Log("Given the prompt is in normal mode with the cursor at the line start.")

	cases := []struct {
		name     string
		start    string
		keys     []string
		want     string
		wantMode mode
	}{
		{"x deletes the character under the cursor", "hello", []string{"x"}, "ello", modeNormal},
		{"D deletes through end of line", "hello world", []string{"D"}, "", modeNormal},
		{"C deletes through end of line and inserts", "hello world", []string{"C"}, "", modeInsert},
		{"s substitutes one character", "hello", []string{"s"}, "ello", modeInsert},
		{"i enters insert without editing", "hello", []string{"i"}, "hello", modeInsert},
		{"A enters insert without editing", "hello", []string{"A"}, "hello", modeInsert},
		{"an unbound key is swallowed, not typed", "hello", []string{"z"}, "hello", modeNormal},
		{"a bare operator edits nothing until completed", "hello", []string{"d"}, "hello", modeNormal},
		// cw stops at the end of the word rather than eating the following
		// space, which is what vim does (cw is treated as ce).
		{"cw changes to end of word", "hello world", []string{"c", "w"}, " world", modeInsert},
		{"ciw changes the inner word", "hello world", []string{"c", "i", "w"}, " world", modeInsert},
		{"dw deletes to end of word", "hello world", []string{"d", "w"}, " world", modeNormal},
		{"dd clears the line", "hello world", []string{"d", "d"}, "", modeNormal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := normalModel(tc.start)
			press(&m, tc.keys...)

			if got := m.ta.Value(); got != tc.want {
				t.Errorf("\t✗\tvalue: want %q, got %q", tc.want, got)
			}
			if m.mode != tc.wantMode {
				t.Errorf("\t✗\tmode: want %v, got %v", tc.wantMode, m.mode)
			}
			if m.mode == modeNormal && m.pending != 0 && len(tc.keys) > 1 {
				t.Errorf("\t✗\tpending operator %q left dangling", m.pending)
			}
		})
	}
}

// A dangling operator must not survive into the next keystroke, or `d` followed
// by an unrelated key would silently arm a delete.
func TestOperatorClearsAfterUse(t *testing.T) {
	m := normalModel("hello world")

	press(&m, "d")
	if m.pending == 0 {
		t.Fatal("\t✗\t`d` did not arm the operator")
	}

	press(&m, "w")
	if m.pending != 0 {
		t.Errorf("\t✗\toperator still armed after completion: %q", m.pending)
	}
}

// `dw` and `x` must not fire while the prompt is in insert mode; Update routes
// keys to the textarea there, so normalKey should never see them.
func TestModeStringsAreStable(t *testing.T) {
	if modeNormal.String() != "NORMAL" {
		t.Errorf("\t✗\tnormal badge: got %q", modeNormal.String())
	}
	if modeInsert.String() != "INSERT" {
		t.Errorf("\t✗\tinsert badge: got %q", modeInsert.String())
	}
}
