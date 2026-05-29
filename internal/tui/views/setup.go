package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/shopspring/decimal"

	"github.com/swtsn/investor/internal/domain"
	"github.com/swtsn/investor/internal/tui/client"
)

type setupState int

const (
	setupStateBucketList setupState = iota
	setupStateBucketCreateForm
	setupStateBucketEditForm
	setupStateAllocationList
	setupStateAllocationUpsertForm
	setupStateAllocationDeleteConfirm
)

type setupLoadedMsg struct {
	buckets     []domain.Bucket
	allocations map[int64][]domain.Allocation
	err         error
}
type setupBucketCreatedMsg struct {
	bucket domain.Bucket
	err    error
}
type setupBucketUpdatedMsg struct {
	bucket domain.Bucket
	err    error
}
type setupAllocSavedMsg struct {
	alloc domain.Allocation
	err   error
}
type setupAllocDeletedMsg struct{ err error }

// SetupView manages the bucket/allocation create-edit surface.
type SetupView struct {
	client client.Client
	state  setupState

	buckets      []domain.Bucket
	allocations  map[int64][]domain.Allocation
	bucketCursor int

	selectedBucket domain.Bucket
	selectedAlloc  domain.Allocation
	allocCursor    int
	allocIsCreate  bool

	// Form inputs (shared across forms; reset before each use)
	nameInput  textinput.Model
	pctInput   textinput.Model
	formField  int // 0=name, 1=type (create bucket only), 2=pct
	createType domain.BucketType

	submitErr error
	width     int
	height    int
}

func NewSetupView(c client.Client) SetupView {
	ni := textinput.New()
	ni.CharLimit = 50

	pi := textinput.New()
	pi.Placeholder = "0"
	pi.CharLimit = 10

	return SetupView{
		client:      c,
		allocations: make(map[int64][]domain.Allocation),
		nameInput:   ni,
		pctInput:    pi,
		createType:  domain.BucketTypeDiversified,
	}
}

func (v SetupView) InputActive() bool {
	switch v.state {
	case setupStateBucketCreateForm, setupStateBucketEditForm, setupStateAllocationUpsertForm, setupStateAllocationDeleteConfirm:
		return true
	}
	return false
}

func (v SetupView) Update(msg tea.Msg, _ SharedState) (SetupView, tea.Cmd) {
	switch msg := msg.(type) {
	case LoadMsg:
		v.state = setupStateBucketList
		v.bucketCursor = 0
		v.submitErr = nil
		return v, v.loadAll()

	case setupLoadedMsg:
		if msg.err != nil {
			v.submitErr = msg.err
			return v, nil
		}
		v.buckets = msg.buckets
		v.allocations = msg.allocations
		return v, nil

	case setupBucketCreatedMsg:
		if msg.err != nil {
			v.submitErr = msg.err
			return v, nil
		}
		v.buckets = append(v.buckets, msg.bucket)
		v.bucketCursor = len(v.buckets) - 1
		v.state = setupStateBucketList
		v.submitErr = nil
		return v, nil

	case setupBucketUpdatedMsg:
		if msg.err != nil {
			v.submitErr = msg.err
			return v, nil
		}
		for i, b := range v.buckets {
			if b.ID == msg.bucket.ID {
				v.buckets[i] = msg.bucket
				break
			}
		}
		v.state = setupStateBucketList
		v.submitErr = nil
		return v, nil

	case setupAllocSavedMsg:
		if msg.err != nil {
			v.submitErr = msg.err
			return v, nil
		}
		allocs := v.allocations[v.selectedBucket.ID]
		updated := false
		for i, a := range allocs {
			if a.Name == msg.alloc.Name {
				allocs[i] = msg.alloc
				updated = true
				break
			}
		}
		if !updated {
			allocs = append(allocs, msg.alloc)
		}
		v.allocations[v.selectedBucket.ID] = allocs
		v.state = setupStateAllocationList
		v.submitErr = nil
		return v, nil

	case setupAllocDeletedMsg:
		if msg.err != nil {
			v.submitErr = msg.err
			v.state = setupStateAllocationList
			return v, nil
		}
		allocs := v.allocations[v.selectedBucket.ID]
		for i, a := range allocs {
			if a.ID == v.selectedAlloc.ID {
				v.allocations[v.selectedBucket.ID] = append(allocs[:i], allocs[i+1:]...)
				break
			}
		}
		if v.allocCursor >= len(v.allocations[v.selectedBucket.ID]) && v.allocCursor > 0 {
			v.allocCursor--
		}
		v.state = setupStateAllocationList
		v.submitErr = nil
		return v, nil

	case tea.KeyMsg:
		return v.handleKey(msg)
	}
	return v, nil
}

