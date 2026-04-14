package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorAccent  = lipgloss.Color("99")  // purple
	colorSuccess = lipgloss.Color("78")  // green
	colorWarning = lipgloss.Color("214") // orange
	colorError   = lipgloss.Color("196") // red
	colorMuted   = lipgloss.Color("241") // gray
	colorInfo    = lipgloss.Color("39")  // blue
	colorBorder  = lipgloss.Color("237") // dark gray

	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	styleMuted = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleSuccess = lipgloss.NewStyle().
			Foreground(colorSuccess)

	styleWarning = lipgloss.NewStyle().
			Foreground(colorWarning)

	styleError = lipgloss.NewStyle().
			Foreground(colorError)

	styleInfo = lipgloss.NewStyle().
			Foreground(colorInfo)

	styleCursor = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	styleCurrent = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSuccess)

	styleColHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorMuted).
			Underline(true)

	styleModal = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(1, 2)

	styleHelp = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleNote = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	styleDetailLabel = lipgloss.NewStyle().
				Foreground(colorMuted).
				Width(10)

	styleDetailVal = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	styleDetailBorder = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(colorBorder).
				PaddingLeft(1)
)

// Color palette for worktree tags.
var colorPalette = []struct {
	Name string
	Dot  string
	Ansi lipgloss.Color
}{
	{"red", "🔴", lipgloss.Color("196")},
	{"orange", "🟠", lipgloss.Color("214")},
	{"yellow", "🟡", lipgloss.Color("226")},
	{"green", "🟢", lipgloss.Color("78")},
	{"blue", "🔵", lipgloss.Color("39")},
	{"purple", "🟣", lipgloss.Color("99")},
	{"white", "⚪", lipgloss.Color("252")},
	{"none", "  ", lipgloss.Color("241")},
}

var iconPalette = []string{
	"🔥", "🐛", "🧪", "🚀", "🔧", "📦",
	"💡", "🔒", "⚡", "🎨", "📝", "✨", "—",
}

func colorDot(name string) string {
	for _, c := range colorPalette {
		if c.Name == name {
			return c.Dot
		}
	}
	return "  "
}
