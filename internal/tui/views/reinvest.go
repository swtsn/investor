package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/shopspring/decimal"

	"github.com/swtsn/investor/internal/domain"
	"github.com/swtsn/investor/internal/tui/client"
)

type reinvestStep int

const (
	reinvestStepBucketSelect reinvestStep = iota
	reinvestStepAmountEntry
	reinvestStepConfirm
	reinvestStepSuccess
)

type reinvestSubmittedMsg struct{ err error }

// ReinvestView: pick bucket → enter amount → confirm → RecordReinvestment.
type ReinvestView struct {
	client client.Client

	step         reinvestStep
	buckets      []domain.Bucket
	bucketCursor int
	bucket       domain.Bucket
	amountInput  textinput.Model
	dateInput    textinput.Model
	amount       decimal.Decimal
	submitErr    error
	width        int
	height       int
}

func NewReinvestView(c client.Client) ReinvestView {
	ai := textinput.New()
	ai.Placeholder = "0.00"
	ai.CharLimit = 20

	di := textinput.New()
	di.CharLimit = 10

	return ReinvestView{client: c, amountInput: ai, dateInput: di}
}

func (v ReinvestView) InputActive() bool {
	return v.step == reinvestStepAmountEntry || v.step == reinvestStepConfirm
}

func (v ReinvestView) Update(msg tea.Msg, state SharedState) (ReinvestView, tea.Cmd) {
	switch msg := msg.(type) {
	case LoadMsg:
		v.step = reinvestStepBucketSelect
		v.buckets = state.Buckets
		v.bucketCursor = 0
		v.submitErr = nil
		v.amountInput.SetValue("")
		return v, nil

	case reinvestSubmittedMsg:
		if msg.err != nil {
			v.submitErr = msg.err
			return v, nil
		}
		v.step = reinvestStepSuccess
		return v, nil

	case tea.KeyMsg:
		return v.handleKey(msg, state)
	}
	return v, nil
}

func (v ReinvestView) handleKey(msg tea.KeyMsg, state SharedState) (ReinvestView, tea.Cmd) {
	_ = state
	switch v.step {
	case reinvestStepBucketSelect:
		switch msg.String() {
		case "up", "k":
			if v.bucketCursor > 0 {
				v.bucketCursor--
			}
		case "down", "j":
			if v.bucketCursor < len(v.buckets)-1 {
				v.bucketCursor++
			}
		case "enter":
			if len(v.buckets) == 0 {
				return v, nil
			}
			v.bucket = v.buckets[v.bucketCursor]
			v.step = reinvestStepAmountEntry
			v.amountInput.SetValue("")
			v.amountInput.Focus()
			return v, textinput.Blink
		case "esc":
			return v, func() tea.Msg { return BackMsg{} }
		}

	case reinvestStepAmountEntry:
		switch msg.String() {
		case "enter":
			a, err := decimal.NewFromString(v.amountInput.Value())
			if err != nil || !a.IsPositive() {
				v.submitErr = fmt.Errorf("enter a positive amount")
				return v, nil
			}
			v.amount = a
			v.submitErr = nil
			v.amountInput.Blur()
			v.step = reinvestStepConfirm
			v.dateInput.SetValue(time.Now().Format("2006-01-02"))
			v.dateInput.Focus()
			return v, textinput.Blink
		case "esc":
			v.amountInput.Blur()
			v.step = reinvestStepBucketSelect
			v.submitErr = nil
		default:
			var cmd tea.Cmd
			v.amountInput, cmd = v.amountInput.Update(msg)
			return v, cmd
		}

	case reinvestStepConfirm:
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
			v.step = reinvestStepAmountEntry
			v.amountInput.SetValue(v.amount.String())
			v.amountInput.Focus()
			return v, textinput.Blink
		default:
			var cmd tea.Cmd
			v.dateInput, cmd = v.dateInput.Update(msg)
			return v, cmd
		}

	case reinvestStepSuccess:
		if msg.String() == "esc" || msg.String() == "enter" {
			return v, func() tea.Msg { return BackMsg{} }
		}
	}
	return v, nil
}

func (v ReinvestView) View() string {
	switch v.step {
	case reinvestStepBucketSelect:
		if len(v.buckets) == 0 {
			return titleStyle.Render("Reinvest — Select Bucket") + "\n\n" +
				dimStyle.Render("No buckets configured.  Esc to go back.")
		}
		lines := []string{titleStyle.Render("Reinvest — Select Bucket"), ""}
		for i, b := range v.buckets {
			cursor := "  "
			if i == v.bucketCursor {
				cursor = "> "
			}
			lines = append(lines, fmt.Sprintf("%s%s", cursor, b.Name))
		}
		lines = append(lines, "", dimStyle.Render("↑↓ navigate  Enter to select  Esc to go back"))
		return strings.Join(lines, "\n")

	case reinvestStepAmountEntry:
		lines := []string{
			titleStyle.Render("Reinvest — Enter Amount"),
			"",
			fmt.Sprintf("Bucket: %s", v.bucket.Name),
			"",
			"Amount:",
			v.amountInput.View(),
		}
		if v.submitErr != nil {
			lines = append(lines, "", errorStyle.Render(v.submitErr.Error()))
		}
		lines = append(lines, "", dimStyle.Render("Enter to continue  Esc to go back"))
		return strings.Join(lines, "\n")

	case reinvestStepConfirm:
		lines := []string{
			titleStyle.Render("Reinvest — Confirm"),
			"",
			fmt.Sprintf("  Bucket: %s", v.bucket.Name),
			fmt.Sprintf("  Amount: %s", formatCurrency(v.amount)),
			"",
			"Date (YYYY-MM-DD):",
			v.dateInput.View(),
		}
		if v.submitErr != nil {
			lines = append(lines, "", errorStyle.Render(v.submitErr.Error()))
		}
		lines = append(lines, "", dimStyle.Render("Enter to confirm  Esc to go back"))
		return strings.Join(lines, "\n")

	case reinvestStepSuccess:
		return strings.Join([]string{
			titleStyle.Render("Reinvestment recorded."),
			"",
			fmt.Sprintf("  Bucket: %s  Amount: %s", v.bucket.Name, formatCurrency(v.amount)),
			"",
			dimStyle.Render("Enter or Esc to return to dashboard"),
		}, "\n")
	}
	return ""
}

func (v *ReinvestView) Resize(w, h int) { v.width = w; v.height = h }

func (v ReinvestView) submit(date time.Time) tea.Cmd {
	c := v.client
	bucketID := v.bucket.ID
	amount := v.amount
	return func() tea.Msg {
		err := c.RecordReinvestment(context.Background(), bucketID, amount, date)
		return reinvestSubmittedMsg{err: err}
	}
}
