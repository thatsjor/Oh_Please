package main

import (
	"fmt"
	"os"

	"opls/config"
	"opls/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	appConfig := config.LoadConfig()

	startPath := "."
	if len(os.Args) > 1 {
		startPath = os.Args[1]
	}

	p := tea.NewProgram(
		ui.NewMainModel(startPath, appConfig),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Oh please error: %v\n", err)
		os.Exit(1)
	}
}
