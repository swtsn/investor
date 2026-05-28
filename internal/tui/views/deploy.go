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

type deployStep int

const (
	stepBucketSelect deployStep = iota
	stepFillType
	stepAmountEntry
	stepSymbolEntry
	stepSharesEntry
	stepPriceEntry
	stepAllocationPick
	stepSourcePick
	stepConfirm
	stepSuccess
)

type fillType int

const (
	fillTypeAmountOnly fillType = iota
	fillTypeFullFill
)

type sourceEntry struct {
	source    client.DeployableSource
	entered   decimal.Decimal
	inputOpen bool
	input     textinput.Model
}

type allocPickLoadedMsg struct {
	allocs []domain.Allocation
	splits []client.AllocationSplit
	err    error
}

type sourcesLoadedMsg struct {
	sources []sourceEntry
	err     error
}

type deploySubmittedMsg struct{ err error }

// DeployView implements the full deploy state machine from ADR-011.
type DeployView struct {
	client client.Client

	step    deployStep
	buckets []domain.Bucket

	// BucketSelect
	bucketCursor int

	// Selected context
	selectedBucket domain.Bucket
	fillType       fillType

	// Text inputs
	amountInput textinput.Model
	symbolInput textinput.Model
	sharesInput textinput.Model
	priceInput  textinput.Model
	dateInput   textinput.Model

	// Computed values
	amount        decimal.Decimal
	symbol        string
	shares        *decimal.Decimal
	pricePerShare *decimal.Decimal

	// AllocationPick
	allocationCursor int
	allocations      []domain.Allocation
	allocSplits      []client.AllocationSplit
	selectedAllocID  *int64

	// SourcePick
	sources      []sourceEntry
	sourceCursor int

	loading   bool
	submitErr error
	width     int
	height    int
}

func NewDeployView(c client.Client) DeployView {
	newInput := func(placeholder string) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.CharLimit = 30
		return ti
	}
	return DeployView{
		client:      c,
		amountInput: newInput("0.00"),
		symbolInput: newInput("AAPL"),
		sharesInput: newInput("0"),
		priceInput:  newInput("0.00"),
		dateInput:   newInput("2006-01-02"),
	}
}

func (v DeployView) InputActive() bool {
	switch v.step {
	case stepAmountEntry, stepSymbolEntry, stepSharesEntry, stepPriceEntry, stepConfirm:
		return true
	case stepSourcePick:
		for i := range v.sources {
			if v.sources[i].inputOpen {
				return true
			}
		}
	}
	return false
}

func (v DeployView) Update(msg tea.Msg, state SharedState) (DeployView, tea.Cmd) {
	switch msg := msg.(type) {
	case LoadMsg:
		v.buckets = state.Buckets
		v.step = stepBucketSelect
		v.bucketCursor = 0
		v.submitErr = nil
		v.sources = nil
		v.allocations = nil
		v.selectedAllocID = nil
		return v, nil

	case allocPickLoadedMsg:
		v.loading = false
		if msg.err != nil {
			v.submitErr = msg.err
			v.step = stepAmountEntry
			return v, nil
		}
		v.allocations = msg.allocs
		v.allocSplits = msg.splits
		v.allocationCursor = 0
		v.step = stepAllocationPick
		return v, nil

	case sourcesLoadedMsg:
		v.loading = false
		if msg.err != nil {
			v.submitErr = msg.err
			v.step = stepAllocationPick
			if v.selectedBucket.Type == domain.BucketTypeFlat {
				v.step = stepAmountEntry
			}
			return v, nil
		}
		v.sources = msg.sources
		v.sourceCursor = 0
		v.step = stepSourcePick
		return v, nil

	case deploySubmittedMsg:
		if msg.err != nil {
			v.submitErr = msg.err
			return v, nil
		}
		v.step = stepSuccess
		return v, nil

	case tea.KeyMsg:
		return v.handleKey(msg)
	}
	return v, nil
}

