package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// bigToolOutput approximates what `!ls` on a large directory pushes into the
// transcript: one roleTool entry holding thousands of lines.
func bigToolOutput(lines int) string {
	var sb strings.Builder
	for range lines {
		sb.WriteString("drwxr-xr-x 1 n0ko n0ko 4096 Jul 31 17:00 some-directory-entry-")
		sb.WriteString(strings.Repeat("x", 20))
		sb.WriteString("\n")
	}
	return sb.String()
}

func sizedModel(w, h int) Model {
	m := newTestModel()
	next, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return next.(Model)
}

// The frame must never be taller than the terminal. A View() that overflows
// makes the alt-screen renderer scroll, which is indistinguishable from a
// frozen or blank client.
func TestViewNeverExceedsTerminalHeight(t *testing.T) {
	for _, h := range []int{24, 40, 50, 80} {
		m := sizedModel(200, h)
		m.push(roleTool, bigToolOutput(5000))
		m.renderTranscript()

		got := strings.Count(m.View(), "\n") + 1
		if got > h {
			t.Errorf("height %d: View() rendered %d lines (%d too many)", h, got, got-h)
		}
	}
}

// After Ctrl+L the frame must still be well-formed and the viewport must be
// back at the top -- not parked past the end of now-empty content.
func TestCtrlLResetsFrame(t *testing.T) {
	m := sizedModel(200, 50)
	m.push(roleTool, bigToolOutput(5000))
	m.renderTranscript()
	m.vp.GotoBottom()

	if m.vp.YOffset == 0 {
		t.Fatal("precondition: viewport should be scrolled down")
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
	m = next.(Model)

	if len(m.transcript) != 0 {
		t.Errorf("transcript not cleared: %d entries", len(m.transcript))
	}
	if m.vp.YOffset != 0 {
		t.Errorf("viewport left at YOffset=%d, want 0", m.vp.YOffset)
	}
	got := strings.Count(m.View(), "\n") + 1
	if got > 50 {
		t.Errorf("after clear, View() rendered %d lines, want <= 50", got)
	}
}

func BenchmarkRenderTranscriptBigTool(b *testing.B) {
	m := sizedModel(200, 50)
	m.push(roleTool, bigToolOutput(5000))
	for b.Loop() {
		m.renderTranscript()
	}
}

func BenchmarkViewBigTool(b *testing.B) {
	m := sizedModel(200, 50)
	m.push(roleTool, bigToolOutput(5000))
	m.renderTranscript()
	for b.Loop() {
		_ = m.View()
	}
}
