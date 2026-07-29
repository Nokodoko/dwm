// Command monty is a general-purpose scratchpad assistant that can also manage
// the X session through dwm's IPC socket.
//
// It is launched as the ai-scratchpad (Alt+a), which dwm floats, pins to its
// own tag so it persists across tag switches, and toggles closed on the same key.
// The window class stays "ai-scratchpad" regardless of the binary name -- dwm's
// rule and AI_SCRATCHPAD_TAG match on the class, not the command.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nokodoko/dwm-agent/internal/agentloop"
	"github.com/Nokodoko/dwm-agent/internal/dwmipc"
	"github.com/Nokodoko/dwm-agent/internal/llm"
	"github.com/Nokodoko/dwm-agent/internal/skills"
	"github.com/Nokodoko/dwm-agent/internal/tui"
)

func main() {
	var (
		socket  = flag.String("socket", "", "dwm IPC socket (default $DWM_SOCKET or /tmp/dwm.sock)")
		backend = flag.String("model", "", "backend to start on (default: first reachable)")
		ask     = flag.String("ask", "", "run one turn headlessly, print the result, and exit")
	)
	flag.Parse()

	if err := run(*socket, *backend, *ask); err != nil {
		fmt.Fprintln(os.Stderr, "monty:", err)
		// The scratchpad terminal closes the moment this exits, so hold the
		// window open long enough for the error to be readable.
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}
}

func run(socket, backend, ask string) error {
	conn, err := dwmipc.Dial(socket)
	if err != nil {
		return fmt.Errorf("%w\n\nis dwm running with the IPC patch?", err)
	}
	defer conn.Close()

	reg := llm.DefaultRegistry()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var chosen llm.Backend
	if backend != "" {
		b, ok := reg.Get(backend)
		if !ok {
			return fmt.Errorf("unknown backend %q", backend)
		}
		chosen = b
	} else {
		// Prefer the monty backend, fall back to the local model off-network.
		if chosen, err = reg.FirstReachable(ctx, "qwen", "local"); err != nil {
			return err
		}
	}

	client := llm.New(chosen)
	sk := skills.New()

	if ask != "" {
		return runHeadless(conn, client, sk, ask)
	}

	p := tea.NewProgram(tui.New(conn, client, reg, sk), tea.WithAltScreen())
	_, err = p.Run()
	return err
}

// runHeadless executes one turn and prints it, for scripting and smoke tests.
func runHeadless(conn *dwmipc.Conn, client *llm.Client, sk *skills.Set, ask string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	history := []llm.Message{
		{Role: llm.RoleSystem, Content: agentloop.SystemPrompt},
		{Role: llm.RoleUser, Content: ask},
	}

	steps, _, err := agentloop.Run(ctx, client, sk, conn, history)
	for _, s := range steps {
		switch s.Kind {
		case agentloop.KindTool:
			fmt.Println("· " + s.Text)
		case agentloop.KindError:
			fmt.Println("✗ " + s.Text)
		default:
			fmt.Println(s.Text)
		}
	}
	return err
}
