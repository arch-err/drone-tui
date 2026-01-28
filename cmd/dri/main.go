package main

import (
	"fmt"
	"os"

	"github.com/arch-err/dri/internal/client"
	"github.com/arch-err/dri/internal/config"
	"github.com/arch-err/dri/internal/tui"
	"github.com/arch-err/dri/internal/version"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("dri %s\n", version.Version)
		os.Exit(0)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	c := client.New(cfg.Server, cfg.Token)
	m := tui.New(c)

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
