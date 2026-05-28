package main

import (
	"fmt"
	"log"
	"os"

	"github.com/alecthomas/kong"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/swtsn/investor/internal/tui"
	"github.com/swtsn/investor/internal/tui/client"
)

func main() {
	var cli struct {
		Addr string `env:"INVESTOR_ADDR" default:"apollo:10001" help:"Server address"`
	}
	kong.Parse(&cli)

	c, err := client.New(cli.Addr)
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