func (v DeployView) handleKey(msg tea.KeyMsg) (DeployView, tea.Cmd) {
	switch v.step {
	case stepBucketSelect:
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
			v.selectedBucket = v.buckets[v.bucketCursor]
			v.submitErr = nil
			if v.selectedBucket.Type == domain.BucketTypeFlat {
				v.step = stepAmountEntry
				v.amountInput.SetValue("")
				v.amountInput.Focus()
				return v, textinput.Blink
			}
			// diversified: choose fill type
			v.step = stepFillType
			v.fillType = fillTypeAmountOnly
		case "esc":
			return v, func() tea.Msg { return BackMsg{} }
		}

	case stepFillType:
		switch msg.String() {
		case "1":
			v.fillType = fillTypeAmountOnly
			v.step = stepAmountEntry
			v.amountInput.SetValue("")
			v.amountInput.Focus()
			return v, textinput.Blink
		case "2":
			v.fillType = fillTypeFullFill
			v.step = stepSymbolEntry
			v.symbolInput.SetValue("")
			v.symbolInput.Focus()
			return v, textinput.Blink
		case "esc":
			v.step = stepBucketSelect
		}

	case stepAmountEntry:
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
			if v.selectedBucket.Type == domain.BucketTypeFlat {
				v.loading = true
				return v, v.loadSources()
			}
			// diversified amount-only: load allocation picker
			v.loading = true
			return v, v.loadAllocations()
		case "esc":
			v.amountInput.Blur()
			v.submitErr = nil
			if v.selectedBucket.Type == domain.BucketTypeFlat {
				v.step = stepBucketSelect
			} else {
				v.step = stepFillType
			}
		default:
			var cmd tea.Cmd
			v.amountInput, cmd = v.amountInput.Update(msg)
			return v, cmd
		}

	case stepSymbolEntry:
		switch msg.String() {
		case "enter":
			sym := strings.TrimSpace(v.symbolInput.Value())
			if sym == "" {
				v.submitErr = fmt.Errorf("enter a symbol")
				return v, nil
			}
			v.symbol = sym
			v.submitErr = nil
			v.symbolInput.Blur()
			v.step = stepSharesEntry
			v.sharesInput.SetValue("")
			v.sharesInput.Focus()
			return v, textinput.Blink
		case "esc":
			v.symbolInput.Blur()
			v.submitErr = nil
			v.step = stepFillType
		default:
			var cmd tea.Cmd
			v.symbolInput, cmd = v.symbolInput.Update(msg)
			return v, cmd
		}

	case stepSharesEntry:
		switch msg.String() {
		case "enter":
			shares, err := decimal.NewFromString(v.sharesInput.Value())
			if err != nil || !shares.IsPositive() {
				v.submitErr = fmt.Errorf("enter a positive number of shares")
				return v, nil
			}
			v.shares = &shares
			v.submitErr = nil
			v.sharesInput.Blur()
			v.step = stepPriceEntry
			v.priceInput.SetValue("")
			v.priceInput.Focus()
			return v, textinput.Blink
		case "esc":
			v.sharesInput.Blur()
			v.submitErr = nil
			v.step = stepSymbolEntry
			v.symbolInput.SetValue(v.symbol)
			v.symbolInput.Focus()
			return v, textinput.Blink
		default:
			var cmd tea.Cmd
			v.sharesInput, cmd = v.sharesInput.Update(msg)
			return v, cmd
		}

	case stepPriceEntry:
		switch msg.String() {
		case "enter":
			price, err := decimal.NewFromString(v.priceInput.Value())
			if err != nil || !price.IsPositive() {
				v.submitErr = fmt.Errorf("enter a positive price")
				return v, nil
			}
			v.pricePerShare = &price
			v.amount = v.shares.Mul(price)
			v.submitErr = nil
			v.priceInput.Blur()
			v.loading = true
			return v, v.loadAllocations()
		case "esc":
			v.priceInput.Blur()
			v.submitErr = nil
			v.step = stepSharesEntry
			if v.shares != nil {
				v.sharesInput.SetValue(v.shares.String())
			}
			v.sharesInput.Focus()
			return v, textinput.Blink
		default:
			var cmd tea.Cmd
			v.priceInput, cmd = v.priceInput.Update(msg)
			return v, cmd
		}

	case stepAllocationPick:
		switch msg.String() {
		case "up", "k":
			if v.allocationCursor > 0 {
				v.allocationCursor--
			}
		case "down", "j":
			if v.allocationCursor < len(v.allocations)-1 {
				v.allocationCursor++
			}
		case "enter":
			if len(v.allocations) == 0 {
				return v, nil
			}
			id := v.allocations[v.allocationCursor].ID
			v.selectedAllocID = &id
			v.loading = true
			return v, v.loadSources()
		case "esc":
			v.submitErr = nil
			if v.fillType == fillTypeFullFill {
				v.step = stepPriceEntry
				v.priceInput.Focus()
				return v, textinput.Blink
			}
			v.step = stepAmountEntry
			v.amountInput.SetValue(v.amount.String())
			v.amountInput.Focus()
			return v, textinput.Blink
		}

	case stepSourcePick:
		return v.handleSourcePick(msg)

	case stepConfirm:
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
			v.submitErr = nil
			v.step = stepSourcePick
		default:
			var cmd tea.Cmd
			v.dateInput, cmd = v.dateInput.Update(msg)
			return v, cmd
		}

	case stepSuccess:
		if msg.String() == "esc" || msg.String() == "enter" {
			return v, func() tea.Msg { return BackMsg{} }
		}
	}
	return v, nil
}

