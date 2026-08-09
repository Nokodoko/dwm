package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The vi family must get the OSC-query guard: without it the terminal's
// background-colour reply lands on stdin after the editor exits, Bubble Tea
// reads it as input, and -- now that the editor is on a control key -- it can
// re-fire the binding and reopen the editor immediately.
func TestEditorArgsGuardsViFamily(t *testing.T) {
	for _, ed := range []string{"nvim", "vim", "vi", "/usr/local/bin/nvim", "NVIM", "nvim.exe"} {
		got := editorArgs(ed, "/tmp/p.md")
		if len(got) != 3 {
			t.Errorf("%s: got %v, want 3 args", ed, got)
			continue
		}
		if got[0] != "--cmd" || !strings.Contains(got[1], "t_RB=") || !strings.Contains(got[1], "t_RF=") {
			t.Errorf("%s: guard missing: %v", ed, got)
		}
		if got[2] != "/tmp/p.md" {
			t.Errorf("%s: path must come last, got %v", ed, got)
		}
	}
}

// Anything else takes the path alone -- passing vim's --cmd to, say, emacs or
// nano would just make the editor fail to start.
func TestEditorArgsLeavesOtherEditorsAlone(t *testing.T) {
	for _, ed := range []string{"nano", "emacs", "helix", "/usr/bin/code"} {
		got := editorArgs(ed, "/tmp/p.md")
		if len(got) != 1 || got[0] != "/tmp/p.md" {
			t.Errorf("%s: got %v, want just the path", ed, got)
		}
	}
}

// openEditor seeds the tempfile with the current prompt, which is what makes
// Ctrl+G mid-sentence keep what was already typed.
func TestOpenEditorSeedsTempFileWithPrompt(t *testing.T) {
	m := newTestModel()
	const seed = "half a thought already typed"
	m.ta.SetValue(seed)

	// openEditor returns a tea.ExecProcess command; running it would launch a
	// real editor, so assert on the artefact it leaves behind instead.
	if cmd := m.openEditor(); cmd == nil {
		t.Fatal("openEditor returned no command")
	}

	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "dwm-agent-prompt-*.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatal("no tempfile created")
	}

	var found bool
	for _, p := range matches {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if string(b) == seed {
			found = true
		}
		os.Remove(p)
	}
	if !found {
		t.Errorf("no tempfile seeded with %q", seed)
	}
}