func (v SetupView) handleKey(msg tea.KeyMsg) (SetupView, tea.Cmd) {
	switch v.state {
	case setupStateBucketList:
		return v.handleBucketList(msg)
	case setupStateBucketCreateForm:
		return v.handleBucketCreateForm(msg)
	case setupStateBucketEditForm:
		return v.handleBucketEditForm(msg)
	case setupStateAllocationList:
		return v.handleAllocationList(msg)
	case setupStateAllocationUpsertForm:
		return v.handleAllocationUpsertForm(msg)
	case setupStateAllocationDeleteConfirm:
		return v.handleDeleteConfirm(msg)
	}
	return v, nil
}

func (v SetupView) handleBucketList(msg tea.KeyMsg) (SetupView, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if v.bucketCursor > 0 {
			v.bucketCursor--
		}
	case "down", "j":
		if v.bucketCursor < len(v.buckets)-1 {
			v.bucketCursor++
		}
	case "c":
		v.submitErr = nil
		v.formField = 0
		v.createType = domain.BucketTypeDiversified
		v.nameInput.SetValue("")
		v.nameInput.Focus()
		v.pctInput.SetValue("")
		v.state = setupStateBucketCreateForm
		return v, textinput.Blink
	case "e":
		if len(v.buckets) == 0 {
			return v, nil
		}
		b := v.buckets[v.bucketCursor]
		v.selectedBucket = b
		v.submitErr = nil
		v.formField = 0
		v.nameInput.SetValue(b.Name)
		v.nameInput.Focus()
		v.pctInput.SetValue(b.TargetPct.String())
		v.state = setupStateBucketEditForm
		return v, textinput.Blink
	case "enter":
		if len(v.buckets) == 0 {
			return v, nil
		}
		v.selectedBucket = v.buckets[v.bucketCursor]
		v.allocCursor = 0
		v.submitErr = nil
		v.state = setupStateAllocationList
	case "esc":
		return v, func() tea.Msg { return BackMsg{} }
	}
	return v, nil
}

func (v SetupView) handleBucketCreateForm(msg tea.KeyMsg) (SetupView, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.nameInput.Blur()
		v.pctInput.Blur()
		v.state = setupStateBucketList
		v.submitErr = nil
		return v, nil
	case "enter":
		switch v.formField {
		case 0: // name → type
			if strings.TrimSpace(v.nameInput.Value()) == "" {
				v.submitErr = fmt.Errorf("name is required")
				return v, nil
			}
			v.submitErr = nil
			v.nameInput.Blur()
			v.formField = 1
			return v, nil
		case 1: // type → pct
			v.formField = 2
			v.pctInput.Focus()
			return v, textinput.Blink
		case 2: // submit
			return v.submitCreateBucket()
		}
	case " ", "tab":
		if v.formField == 1 {
			if v.createType == domain.BucketTypeFlat {
				v.createType = domain.BucketTypeDiversified
			} else {
				v.createType = domain.BucketTypeFlat
			}
		}
	default:
		if v.formField == 0 {
			var cmd tea.Cmd
			v.nameInput, cmd = v.nameInput.Update(msg)
			return v, cmd
		}
		if v.formField == 2 {
			var cmd tea.Cmd
			v.pctInput, cmd = v.pctInput.Update(msg)
			return v, cmd
		}
	}
	return v, nil
}

func (v SetupView) submitCreateBucket() (SetupView, tea.Cmd) {
	name := strings.TrimSpace(v.nameInput.Value())
	if name == "" {
		v.submitErr = fmt.Errorf("name is required")
		return v, nil
	}
	pct, err := decimal.NewFromString(v.pctInput.Value())
	if err != nil || !pct.IsPositive() {
		v.submitErr = fmt.Errorf("enter a positive percentage")
		return v, nil
	}
	v.submitErr = nil
	v.pctInput.Blur()
	c := v.client
	bucketType := v.createType
	return v, func() tea.Msg {
		b, err := c.CreateBucket(context.Background(), name, bucketType, pct)
		return setupBucketCreatedMsg{bucket: b, err: err}
	}
}

