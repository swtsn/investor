package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shopspring/decimal"

	"github.com/swtsn/investor/internal/domain"
	"github.com/swtsn/investor/internal/tui/client"
)

var planColors = []lipgloss.Color{"4", "2", "3", "5", "6", "1"}

type planDashboardLoadedMsg struct {
	data []client.BucketDashboard
	err  error
}

type planPreviewMsg struct {
	total       decimal.Decimal
	splits      []client.BucketAllocation
	allocSplits map[int64][]client.AllocationSplit
	err         error
}

// PlanView is a read-only view for planning a deployment amount across buckets.
type PlanView struct {
	client      client.Client
	amountInput textinput.Model

	dashboard   []client.BucketDashboard
	loadingDash bool
	dashErr     error

	total        decimal.Decimal
	bucketSplits []client.BucketAllocation
	allocSplits  map[int64][]client.AllocationSplit
	loadingPlan  bool
	planErr      error

	width  int
	height int
}

func NewPlanView(c client.Client) PlanView {
	ti := textinput.New()
	ti.Placeholder = "0.00"
	ti.CharLimit = 20
	return PlanView{client: c, amountInput: ti}
}

func (v PlanView) InputActive() bool { return !v.loadingPlan }

func (v PlanView) Update(msg tea.Msg, _ SharedState) (PlanView, tea.Cmd) {
	switch msg := msg.(type) {
	case LoadMsg:
		v.loadingDash = true
		v.dashErr = nil
		v.bucketSplits = nil
		v.allocSplits = nil
		v.planErr = nil
		v.amountInput.Focus()
		return v, tea.Batch(v.loadDashboard(), textinput.Blink)

	case planDashboardLoadedMsg:
		v.loadingDash = false
		if msg.err != nil {
			v.dashErr = msg.err
		} else {
			v.dashboard = msg.data
		}
		return v, nil

	case planPreviewMsg:
		v.loadingPlan = false
		v.amountInput.Focus()
		if msg.err != nil {
			v.planErr = msg.err
		} else {
			v.total = msg.total
			v.bucketSplits = msg.splits
			v.allocSplits = msg.allocSplits
			v.planErr = nil
		}
		return v, textinput.Blink

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			raw := strings.TrimSpace(v.amountInput.Value())
			amount, err := decimal.NewFromString(raw)
			if err != nil || !amount.IsPositive() {
				v.planErr = fmt.Errorf("enter a positive amount")
				return v, nil
			}
			v.loadingPlan = true
			v.bucketSplits = nil
			v.allocSplits = nil
			v.planErr = nil
			v.amountInput.Blur()
			return v, v.loadPreview(amount)
		case "esc":
			v.amountInput.Blur()
			return v, func() tea.Msg { return BackMsg{} }
		default:
			var cmd tea.Cmd
			v.amountInput, cmd = v.amountInput.Update(msg)
			return v, cmd
		}
	}
	return v, nil
}

func (v PlanView) loadDashboard() tea.Cmd {
	c := v.client
	return func() tea.Msg {
		data, err := c.GetDashboard(context.Background())
		return planDashboardLoadedMsg{data: data, err: err}
	}
}

func (v PlanView) loadPreview(total decimal.Decimal) tea.Cmd {
	c := v.client
	return func() tea.Msg {
		ctx := context.Background()
		splits, err := c.PreviewBudget(ctx, total)
		if err != nil {
			return planPreviewMsg{err: err}
		}
		allocSplits := make(map[int64][]client.AllocationSplit)
		for _, ba := range splits {
			if ba.Bucket.Type == domain.BucketTypeDiversified {
				as, err := c.PreviewDeployment(ctx, ba.Bucket.ID, ba.Amount)
				if err != nil {
					return planPreviewMsg{err: err}
				}
				allocSplits[ba.Bucket.ID] = as
			}
		}
		return planPreviewMsg{total: total, splits: splits, allocSplits: allocSplits}
	}
}

func (v *PlanView) Resize(w, h int) {
	v.width = w
	v.height = h
}

const planLeftWidth = 30

func (v PlanView) View() string {
	return lipgloss.JoinHorizontal(lipgloss.Top, v.renderLeft(), v.renderRight())
}

func (v PlanView) renderLeft() string {
	leftStyle := lipgloss.NewStyle().Width(planLeftWidth)

	if v.loadingDash {
		return leftStyle.Render("Loading...")
	}
	if v.dashErr != nil {
		return leftStyle.Render(errorStyle.Render(fmt.Sprintf("Error: %v", v.dashErr)))
	}

	lines := []string{
		titleStyle.Render("Plan"),
		"",
		"Amount to deploy:",
		v.amountInput.View(),
		"",
	}
	if v.planErr != nil {
		lines = append(lines, errorStyle.Render(v.planErr.Error()), "")
	}
	lines = append(lines, dimStyle.Render("Enter to preview  Esc to go back"))

	return leftStyle.Render(strings.Join(lines, "\n"))
}

func (v PlanView) renderRight() string {
	rightWidth := v.width - planLeftWidth
	if rightWidth < 1 {
		rightWidth = 40
	}

	if v.loadingPlan {
		return dimStyle.Render("Loading plan...")
	}

	if len(v.dashboard) == 0 {
		return dimStyle.Render("No buckets configured.")
	}

	splitsByID := make(map[int64]decimal.Decimal)
	typeByID := make(map[int64]domain.BucketType)
	for _, ba := range v.bucketSplits {
		splitsByID[ba.Bucket.ID] = ba.Amount
		typeByID[ba.Bucket.ID] = ba.Bucket.Type
	}

	const cardOverhead = 4 // 1 border + 1 padding per side
	cardWidth := (rightWidth / len(v.dashboard)) - cardOverhead
	if cardWidth < 14 {
		cardWidth = 14
	}

	cards := make([]string, len(v.dashboard))
	for i, bd := range v.dashboard {
		color := planColors[i%len(planColors)]
		cardStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(color).
			Width(cardWidth).
			Padding(0, 1)
		nameStyle := lipgloss.NewStyle().Foreground(color).Bold(true)

		header := fmt.Sprintf("%s  %s%%",
			nameStyle.Render(bd.Bucket.Name),
			bd.Bucket.TargetPct.StringFixed(0))
		pool := fmt.Sprintf("Pool: %s", formatCurrency(bd.PoolBalance))
		divider := strings.Repeat("─", cardWidth-2)

		planned := "Plan: —"
		if amount, ok := splitsByID[bd.Bucket.ID]; ok {
			planned = fmt.Sprintf("Plan: %s", formatCurrency(amount))
		}

		lines := []string{header, pool, divider, planned}

		if typeByID[bd.Bucket.ID] == domain.BucketTypeDiversified {
			if allocs, ok := v.allocSplits[bd.Bucket.ID]; ok && len(allocs) > 0 {
				lines = append(lines, "")
				for _, as := range allocs {
					lines = append(lines, fmt.Sprintf("%-12s %3s%%  %s",
						as.Allocation.Name,
						as.Allocation.TargetPct.StringFixed(0),
						formatCurrency(as.Amount)))
				}
			}
		}

		cards[i] = cardStyle.Render(strings.Join(lines, "\n"))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, cards...)
}
