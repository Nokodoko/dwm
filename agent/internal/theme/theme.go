// Package theme holds the dwm-agent colour palette and lipgloss styles.
//
// The palette is Tokyo Night Storm, chosen to match the icarus web UI so the
// two surfaces read as one system.
package theme

import "github.com/charmbracelet/lipgloss"

// Tokyo Night Storm.
const (
	Bg        = "#24283b"
	BgDark    = "#1f2335"
	BgHigh    = "#292e42"
	Fg        = "#c0caf5"
	FgDim     = "#a9b1d6"
	Comment   = "#565f89"
	Border    = "#3b4261"
	Blue      = "#7aa2f7"
	Cyan      = "#7dcfff"
	Green     = "#9ece6a"
	Magenta   = "#bb9af7"
	Red       = "#f7768e"
	Yellow    = "#e0af68"
	Orange    = "#ff9e64"
	Teal      = "#1abc9c"
)

// Styles is the resolved style set for one terminal width.
type Styles struct {
	App        lipgloss.Style
	Header     lipgloss.Style
	HeaderKey  lipgloss.Style
	HeaderDim  lipgloss.Style
	Viewport   lipgloss.Style
	UserLabel  lipgloss.Style
	AgentLabel lipgloss.Style
	SysLabel   lipgloss.Style
	UserText   lipgloss.Style
	Tool       lipgloss.Style
	ToolOK     lipgloss.Style
	ToolErr    lipgloss.Style
	Prompt     lipgloss.Style
	Input      lipgloss.Style
	Status     lipgloss.Style
	StatusOK   lipgloss.Style
	StatusWarn lipgloss.Style
	Spinner    lipgloss.Style
	Err        lipgloss.Style
	ModeNormal lipgloss.Style
	ModeInsert lipgloss.Style
}

// c converts a hex string to a lipgloss colour. lipgloss.Color is a type, not
// a function, so it cannot be bound to a variable directly.
func c(hex string) lipgloss.Color { return lipgloss.Color(hex) }

// New builds the style set.
func New() Styles {
	return Styles{
		App: lipgloss.NewStyle().
			Background(c(BgDark)),

		Header: lipgloss.NewStyle().
			Foreground(c(Bg)).Background(c(Magenta)).
			Bold(true).Padding(0, 1),

		HeaderKey: lipgloss.NewStyle().
			Foreground(c(Cyan)).Background(c(BgHigh)).Padding(0, 1),

		HeaderDim: lipgloss.NewStyle().
			Foreground(c(Comment)).Background(c(BgHigh)).Padding(0, 1),

		Viewport: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(c(Border)).
			Padding(0, 1),

		UserLabel: lipgloss.NewStyle().
			Foreground(c(Green)).Bold(true),

		AgentLabel: lipgloss.NewStyle().
			Foreground(c(Magenta)).Bold(true),

		SysLabel: lipgloss.NewStyle().
			Foreground(c(Comment)).Italic(true),

		UserText: lipgloss.NewStyle().
			Foreground(c(Fg)),

		Tool: lipgloss.NewStyle().
			Foreground(c(Cyan)).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(c(Blue)).
			PaddingLeft(1),

		ToolOK:  lipgloss.NewStyle().Foreground(c(Green)),
		ToolErr: lipgloss.NewStyle().Foreground(c(Red)),

		Prompt: lipgloss.NewStyle().
			Foreground(c(Magenta)).Bold(true),

		Input: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(c(Blue)).
			Padding(0, 1),

		Status: lipgloss.NewStyle().
			Foreground(c(Comment)),

		StatusOK: lipgloss.NewStyle().
			Foreground(c(Green)),

		StatusWarn: lipgloss.NewStyle().
			Foreground(c(Yellow)),

		Spinner: lipgloss.NewStyle().
			Foreground(c(Magenta)),

		Err: lipgloss.NewStyle().
			Foreground(c(Red)).Bold(true),

		// The mode badges sit where a vim statusline would put them. Blue for
		// normal and green for insert matches the palette's existing use of
		// green as the "live, accepting input" colour (see StatusOK).
		ModeNormal: lipgloss.NewStyle().
			Foreground(c(Bg)).Background(c(Blue)).
			Bold(true).Padding(0, 1),

		ModeInsert: lipgloss.NewStyle().
			Foreground(c(Bg)).Background(c(Green)).
			Bold(true).Padding(0, 1),
	}
}
