package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/shopspring/decimal"

	"github.com/swtsn/investor/internal/tui/client"
)

type budgetStep int

const (
	budgetStepAmountEntry budgetStep = iota
	budgetStepPreview
	budgetStepConfirm
	budgetStepSuccess
)

type budgetPreviewMsg struct {
	allocs []client.BucketAllocation
	err    error
}
type budgetSubmittedMsg struct{ err error }

// BudgetView: enter total → preview splits → confirm date → ApplyBudget.
type BudgetView struct {
	client      client.Client
	step        budgetStep
	amountInput textinput.Model
	dateInput   textinput.Model
	total       decimal.Decimal
	preview     []client.BucketAllocation
	submitErr   error
	loading     bool
	width       int
	height      int
}

func NewBudgetView(c client.Client) BudgetView {
	ai := textinput.New()
	ai.Placeholder = "0.00"
	ai.CharLimit = 20

	di := textinput.New()
	di.CharLimit = 10

	return BudgetView{client: c, amountInput: ai, dateInput: di}
}

func (v BudgetView) InputActive() bool {
	return v.step == budgetStepAmountEntry || v.step == budgetStepConfirm
}

func (v BudgetView) Update(msg tea.Msg, _ SharedState) (BudgetView, tea.Cmd) {
	switch msg := msg.(type) {
	case LoadMsg:
		v.step = budgetStepAmountEntry
		v.submitErr = nil
		v.amountInput.SetValue("")
		v.amountInput.Focus()
		return v, textinput.Blink

	case budgetPreviewMsg:
		v.loading = false
		if msg.err != nil {
			v.submitErr = msg.err
			v.step = budgetStepAmountEntry
			v.amountInput.Focus()
			return v, textinput.Blink
		}
		v.preview = msg.allocs
		v.step = budgetStepPreview
		return v, nil

	case budgetSubmittedMsg:
		if msg.err != nil {
			v.submitErr = msg.err
			return v, nil
		}
		v.step = budgetStepSuccess
		return v, nil

	case tea.KeyMsg:
		return v.handleKey(msg)
	}
	return v, nil
}

func (v BudgetView) handleKey(msg tea.KeyMsg) (BudgetView, tea.Cmd) {
	switch v.step {
	case budgetStepAmountEntry:
		switch msg.String() {
		case "enter":
			a, err := decimal.NewFromString(v.amountInput.Value())
			if err != nil || !a.IsPositive() {
				v.submitErr = fmt.Errorf("enter a positive amount")
				return v, nil
			}
			v.total = a
			v.submitErr = nil
			v.loading = true
			v.amountInput.Blur()
			return v, v.loadPreview()
		case "esc":
			v.amountInput.Blur()
			return v, func() tea.Msg { return BackMsg{} }
		default:
			var cmd tea.Cmd
			v.amountInput, cmd = v.amountInput.Update(msg)
			return v, cmd
		}

	case budgetStepPreview:
		switch msg.String() {
		case "enter":
			v.step = budgetStepConfirm
			v.submitErr = nil
			v.dateInput.SetValue(time.Now().Format("2006-01-02"))
			v.dateInput.Focus()
			return v, textinput.Blink
		case "esc":
			v.step = budgetStepAmountEntry
			v.amountInput.SetValue(v.total.String())
			v.amountInput.Focus()
			return v, textinput.Blink
		}

	case budgetStepConfirm:
		switch msg.String() {
		case "enter":
			d, err := time.Parse("2006-01-02", v.dateInput.Value())
			if err != nil {
				v.submitErr = fmt.Errorf("invalid date (use YYYY-MM-DD)")
				return v, nil
			}
			v.submitErr = nil
			v.dateInput.Blur()
			return v, v.submit(d)
		case "esc":
			v.dateInput.Blur()
			v.step = budgetStepPreview
			v.submitErr = nil
		default:
			var cmd tea.Cmd
			v.dateInput, cmd = v.dateInput.Update(msg)
			return v, cmd
		}

	case budgetStepSuccess:
		if msg.String() == "esc" || msg.String() == "enter" {
			return v, func() tea.Msg { return BackMsg{} }
		}
	}
	return v, nil
}

func (v BudgetView) View() string {
	switch v.step {
	case budgetStepAmountEntry:
		lines := []string{
			titleStyle.Render("Budget — Enter Total"),
			"",
			"Monthly budget amount:",
			v.amountInput.View(),
		}
		if v.submitErr != nil {
			lines = append(lines, "", errorStyle.Render(v.submitErr.Error()))
		}
		lines = append(lines, "", dimStyle.Render("Enter to preview  Esc to go back"))
		return strings.Join(lines, "\n")

	case budgetStepPreview:
		if v.loading {
			return "Loading preview..."
		}
		lines := []string{
			titleStyle.Render("Budget — Preview"),
			"",
			fmt.Sprintf("Total: %s", formatCurrency(v.total)),
			"",
			fmt.Sprintf("  %-14s  %-8s  %s", "Bucket", "Target%", "Amount"),
			fmt.Sprintf("  %-14s  %-8s  %s", strings.Repeat("─", 14), strings.Repeat("─", 7), strings.Repeat("─", 10)),
		}
		for _, a := range v.preview {
			lines = append(lines, fmt.Sprintf("  %-14s  %-8s  %s",
				a.Bucket.Name,
				a.Bucket.TargetPct.StringFixed(1)+"%",
				formatCurrency(a.Amount)))
		}
		lines = append(lines, "", dimStyle.Render("Enter to confirm  Esc to go back"))
		return strings.Join(lines, "\n")

	case budgetStepConfirm:
		lines := []string{
			titleStyle.Render("Budget — Confirm"),
			"",
			fmt.Sprintf("  Total: %s", formatCurrency(v.total)),
			"",
			"Date (YYYY-MM-DD):",
			v.dateInput.View(),
		}
		if v.submitErr != nil {
			lines = append(lines, "", errorStyle.Render(v.submitErr.Error()))
		}
		lines = append(lines, "", dimStyle.Render("Enter to apply  Esc to go back"))
		return strings.Join(lines, "\n")

	case budgetStepSuccess:
		return strings.Join([]string{
			titleStyle.Render("Budget applied."),
			"",
			fmt.Sprintf("  Total: %s applied across %d bucket(s).", formatCurrency(v.total), len(v.preview)),
			"",
			dimStyle.Render("Enter or Esc to return to dashboard"),
		}, "\n")
	}
	return ""
}

func (v *BudgetView) Resize(w, h int) { v.width = w; v.height = h }

func (v BudgetView) loadPreview() tea.Cmd {
	c := v.client
	total := v.total
	return func() tea.Msg {
		allocs, err := c.PreviewBudget(context.Background(), total)
		return budgetPreviewMsg{allocs: allocs, err: err}
	}
}

func (v BudgetView) submit(date time.Time) tea.Cmd {
	c := v.client
	total := v.total
	return func() tea.Msg {
		err := c.ApplyBudget(context.Background(), total, date)
		return budgetSubmittedMsg{err: err}
	}
}
