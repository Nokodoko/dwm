// Command probe exercises the dwmipc client against a live dwm instance.
// It is a development smoke test, not part of the shipped agent.
package main

import (
	"fmt"
	"os"

	"github.com/Nokodoko/dwm-agent/internal/dwmipc"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "probe:", err)
		os.Exit(1)
	}
}

func run() error {
	conn, err := dwmipc.Dial("")
	if err != nil {
		return err
	}
	defer conn.Close()

	tags, err := conn.GetTags()
	if err != nil {
		return err
	}
	fmt.Printf("tags (%d): ", len(tags))
	for _, t := range tags {
		fmt.Printf("%s(%d) ", t.Name, t.BitMask)
	}
	fmt.Println()

	layouts, err := conn.GetLayouts()
	if err != nil {
		return err
	}
	fmt.Printf("layouts (%d): ", len(layouts))
	for _, l := range layouts {
		fmt.Printf("%q ", l.Symbol)
	}
	fmt.Println()

	mons, err := conn.GetMonitors()
	if err != nil {
		return err
	}
	for _, m := range mons {
		fmt.Printf("\nmonitor %d  selected=%v  %dx%d  layout=%q  mfact=%.2f\n",
			m.Num, m.IsSelected,
			m.MonitorGeometry.Width, m.MonitorGeometry.Height,
			m.Layout.Symbol.Current, m.MasterFactor)
		fmt.Printf("  tags: selected=%d occupied=%d urgent=%d\n",
			m.TagState.Selected, m.TagState.Occupied, m.TagState.Urgent)

		for _, win := range m.Clients.All {
			cl, err := conn.GetClient(win)
			if err != nil {
				fmt.Printf("  client %d: ERROR %v\n", win, err)
				continue
			}
			marker := " "
			if win == m.Clients.Selected {
				marker = "*"
			}
			fmt.Printf("  %s win=%-10d tags=%-4d float=%-5v fs=%-5v %q\n",
				marker, cl.WindowID, cl.Tags,
				cl.States.IsFloating, cl.States.IsFullscreen, cl.Name)
		}
	}
	return nil
}
