package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/swtsn/investor/internal/tui"
	"github.com/swtsn/investor/internal/tui/client"
)

func main() {
	addr := os.Getenv("INVESTOR_ADDR")
	if addr == "" {
		addr = "localhost:50051"
	}

	c, err := client.New(addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect: %v\n", err)
		os.Exit(1)
	}
	defer c.Close() //nolint:errcheck

	p := tea.NewProgram(tui.New(c), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