func (v SetupView) handleBucketEditForm(msg tea.KeyMsg) (SetupView, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.nameInput.Blur()
		v.pctInput.Blur()
		v.state = setupStateBucketList
		v.submitErr = nil
		return v, nil
	case "enter":
		switch v.formField {
		case 0: // name → pct
			if strings.TrimSpace(v.nameInput.Value()) == "" {
				v.submitErr = fmt.Errorf("name is required")
				return v, nil
			}
			v.submitErr = nil
			v.nameInput.Blur()
			v.formField = 1
			v.pctInput.Focus()
			return v, textinput.Blink
		case 1: // submit
			name := strings.TrimSpace(v.nameInput.Value())
			pct, err := decimal.NewFromString(v.pctInput.Value())
			if err != nil || !pct.IsPositive() {
				v.submitErr = fmt.Errorf("enter a positive percentage")
				return v, nil
			}
			v.submitErr = nil
			v.pctInput.Blur()
			c := v.client
			id := v.selectedBucket.ID
			return v, func() tea.Msg {
				b, err := c.UpdateBucket(context.Background(), id, name, pct)
				return setupBucketUpdatedMsg{bucket: b, err: err}
			}
		}
	default:
		if v.formField == 0 {
			var cmd tea.Cmd
			v.nameInput, cmd = v.nameInput.Update(msg)
			return v, cmd
		}
		if v.formField == 1 {
			var cmd tea.Cmd
			v.pctInput, cmd = v.pctInput.Update(msg)
			return v, cmd
		}
	}
	return v, nil
}

func (v SetupView) handleAllocationList(msg tea.KeyMsg) (SetupView, tea.Cmd) {
	allocs := v.allocations[v.selectedBucket.ID]
	switch msg.String() {
	case "up", "k":
		if v.allocCursor > 0 {
			v.allocCursor--
		}
	case "down", "j":
		if v.allocCursor < len(allocs)-1 {
			v.allocCursor++
		}
	case "c":
		if v.selectedBucket.Type == domain.BucketTypeFlat {
			v.submitErr = fmt.Errorf("flat buckets do not support allocations")
			return v, nil
		}
		v.submitErr = nil
		v.allocIsCreate = true
		v.formField = 0
		v.nameInput.SetValue("")
		v.nameInput.Focus()
		v.pctInput.SetValue("")
		v.state = setupStateAllocationUpsertForm
		return v, textinput.Blink
	case "e":
		if len(allocs) == 0 {
			return v, nil
		}
		v.selectedAlloc = allocs[v.allocCursor]
		v.submitErr = nil
		v.allocIsCreate = false
		v.formField = 1
		v.pctInput.SetValue(v.selectedAlloc.TargetPct.String())
		v.pctInput.Focus()
		v.state = setupStateAllocationUpsertForm
		return v, textinput.Blink
	case "d":
		if len(allocs) == 0 {
			return v, nil
		}
		v.selectedAlloc = allocs[v.allocCursor]
		v.submitErr = nil
		v.state = setupStateAllocationDeleteConfirm
	case "esc":
		v.submitErr = nil
		v.state = setupStateBucketList
	}
	return v, nil
}

func (v SetupView) handleAllocationUpsertForm(msg tea.KeyMsg) (SetupView, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.nameInput.Blur()
		v.pctInput.Blur()
		v.state = setupStateAllocationList
		v.submitErr = nil
		return v, nil
	case "enter":
		if v.allocIsCreate && v.formField == 0 {
			if strings.TrimSpace(v.nameInput.Value()) == "" {
				v.submitErr = fmt.Errorf("name is required")
				return v, nil
			}
			v.submitErr = nil
			v.nameInput.Blur()
			v.formField = 1
			v.pctInput.Focus()
			return v, textinput.Blink
		}
		// submit (edit mode always lands here; create mode reaches here on field 1)
		name := v.selectedAlloc.Name
		if v.allocIsCreate {
			name = strings.TrimSpace(v.nameInput.Value())
		}
		pct, err := decimal.NewFromString(v.pctInput.Value())
		if err != nil || !pct.IsPositive() {
			v.submitErr = fmt.Errorf("enter a positive percentage")
			return v, nil
		}
		v.submitErr = nil
		v.pctInput.Blur()
		c := v.client
		bucketID := v.selectedBucket.ID
		return v, func() tea.Msg {
			a, err := c.UpsertAllocation(context.Background(), bucketID, name, pct)
			return setupAllocSavedMsg{alloc: a, err: err}
		}
	default:
		if v.allocIsCreate && v.formField == 0 {
			var cmd tea.Cmd
			v.nameInput, cmd = v.nameInput.Update(msg)
			return v, cmd
		}
		if !v.allocIsCreate || v.formField == 1 {
			var cmd tea.Cmd
			v.pctInput, cmd = v.pctInput.Update(msg)
			return v, cmd
		}
	}
	return v, nil
}

