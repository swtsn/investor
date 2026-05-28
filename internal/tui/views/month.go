package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/swtsn/investor/internal/tui/client"
)

type monthLoadedMsg struct {
	summary client.MonthSummary
	err     error
}

// MonthView shows budget events, contributions, and deployments for a selected month.
type MonthView struct {
	client  client.Client
	year    int
	month   int
	summary client.MonthSummary
	loading bool
	err     error
	width   int
	height  int
}

func NewMonthView(c client.Client) MonthView {
	now := time.Now()
	return MonthView{client: c, year: now.Year(), month: int(now.Month()), loading: true}
}

func (v MonthView) InputActive() bool { return false }

func (v MonthView) Update(msg tea.Msg, _ SharedState) (MonthView, tea.Cmd) {
	switch msg := msg.(type) {
	case LoadMsg:
		v.loading = true
		v.err = nil
		return v, v.load()

	case monthLoadedMsg:
		v.loading = false
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		v.summary = msg.summary
		return v, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "[":
			v.month--
			if v.month < 1 {
				v.month = 12
				v.year--
			}
			v.loading = true
			v.err = nil
			return v, v.load()
		case "]":
			v.month++
			if v.month > 12 {
				v.month = 1
				v.year++
			}
			v.loading = true
			v.err = nil
			return v, v.load()
		case "esc":
			return v, func() tea.Msg { return BackMsg{} }
		}
	}
	return v, nil
}

func (v MonthView) View() string {
	header := fmt.Sprintf("%s — %s %d",
		titleStyle.Render("Month Navigator"),
		time.Month(v.month).String(), v.year)
	nav := dimStyle.Render("[  prev month    ]  next month    Esc back")

	if v.loading {
		return strings.Join([]string{header, nav, "", "Loading..."}, "\n")
	}
	if v.err != nil {
		return strings.Join([]string{header, nav, "", errorStyle.Render(fmt.Sprintf("Error: %v", v.err))}, "\n")
	}

	lines := []string{header, nav, ""}

	if len(v.summary.BudgetEvents) == 0 && len(v.summary.Buckets) == 0 {
		lines = append(lines, dimStyle.Render("No events this month."))
		return strings.Join(lines, "\n")
	}

	for _, e := range v.summary.BudgetEvents {
		lines = append(lines, fmt.Sprintf("Budget event  %s  total %s",
			e.Date.Format("2006-01-02"), formatCurrency(e.TotalAmount)))
	}
	if len(v.summary.BudgetEvents) > 0 {
		lines = append(lines, "")
	}

	for _, bm := range v.summary.Buckets {
		if len(bm.Contributions) == 0 && len(bm.Deployments) == 0 {
			continue
		}
		lines = append(lines, titleStyle.Render(bm.Bucket.Name))
		for _, c := range bm.Contributions {
			lines = append(lines, fmt.Sprintf("  contrib  %s  %s  %s",
				c.Date.Format("2006-01-02"),
				string(c.Origination),
				formatCurrency(c.Amount)))
		}
		for _, d := range bm.Deployments {
			sym := "—"
			if d.Symbol != nil {
				sym = *d.Symbol
			}
			lines = append(lines, fmt.Sprintf("  deploy   %s  %s  %s",
				d.Date.Format("2006-01-02"), sym, formatCurrency(d.Amount)))
		}
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func (v *MonthView) Resize(w, h int) { v.width = w; v.height = h }

func (v MonthView) load() tea.Cmd {
	c := v.client
	year, month := v.year, v.month
	return func() tea.Msg {
		summary, err := c.GetMonthSummary(context.Background(), year, month)
		return monthLoadedMsg{summary: summary, err: err}
	}
}