func (v DeployView) handleSourcePick(msg tea.KeyMsg) (DeployView, tea.Cmd) {
	// Check if any input is open
	for i := range v.sources {
		if v.sources[i].inputOpen {
			switch msg.String() {
			case "enter":
				s := v.sources[i].input.Value()
				a, err := decimal.NewFromString(s)
				if err != nil || !a.IsPositive() {
					return v, nil
				}
				v.sources[i].entered = a
				v.sources[i].inputOpen = false
				v.sources[i].input.Blur()
				return v, nil
			case "esc":
				v.sources[i].inputOpen = false
				v.sources[i].input.Blur()
				return v, nil
			default:
				var cmd tea.Cmd
				v.sources[i].input, cmd = v.sources[i].input.Update(msg)
				return v, cmd
			}
		}
	}

	// No input open
	switch msg.String() {
	case "up", "k":
		if v.sourceCursor > 0 {
			v.sourceCursor--
		}
	case "down", "j":
		if v.sourceCursor < len(v.sources)-1 {
			v.sourceCursor++
		}
	case "enter":
		total := v.sourceTotal()
		if total.Equal(v.amount) {
			// All allocated; advance to confirm.
			v.step = stepConfirm
			v.dateInput.SetValue(time.Now().Format("2006-01-02"))
			v.dateInput.Focus()
			return v, textinput.Blink
		}
		if len(v.sources) == 0 {
			return v, nil
		}
		// Open inline input for cursor source.
		cur := &v.sources[v.sourceCursor]
		if cur.entered.IsZero() {
			cur.input.SetValue("")
		} else {
			cur.input.SetValue(cur.entered.String())
		}
		cur.input.Focus()
		cur.inputOpen = true
		return v, textinput.Blink
	case "esc":
		v.submitErr = nil
		if v.selectedBucket.Type == domain.BucketTypeFlat {
			v.step = stepAmountEntry
			v.amountInput.SetValue(v.amount.String())
			v.amountInput.Focus()
			return v, textinput.Blink
		}
		v.step = stepAllocationPick
	}
	return v, nil
}

func (v DeployView) sourceTotal() decimal.Decimal {
	var total decimal.Decimal
	for _, s := range v.sources {
		total = total.Add(s.entered)
	}
	return total
}