func (v SetupView) handleDeleteConfirm(msg tea.KeyMsg) (SetupView, tea.Cmd) {
	switch msg.String() {
	case "y":
		c := v.client
		id := v.selectedAlloc.ID
		return v, func() tea.Msg {
			err := c.DeleteAllocation(context.Background(), id)
			return setupAllocDeletedMsg{err: err}
		}
	case "n", "esc":
		v.state = setupStateAllocationList
		v.submitErr = nil
	}
	return v, nil
}

func (v SetupView) View() string {
	switch v.state {
	case setupStateBucketList, setupStateBucketCreateForm, setupStateBucketEditForm:
		return v.viewBuckets()
	case setupStateAllocationList, setupStateAllocationUpsertForm, setupStateAllocationDeleteConfirm:
		return v.viewAllocations()
	}
	return ""
}

func (v SetupView) viewBuckets() string {
	lines := []string{titleStyle.Render("Setup — Buckets"), ""}

	hundred := decimal.NewFromInt(100)
	var bucketSum decimal.Decimal
	for _, b := range v.buckets {
		bucketSum = bucketSum.Add(b.TargetPct)
	}
	pctOk := len(v.buckets) == 0 || bucketSum.Equal(hundred)

	if len(v.buckets) == 0 {
		lines = append(lines, dimStyle.Render("No buckets yet."))
	} else {
		for i, b := range v.buckets {
			cursor := "  "
			if i == v.bucketCursor {
				cursor = "> "
			}
			name := b.Name
			needsAttention := !pctOk
			if !needsAttention && b.Type == domain.BucketTypeDiversified {
				var aSum decimal.Decimal
				for _, a := range v.allocations[b.ID] {
					aSum = aSum.Add(a.TargetPct)
				}
				if len(v.allocations[b.ID]) > 0 && !aSum.Equal(hundred) {
					needsAttention = true
				}
			}
			marker := "  "
			if needsAttention {
				marker = warningStyle.Render("! ")
				name = warningStyle.Render(name)
			}
			lines = append(lines, fmt.Sprintf("%s%s%-16s  %s%%  (%s)",
				cursor, marker, name,
				b.TargetPct.StringFixed(1),
				string(b.Type)))
		}
	}

	// Panel or summary
	switch v.state {
	case setupStateBucketCreateForm:
		lines = append(lines, "", strings.Repeat("─", 40), "Create Bucket", "")
		lines = append(lines, v.renderBucketCreateForm()...)
	case setupStateBucketEditForm:
		lines = append(lines, "", strings.Repeat("─", 40), "Edit Bucket", "")
		lines = append(lines, v.renderBucketEditForm()...)
	default:
		if !pctOk {
			lines = append(lines, "", warningStyle.Render(fmt.Sprintf("Buckets sum to %s%% (need 100%%)", bucketSum.StringFixed(1))))
		}
		if v.submitErr != nil {
			lines = append(lines, "", errorStyle.Render(v.submitErr.Error()))
		}
		lines = append(lines, "", dimStyle.Render("↑↓ navigate  c create  e edit  Enter → allocations  Esc back"))
	}
	return strings.Join(lines, "\n")
}

func (v SetupView) renderBucketCreateForm() []string {
	lines := []string{}

	// Name field
	if v.formField == 0 {
		lines = append(lines, "Name: "+v.nameInput.View())
	} else {
		lines = append(lines, fmt.Sprintf("Name: %s", strings.TrimSpace(v.nameInput.Value())))
	}

	// Type toggle
	flat := "flat"
	div := "diversified"
	if v.createType == domain.BucketTypeFlat {
		flat = titleStyle.Render("[flat]")
	} else {
		div = titleStyle.Render("[diversified]")
	}
	typeLine := fmt.Sprintf("Type: %s / %s", flat, div)
	if v.formField == 1 {
		typeLine += "  " + dimStyle.Render("(Space/Tab to toggle, Enter to continue)")
	}
	lines = append(lines, typeLine)

	// Pct field
	if v.formField == 2 {
		lines = append(lines, "Pct%: "+v.pctInput.View())
	} else {
		lines = append(lines, fmt.Sprintf("Pct%%: %s", v.pctInput.Value()))
	}

	if v.submitErr != nil {
		lines = append(lines, "", errorStyle.Render(v.submitErr.Error()))
	}
	lines = append(lines, "", dimStyle.Render("Enter to advance  Esc to cancel"))
	return lines
}

