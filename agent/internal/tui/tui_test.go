package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nokodoko/dwm-agent/internal/llm"
	"github.com/Nokodoko/dwm-agent/internal/skills"
)

// newTestModel builds a Model without touching dwm or a model backend. A nil
// dwmipc.Conn is fine here: these tests exercise input routing, which must
// decide where a line goes before any dispatch happens.
func newTestModel() Model {
	reg := llm.DefaultRegistry()
	b, _ := reg.Get("qwen")
	return New(nil, llm.New(b), reg, skills.New())
}

// submit feeds a line to the model as if the user typed it and pressed enter.
func submit(m Model, line string) (Model, tea.Cmd) {
	m.ta.SetValue(line)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return next.(Model), cmd
}

func lastEntry(m Model) entry {
	if len(m.transcript) == 0 {
		return entry{}
	}
	return m.transcript[len(m.transcript)-1]
}

// A "!" line must run in the shell instead of being sent to the model. The
// regression this guards is silent: without the prefix check the line reaches
// the LLM as prose, and the model answers "I can't run commands".
func TestBangRoutesToShellNotModel(t *testing.T) {
	m, cmd := submit(newTestModel(), "!echo hello")

	if cmd == nil {
		t.Fatal("expected a command to be scheduled, got nil")
	}
	if !m.busy {
		t.Error("expected busy while the shell runs")
	}
	if got := lastEntry(m); !strings.Contains(got.text, "echo hello") {
		t.Errorf("transcript should echo the command, got %q", got.text)
	}
	// A bang line must not enter the model conversation as a user turn. Only
	// the system prompt should be present at this point.
	if len(m.history) != 1 {
		t.Errorf("bang must not append to model history, got %d messages", len(m.history))
	}
}

// "!" with nothing after it should explain itself, not spawn an empty shell.
func TestBareBangIsRejected(t *testing.T) {
	m, cmd := submit(newTestModel(), "!")

	if cmd != nil {
		t.Error("bare ! should not schedule a shell run")
	}
	if m.busy {
		t.Error("bare ! should not mark the model busy")
	}
	if got := lastEntry(m); got.role != roleError {
		t.Errorf("expected an error entry, got role %v (%q)", got.role, got.text)
	}
}

// Slash commands must keep working and must not be treated as shell input.
func TestSlashCommandStillRoutesToCommandHandler(t *testing.T) {
	m, _ := submit(newTestModel(), "/skills")

	if m.busy {
		t.Error("/skills is local and should not mark the model busy")
	}
	if got := lastEntry(m); !strings.Contains(got.text, "shell") {
		t.Errorf("expected the skill list to mention shell, got %q", got.text)
	}
}

// Ordinary prose must still reach the model rather than the shell.
func TestPlainInputGoesToModel(t *testing.T) {
	m, cmd := submit(newTestModel(), "what is the capital of Peru")

	if cmd == nil {
		t.Fatal("expected a model turn to be scheduled")
	}
	if len(m.history) != 2 {
		t.Fatalf("expected system + user message, got %d", len(m.history))
	}
	if m.history[1].Role != llm.RoleUser {
		t.Errorf("expected a user message, got role %q", m.history[1].Role)
	}
}