func (v DeployView) View() string {
	switch v.step {
	case stepBucketSelect:
		if len(v.buckets) == 0 {
			return titleStyle.Render("Deploy — Select Bucket") + "\n\n" +
				dimStyle.Render("No buckets configured.  Esc to go back.")
		}
		lines := []string{titleStyle.Render("Deploy — Select Bucket"), ""}
		for i, b := range v.buckets {
			cursor := "  "
			if i == v.bucketCursor {
				cursor = "> "
			}
			lines = append(lines, fmt.Sprintf("%s%s  (%s)", cursor, b.Name, string(b.Type)))
		}
		lines = append(lines, "", dimStyle.Render("↑↓ navigate  Enter to select  Esc to go back"))
		return strings.Join(lines, "\n")

	case stepFillType:
		return strings.Join([]string{
			titleStyle.Render("Deploy — Fill Type"),
			"",
			fmt.Sprintf("Bucket: %s", v.selectedBucket.Name),
			"",
			"  [1] Amount only",
			"  [2] Full fill (symbol + shares × price)",
			"",
			dimStyle.Render("1 or 2 to select  Esc to go back"),
		}, "\n")

	case stepAmountEntry:
		lines := []string{
			titleStyle.Render("Deploy — Enter Amount"),
			"",
			fmt.Sprintf("Bucket: %s", v.selectedBucket.Name),
			"",
			"Amount:",
			v.amountInput.View(),
		}
		if v.submitErr != nil {
			lines = append(lines, "", errorStyle.Render(v.submitErr.Error()))
		}
		lines = append(lines, "", dimStyle.Render("Enter to continue  Esc to go back"))
		return strings.Join(lines, "\n")

	case stepSymbolEntry:
		lines := []string{
			titleStyle.Render("Deploy — Enter Symbol"),
			"",
			fmt.Sprintf("Bucket: %s", v.selectedBucket.Name),
			"",
			"Symbol:",
			v.symbolInput.View(),
		}
		if v.submitErr != nil {
			lines = append(lines, "", errorStyle.Render(v.submitErr.Error()))
		}
		lines = append(lines, "", dimStyle.Render("Enter to continue  Esc to go back"))
		return strings.Join(lines, "\n")

	case stepSharesEntry:
		lines := []string{
			titleStyle.Render("Deploy — Enter Shares"),
			"",
			fmt.Sprintf("Bucket: %s  Symbol: %s", v.selectedBucket.Name, v.symbol),
			"",
			"Number of shares:",
			v.sharesInput.View(),
		}
		if v.submitErr != nil {
			lines = append(lines, "", errorStyle.Render(v.submitErr.Error()))
		}
		lines = append(lines, "", dimStyle.Render("Enter to continue  Esc to go back"))
		return strings.Join(lines, "\n")

	case stepPriceEntry:
		sharesStr := ""
		if v.shares != nil {
			sharesStr = v.shares.String()
		}
		lines := []string{
			titleStyle.Render("Deploy — Enter Price"),
			"",
			fmt.Sprintf("Bucket: %s  Symbol: %s  Shares: %s", v.selectedBucket.Name, v.symbol, sharesStr),
			"",
			"Price per share:",
			v.priceInput.View(),
		}
		if v.submitErr != nil {
			lines = append(lines, "", errorStyle.Render(v.submitErr.Error()))
		}
		lines = append(lines, "", dimStyle.Render("Enter to continue  Esc to go back"))
		return strings.Join(lines, "\n")

	case stepAllocationPick:
		if v.loading {
			return "Loading allocations..."
		}
		lines := []string{
			titleStyle.Render("Deploy — Select Allocation"),
			"",
			fmt.Sprintf("Bucket: %s  Amount: %s", v.selectedBucket.Name, formatCurrency(v.amount)),
			"",
			fmt.Sprintf("  %-14s  %-8s  %s", "Allocation", "Target%", "Preview"),
		}
		for i, a := range v.allocations {
			cursor := "  "
			if i == v.allocationCursor {
				cursor = "> "
			}
			preview := "—"
			for _, s := range v.allocSplits {
				if s.Allocation.ID == a.ID {
					preview = formatCurrency(s.Amount)
					break
				}
			}
			lines = append(lines, fmt.Sprintf("%s%-14s  %-8s  %s",
				cursor, a.Name, a.TargetPct.StringFixed(1)+"%", preview))
		}
		lines = append(lines, "", dimStyle.Render("↑↓ navigate  Enter to select  Esc to go back"))
		return strings.Join(lines, "\n")

	case stepSourcePick:
		if v.loading {
			return "Loading sources..."
		}
		total := v.sourceTotal()
		lines := []string{
			titleStyle.Render("Deploy — Pick Sources"),
			"",
			fmt.Sprintf("Bucket: %s  Amount: %s", v.selectedBucket.Name, formatCurrency(v.amount)),
			fmt.Sprintf("Allocated: %s / %s", formatCurrency(total), formatCurrency(v.amount)),
			"",
		}
		if len(v.sources) == 0 {
			lines = append(lines, dimStyle.Render("No deployable sources available."))
		}
		for i, s := range v.sources {
			cursor := "  "
			if i == v.sourceCursor {
				cursor = "> "
			}
			orig := string(s.source.Contribution.Origination)
			date := s.source.Contribution.Date.Format("2006-01-02")
			remaining := formatCurrency(s.source.Remaining)
			if s.inputOpen {
				lines = append(lines, fmt.Sprintf("%s%s  %s  remaining:%s  [%s]",
					cursor, orig, date, remaining, s.input.View()))
			} else {
				entered := ""
				if !s.entered.IsZero() {
					entered = "  → " + formatCurrency(s.entered)
				}
				lines = append(lines, fmt.Sprintf("%s%s  %s  remaining:%s%s",
					cursor, orig, date, remaining, entered))
			}
		}
		hint := "↑↓ navigate  Enter to edit amount  Esc to go back"
		if total.Equal(v.amount) {
			hint = "All allocated.  Enter to continue to confirm  Esc to go back"
		}
		lines = append(lines, "", dimStyle.Render(hint))
		return strings.Join(lines, "\n")

	case stepConfirm:
		lines := []string{
			titleStyle.Render("Deploy — Confirm"),
			"",
			fmt.Sprintf("  Bucket:  %s", v.selectedBucket.Name),
		}
		if v.symbol != "" {
			lines = append(lines, fmt.Sprintf("  Symbol:  %s", v.symbol))
		}
		if v.shares != nil {
			lines = append(lines, fmt.Sprintf("  Shares:  %s", v.shares.String()))
		}
		if v.pricePerShare != nil {
			lines = append(lines, fmt.Sprintf("  Price:   %s", formatCurrency(*v.pricePerShare)))
		}
		lines = append(lines, fmt.Sprintf("  Amount:  %s", formatCurrency(v.amount)))
		lines = append(lines, "", "Date (YYYY-MM-DD):", v.dateInput.View())
		if v.submitErr != nil {
			lines = append(lines, "", errorStyle.Render(v.submitErr.Error()))
		}
		lines = append(lines, "", dimStyle.Render("Enter to record  Esc to go back"))
		return strings.Join(lines, "\n")

	case stepSuccess:
		return strings.Join([]string{
			titleStyle.Render("Deployment recorded."),
			"",
			fmt.Sprintf("  Bucket: %s  Amount: %s", v.selectedBucket.Name, formatCurrency(v.amount)),
			"",
			dimStyle.Render("Enter or Esc to return to dashboard"),
		}, "\n")
	}
	return ""
}

