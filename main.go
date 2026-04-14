package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/doganarif/grove/internal/tui"
)

func main() {
	m, err := tui.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "grove: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "grove: %v\n", err)
		os.Exit(1)
	}

	if m, ok := result.(tui.Model); ok && m.Selected != "" {
		fmt.Print(m.Selected)
	}
}
