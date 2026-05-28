package views

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/swtsn/investor/internal/tui/client"
)

type dashboardLoadedMsg struct {
	data []client.BucketDashboard
	err  error
}

// DashboardView shows pool balances and origination breakdown per bucket.
type DashboardView struct {
	client  client.Client
	table   table.Model
	data    []client.BucketDashboard
	loading bool
	err     error
	width   int
	height  int
}

func NewDashboardView(c client.Client) DashboardView {
	return DashboardView{client: c, loading: true}
}

func (v DashboardView) InputActive() bool { return false }

func (v DashboardView) Update(msg tea.Msg, _ SharedState) (DashboardView, tea.Cmd) {
	switch msg := msg.(type) {
	case LoadMsg:
		v.loading = true
		v.err = nil
		return v, v.load()

	case dashboardLoadedMsg:
		v.loading = false
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		v.data = msg.data
		v.table = buildDashboardTable(msg.data, v.width, v.height)
		return v, nil

	case tea.KeyMsg:
		if !v.loading && v.err == nil && len(v.data) > 0 {
			var cmd tea.Cmd
			v.table, cmd = v.table.Update(msg)
			return v, cmd
		}
	}
	return v, nil
}

func (v DashboardView) View() string {
	if v.loading {
		return "Loading dashboard..."
	}
	if v.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	if len(v.data) == 0 {
		return dimStyle.Render("No buckets configured.")
	}
	return tableStyle.Render(v.table.View())
}

func (v *DashboardView) Resize(w, h int) {
	v.width = w
	v.height = h
	if len(v.data) > 0 {
		cursor := v.table.Cursor()
		v.table = buildDashboardTable(v.data, w, h)
		v.table.SetCursor(cursor)
	}
}

func (v DashboardView) load() tea.Cmd {
	c := v.client
	return func() tea.Msg {
		data, err := c.GetDashboard(context.Background())
		return dashboardLoadedMsg{data: data, err: err}
	}
}

func buildDashboardTable(data []client.BucketDashboard, w, h int) table.Model {
	_ = w
	cols := []table.Column{
		{Title: "Bucket", Width: 12},
		{Title: "Pool", Width: 10},
		{Title: "Budget", Width: 10},
		{Title: "Reinvest", Width: 10},
		{Title: "Slush", Width: 10},
		{Title: "Deployed", Width: 10},
		{Title: "Actual (Target)", Width: 15},
	}

	rows := make([]table.Row, len(data))
	for i, d := range data {
		alloc := fmt.Sprintf("%s%% (%s%%)",
			d.ActualPct.StringFixed(1),
			d.Bucket.TargetPct.StringFixed(1))
		rows[i] = table.Row{
			d.Bucket.Name,
			formatCurrency(d.PoolBalance),
			formatCurrency(d.Budget),
			formatCurrency(d.Reinvestment),
			formatCurrency(d.Slush),
			formatCurrency(d.DeployedMonth),
			alloc,
		}
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(h-2),
	)
	t.SetStyles(defaultTableStyles())
	return t
}