func (v *DeployView) Resize(w, h int) { v.width = w; v.height = h }

func (v DeployView) loadAllocations() tea.Cmd {
	c := v.client
	bucketID := v.selectedBucket.ID
	amount := v.amount
	return func() tea.Msg {
		ctx := context.Background()
		allocs, err := c.ListAllocations(ctx, bucketID)
		if err != nil {
			return allocPickLoadedMsg{err: err}
		}
		splits, err := c.PreviewDeployment(ctx, bucketID, amount)
		if err != nil {
			return allocPickLoadedMsg{err: err}
		}
		return allocPickLoadedMsg{allocs: allocs, splits: splits}
	}
}

func (v DeployView) loadSources() tea.Cmd {
	c := v.client
	bucketID := v.selectedBucket.ID
	return func() tea.Msg {
		srcs, err := c.ListDeployableSources(context.Background(), bucketID)
		if err != nil {
			return sourcesLoadedMsg{err: err}
		}
		entries := make([]sourceEntry, len(srcs))
		for i, s := range srcs {
			ti := textinput.New()
			ti.Placeholder = "0.00"
			ti.CharLimit = 20
			entries[i] = sourceEntry{source: s, input: ti}
		}
		return sourcesLoadedMsg{sources: entries}
	}
}

func (v DeployView) submit(date time.Time) tea.Cmd {
	c := v.client

	d := domain.Deployment{
		BucketID:      v.selectedBucket.ID,
		AllocationID:  v.selectedAllocID,
		Amount:        v.amount,
		Date:          date,
	}
	if v.symbol != "" {
		s := v.symbol
		d.Symbol = &s
	}
	d.Shares = v.shares
	d.PricePerShare = v.pricePerShare

	sources := make([]domain.DeploymentSource, 0, len(v.sources))
	for _, s := range v.sources {
		if s.entered.IsPositive() {
			sources = append(sources, domain.DeploymentSource{
				ContributionID: s.source.Contribution.ID,
				Amount:         s.entered,
			})
		}
	}

	return func() tea.Msg {
		err := c.RecordDeployment(context.Background(), d, sources)
		return deploySubmittedMsg{err: err}
	}
}