func (v SetupView) renderBucketEditForm() []string {
	lines := []string{}

	if v.formField == 0 {
		lines = append(lines, "Name: "+v.nameInput.View())
	} else {
		lines = append(lines, fmt.Sprintf("Name: %s", strings.TrimSpace(v.nameInput.Value())))
	}
	lines = append(lines, fmt.Sprintf("Type: %s  (read-only)", string(v.selectedBucket.Type)))

	if v.formField == 1 {
		lines = append(lines, "Pct%: "+v.pctInput.View())
	} else {
		lines = append(lines, fmt.Sprintf("Pct%%: %s", v.pctInput.Value()))
	}

	if v.submitErr != nil {
		lines = append(lines, "", errorStyle.Render(v.submitErr.Error()))
	}
	lines = append(lines, "", dimStyle.Render("Enter to advance  Esc to cancel"))
	return lines
}

func (v SetupView) viewAllocations() string {
	allocs := v.allocations[v.selectedBucket.ID]
	lines := []string{titleStyle.Render(fmt.Sprintf("Setup — Allocations: %s", v.selectedBucket.Name)), ""}

	hundred := decimal.NewFromInt(100)
	var allocSum decimal.Decimal
	for _, a := range allocs {
		allocSum = allocSum.Add(a.TargetPct)
	}
	pctOk := len(allocs) == 0 || allocSum.Equal(hundred)

	if len(allocs) == 0 {
		lines = append(lines, dimStyle.Render("No allocations yet."))
	} else {
		for i, a := range allocs {
			cursor := "  "
			if i == v.allocCursor {
				cursor = "> "
			}
			name := a.Name
			if !pctOk {
				name = warningStyle.Render(name)
			}
			lines = append(lines, fmt.Sprintf("%s  %-16s  %s%%", cursor, name, a.TargetPct.StringFixed(1)))
		}
	}

	// Panel or summary
	switch v.state {
	case setupStateAllocationUpsertForm:
		label := "Create Allocation"
		if !v.allocIsCreate {
			label = "Edit Allocation"
		}
		lines = append(lines, "", strings.Repeat("─", 40), label, "")
		lines = append(lines, v.renderAllocUpsertForm()...)
	case setupStateAllocationDeleteConfirm:
		lines = append(lines, "", strings.Repeat("─", 40))
		lines = append(lines, warningStyle.Render(fmt.Sprintf(`Delete "%s"? [y/n]`, v.selectedAlloc.Name)))
		if v.submitErr != nil {
			lines = append(lines, "", errorStyle.Render(v.submitErr.Error()))
		}
	default:
		if !pctOk {
			lines = append(lines, "", warningStyle.Render(fmt.Sprintf("Allocations sum to %s%% (need 100%%)", allocSum.StringFixed(1))))
		}
		if v.submitErr != nil {
			lines = append(lines, "", errorStyle.Render(v.submitErr.Error()))
		}
		lines = append(lines, "", dimStyle.Render("↑↓ navigate  c create  e edit  d delete  Esc back"))
	}
	return strings.Join(lines, "\n")
}

func (v SetupView) renderAllocUpsertForm() []string {
	lines := []string{}

	if v.allocIsCreate {
		if v.formField == 0 {
			lines = append(lines, "Name: "+v.nameInput.View())
		} else {
			lines = append(lines, fmt.Sprintf("Name: %s", strings.TrimSpace(v.nameInput.Value())))
		}
		if v.formField == 1 {
			lines = append(lines, "Pct%: "+v.pctInput.View())
		} else {
			lines = append(lines, fmt.Sprintf("Pct%%: %s", v.pctInput.Value()))
		}
	} else {
		lines = append(lines, fmt.Sprintf("Name: %s  (read-only)", v.selectedAlloc.Name))
		lines = append(lines, "Pct%: "+v.pctInput.View())
	}

	if v.submitErr != nil {
		lines = append(lines, "", errorStyle.Render(v.submitErr.Error()))
	}
	lines = append(lines, "", dimStyle.Render("Enter to advance/save  Esc to cancel"))
	return lines
}

func (v *SetupView) Resize(w, h int) { v.width = w; v.height = h }

func (v SetupView) loadAll() tea.Cmd {
	c := v.client
	return func() tea.Msg {
		ctx := context.Background()
		buckets, err := c.ListBuckets(ctx)
		if err != nil {
			return setupLoadedMsg{err: err}
		}
		allocs := make(map[int64][]domain.Allocation)
		for _, b := range buckets {
			if b.Type == domain.BucketTypeDiversified {
				list, err := c.ListAllocations(ctx, b.ID)
				if err != nil {
					return setupLoadedMsg{err: err}
				}
				allocs[b.ID] = list
			}
		}
		return setupLoadedMsg{buckets: buckets, allocations: allocs}
	}
}
