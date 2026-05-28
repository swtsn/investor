package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/swtsn/investor/internal/domain"
	"github.com/swtsn/investor/internal/tui/client"
	"github.com/swtsn/investor/internal/tui/views"
)

type bucketsLoadedMsg struct {
	buckets []domain.Bucket
	err     error
}

type mode int

const (
	modeDashboard mode = iota
	modeMonth
	modeBudget
	modeDeploy
	modeReinvest
)

// App is the root bubbletea model.
type App struct {
	state     views.SharedState
	client    client.Client
	mode      mode
	dashboard views.DashboardView
	month     views.MonthView
	budget    views.BudgetView
	deploy    views.DeployView
	reinvest  views.ReinvestView
}

func New(c client.Client) App {
	return App{
		client:   c,
		mode:     modeDashboard,
		dashboard: views.NewDashboardView(c),
		month:     views.NewMonthView(c),
		budget:    views.NewBudgetView(c),
		deploy:    views.NewDeployView(c),
		reinvest:  views.NewReinvestView(c),
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return views.LoadMsg{State: a.state} },
		a.fetchBuckets(),
	)
}

func (a App) fetchBuckets() tea.Cmd {
	c := a.client
	return func() tea.Msg {
		buckets, err := c.ListBuckets(context.Background())
		return bucketsLoadedMsg{buckets: buckets, err: err}
	}
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.state.Width = msg.Width
		a.state.Height = msg.Height - 2 // reserve help bar
		a.propagateSize()
		return a, nil

	case tea.KeyMsg:
		if !a.activeInputActive() {
			switch msg.String() {
			case "q", "ctrl+c":
				return a, tea.Quit
			case "b":
				return a.switchTo(modeBudget)
			case "d":
				return a.switchTo(modeDeploy)
			case "r":
				return a.switchTo(modeReinvest)
			case "m":
				return a.switchTo(modeMonth)
			}
		}

	case views.BackMsg:
		return a.switchTo(modeDashboard)

	case bucketsLoadedMsg:
		if msg.err == nil {
			a.state.Buckets = msg.buckets
		}
		return a, nil
	}

	return a.routeToActive(msg)
}

func (a App) View() string {
	help := "[b]udget  [d]eploy  [r]einvest  [m]onth  [q]uit"
	return fmt.Sprintf("%s\n\n%s", a.activeView(), help)
}

func (a App) activeView() string {
	switch a.mode {
	case modeMonth:
		return a.month.View()
	case modeBudget:
		return a.budget.View()
	case modeDeploy:
		return a.deploy.View()
	case modeReinvest:
		return a.reinvest.View()
	default:
		return a.dashboard.View()
	}
}

func (a App) activeInputActive() bool {
	switch a.mode {
	case modeMonth:
		return a.month.InputActive()
	case modeBudget:
		return a.budget.InputActive()
	case modeDeploy:
		return a.deploy.InputActive()
	case modeReinvest:
		return a.reinvest.InputActive()
	default:
		return a.dashboard.InputActive()
	}
}

func (a App) switchTo(m mode) (tea.Model, tea.Cmd) {
	a.mode = m
	loadMsg := views.LoadMsg{State: a.state}
	var cmd tea.Cmd

	switch m {
	case modeMonth:
		a.month, cmd = a.month.Update(loadMsg, a.state)
	case modeBudget:
		a.budget, cmd = a.budget.Update(loadMsg, a.state)
	case modeDeploy:
		a.deploy, cmd = a.deploy.Update(loadMsg, a.state)
	case modeReinvest:
		a.reinvest, cmd = a.reinvest.Update(loadMsg, a.state)
	default:
		a.dashboard, cmd = a.dashboard.Update(loadMsg, a.state)
	}
	return a, cmd
}

func (a App) routeToActive(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch a.mode {
	case modeMonth:
		a.month, cmd = a.month.Update(msg, a.state)
	case modeBudget:
		a.budget, cmd = a.budget.Update(msg, a.state)
	case modeDeploy:
		a.deploy, cmd = a.deploy.Update(msg, a.state)
	case modeReinvest:
		a.reinvest, cmd = a.reinvest.Update(msg, a.state)
	default:
		a.dashboard, cmd = a.dashboard.Update(msg, a.state)
	}
	return a, cmd
}

func (a *App) propagateSize() {
	a.dashboard.Resize(a.state.Width, a.state.Height)
	a.month.Resize(a.state.Width, a.state.Height)
	a.budget.Resize(a.state.Width, a.state.Height)
	a.deploy.Resize(a.state.Width, a.state.Height)
	a.reinvest.Resize(a.state.Width, a.state.Height)
}
