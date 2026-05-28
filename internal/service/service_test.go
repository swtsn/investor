package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/swtsn/investor/internal/db"
	"github.com/swtsn/investor/internal/domain"
	"github.com/swtsn/investor/internal/service"
)

var ctx = context.Background()

func newTestStore(t *testing.T) *db.Store {
	t.Helper()
	s, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }
func pInt64(i int64) *int64        { return &i }

// --- BudgetService ---

func TestPreviewBudget(t *testing.T) {
	s := newTestStore(t)
	svc := service.NewBudgetService(s)

	_, _ = s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "Investing", Type: domain.BucketTypeDiversified, TargetPct: dec("60")})
	_, _ = s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "Options", Type: domain.BucketTypeFlat, TargetPct: dec("30")})
	_, _ = s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "Crypto", Type: domain.BucketTypeDiversified, TargetPct: dec("10")})

	allocs, err := svc.PreviewBudget(ctx, dec("1000"))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if len(allocs) != 3 {
		t.Fatalf("expected 3 allocations, got %d", len(allocs))
	}
	if !allocs[0].Amount.Equal(dec("600")) {
		t.Errorf("bucket 0: expected 600, got %s", allocs[0].Amount)
	}
	if !allocs[1].Amount.Equal(dec("300")) {
		t.Errorf("bucket 1: expected 300, got %s", allocs[1].Amount)
	}
	if !allocs[2].Amount.Equal(dec("100")) {
		t.Errorf("bucket 2: expected 100, got %s", allocs[2].Amount)
	}
}

func TestPreviewBudgetRoundingLastBucketAbsorbsRemainder(t *testing.T) {
	s := newTestStore(t)
	svc := service.NewBudgetService(s)

	// Three equal buckets at ~33.33%; decimal split will have a remainder.
	_, _ = s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "A", Type: domain.BucketTypeFlat, TargetPct: dec("33.33")})
	_, _ = s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "B", Type: domain.BucketTypeFlat, TargetPct: dec("33.33")})
	_, _ = s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "C", Type: domain.BucketTypeFlat, TargetPct: dec("33.34")})

	allocs, err := svc.PreviewBudget(ctx, dec("100"))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}

	total := allocs[0].Amount.Add(allocs[1].Amount).Add(allocs[2].Amount)
	if !total.Equal(dec("100")) {
		t.Errorf("sum must equal totalAmount exactly; got %s", total)
	}
	// A and B: 100 * 33.33 / 100 = 33.33 (exact with truncation)
	// C: 100 - 33.33 - 33.33 = 33.34
	if !allocs[2].Amount.Equal(dec("33.34")) {
		t.Errorf("last bucket should absorb remainder; got %s", allocs[2].Amount)
	}
}

func TestApplyBudgetMonthOpeningWithPriorBalance(t *testing.T) {
	s := newTestStore(t)
	svc := service.NewBudgetService(s)

	b1, _ := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "Investing", Type: domain.BucketTypeDiversified, TargetPct: dec("60")})
	b2, _ := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "Options", Type: domain.BucketTypeFlat, TargetPct: dec("40")})

	// Prior month contributions (creates positive balance).
	ev0, _ := s.BudgetEvents.CreateBudgetEvent(ctx, domain.BudgetEvent{
		TotalAmount: dec("500"), Date: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	})
	_, _ = s.Contributions.CreateContribution(ctx, domain.Contribution{
		BucketID: b1.ID, Amount: dec("300"), Origination: domain.OriginationBudget,
		BudgetEventID: pInt64(ev0.ID), Date: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	})
	_, _ = s.Contributions.CreateContribution(ctx, domain.Contribution{
		BucketID: b2.ID, Amount: dec("200"), Origination: domain.OriginationBudget,
		BudgetEventID: pInt64(ev0.ID), Date: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	})

	event, err := svc.ApplyBudget(ctx, dec("1000"), time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if event.ID == 0 {
		t.Fatal("expected non-zero BudgetEvent ID")
	}

	events, _ := s.BudgetEvents.ListByMonth(ctx, 2026, 5)
	if len(events) != 1 {
		t.Errorf("expected 1 budget event for May, got %d", len(events))
	}

	// b1: April budget + May slush(300) + May budget(600)
	c1, _ := s.Contributions.ListByBucket(ctx, b1.ID)
	if len(c1) != 3 {
		t.Fatalf("b1: expected 3 contributions, got %d", len(c1))
	}
	var maySlush, mayBudget decimal.Decimal
	for _, c := range c1 {
		if c.Date.Month() == 5 {
			switch c.Origination {
			case domain.OriginationSlush:
				maySlush = c.Amount
			case domain.OriginationBudget:
				mayBudget = c.Amount
			}
		}
	}
	if !maySlush.Equal(dec("300")) {
		t.Errorf("b1 May slush: expected 300, got %s", maySlush)
	}
	if !mayBudget.Equal(dec("600")) {
		t.Errorf("b1 May budget: expected 600, got %s", mayBudget)
	}

	// b2: April budget + May slush(200) + May budget(400)
	c2, _ := s.Contributions.ListByBucket(ctx, b2.ID)
	if len(c2) != 3 {
		t.Errorf("b2: expected 3 contributions, got %d", len(c2))
	}
}

func TestApplyBudgetMonthOpeningZeroBalance(t *testing.T) {
	s := newTestStore(t)
	svc := service.NewBudgetService(s)

	b1, _ := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "Investing", Type: domain.BucketTypeDiversified, TargetPct: dec("100")})

	_, err := svc.ApplyBudget(ctx, dec("1000"), time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	contribs, _ := s.Contributions.ListByBucket(ctx, b1.ID)
	if len(contribs) != 1 {
		t.Fatalf("expected 1 contribution (budget only, no slush), got %d", len(contribs))
	}
	if contribs[0].Origination != domain.OriginationBudget {
		t.Errorf("expected origination=budget, got %s", contribs[0].Origination)
	}
}

func TestApplyBudgetTopUp(t *testing.T) {
	s := newTestStore(t)
	svc := service.NewBudgetService(s)

	b1, _ := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "Investing", Type: domain.BucketTypeDiversified, TargetPct: dec("100")})

	_, err := svc.ApplyBudget(ctx, dec("1000"), time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("first apply: %v", err)
	}
	_, err = svc.ApplyBudget(ctx, dec("500"), time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("top-up apply: %v", err)
	}

	events, _ := s.BudgetEvents.ListByMonth(ctx, 2026, 5)
	if len(events) != 2 {
		t.Errorf("expected 2 budget events, got %d", len(events))
	}

	contribs, _ := s.Contributions.ListByBucket(ctx, b1.ID)
	// Two budget contributions; no slush (top-up skips slush).
	if len(contribs) != 2 {
		t.Fatalf("expected 2 contributions, got %d", len(contribs))
	}
	for _, c := range contribs {
		if c.Origination != domain.OriginationBudget {
			t.Errorf("expected only budget origination, got %s", c.Origination)
		}
	}
}

func TestApplyBudgetRollback(t *testing.T) {
	s := newTestStore(t)
	svc := service.NewBudgetService(s)

	b1, _ := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "Investing", Type: domain.BucketTypeDiversified, TargetPct: dec("60")})
	b2, _ := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "Options", Type: domain.BucketTypeFlat, TargetPct: dec("40")})

	// Unique index limits each origination value to one row total. The second
	// budget contribution (for bucket 2) will violate it mid-transaction.
	_, err := s.DB().ExecContext(ctx, `CREATE UNIQUE INDEX fail_dup_origination ON contributions (origination)`)
	if err != nil {
		t.Fatalf("create index: %v", err)
	}

	_, err = svc.ApplyBudget(ctx, dec("1000"), time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected mid-transaction error")
	}

	// Full rollback: no budget event, no contributions for either bucket.
	events, _ := s.BudgetEvents.ListByMonth(ctx, 2026, 5)
	if len(events) != 0 {
		t.Errorf("expected 0 budget events after rollback, got %d", len(events))
	}
	contribs1, _ := s.Contributions.ListByBucket(ctx, b1.ID)
	if len(contribs1) != 0 {
		t.Errorf("expected 0 contributions for b1 after rollback, got %d", len(contribs1))
	}
	contribs2, _ := s.Contributions.ListByBucket(ctx, b2.ID)
	if len(contribs2) != 0 {
		t.Errorf("expected 0 contributions for b2 after rollback, got %d", len(contribs2))
	}
}

// --- DeploymentService ---

func TestPreviewDeployment(t *testing.T) {
	s := newTestStore(t)
	svc := service.NewDeploymentService(s)

	b, _ := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "Investing", Type: domain.BucketTypeDiversified, TargetPct: dec("100")})
	_ = s.Buckets.UpsertAllocation(ctx, domain.Allocation{BucketID: b.ID, Name: "metals", TargetPct: dec("40")})
	_ = s.Buckets.UpsertAllocation(ctx, domain.Allocation{BucketID: b.ID, Name: "tech", TargetPct: dec("60")})

	splits, err := svc.PreviewDeployment(ctx, b.ID, dec("1000"))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if len(splits) != 2 {
		t.Fatalf("expected 2 splits, got %d", len(splits))
	}

	// metals (first, 40%): truncate(1000 * 40/100) = 400
	// tech (last, 60%): 1000 - 400 = 600
	if !splits[0].Amount.Equal(dec("400")) {
		t.Errorf("metals: expected 400, got %s", splits[0].Amount)
	}
	if !splits[1].Amount.Equal(dec("600")) {
		t.Errorf("tech: expected 600, got %s", splits[1].Amount)
	}
	total := splits[0].Amount.Add(splits[1].Amount)
	if !total.Equal(dec("1000")) {
		t.Errorf("splits must sum to 1000; got %s", total)
	}
}

func TestPreviewDeploymentFlatBucketError(t *testing.T) {
	s := newTestStore(t)
	svc := service.NewDeploymentService(s)

	b, _ := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "Options", Type: domain.BucketTypeFlat, TargetPct: dec("100")})

	_, err := svc.PreviewDeployment(ctx, b.ID, dec("1000"))
	if err == nil {
		t.Fatal("expected error for flat bucket")
	}
}

func TestRecordDeploymentHappyPath(t *testing.T) {
	s := newTestStore(t)
	svc := service.NewDeploymentService(s)

	b, _ := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "Options", Type: domain.BucketTypeFlat, TargetPct: dec("100")})
	ev, _ := s.BudgetEvents.CreateBudgetEvent(ctx, domain.BudgetEvent{
		TotalAmount: dec("1000"), Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	c, _ := s.Contributions.CreateContribution(ctx, domain.Contribution{
		BucketID: b.ID, Amount: dec("1000"), Origination: domain.OriginationBudget,
		BudgetEventID: pInt64(ev.ID), Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})

	d := domain.Deployment{
		BucketID: b.ID,
		Amount:   dec("500"),
		Date:     time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC),
	}
	result, err := svc.RecordDeployment(ctx, d, []domain.DeploymentSource{{ContributionID: c.ID, Amount: dec("500")}})
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if result.ID == 0 {
		t.Fatal("expected non-zero deployment ID")
	}

	deps, _ := s.Deployments.ListByBucket(ctx, b.ID)
	if len(deps) != 1 {
		t.Errorf("expected 1 deployment, got %d", len(deps))
	}
	srcs, _ := s.Deployments.ListSourcesByDeployment(ctx, result.ID)
	if len(srcs) != 1 || !srcs[0].Amount.Equal(dec("500")) {
		t.Errorf("unexpected sources: %+v", srcs)
	}
}

func TestRecordDeploymentSourceSumMismatch(t *testing.T) {
	s := newTestStore(t)
	svc := service.NewDeploymentService(s)

	b, _ := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "Options", Type: domain.BucketTypeFlat, TargetPct: dec("100")})
	ev, _ := s.BudgetEvents.CreateBudgetEvent(ctx, domain.BudgetEvent{
		TotalAmount: dec("1000"), Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	c, _ := s.Contributions.CreateContribution(ctx, domain.Contribution{
		BucketID: b.ID, Amount: dec("1000"), Origination: domain.OriginationBudget,
		BudgetEventID: pInt64(ev.ID), Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})

	d := domain.Deployment{BucketID: b.ID, Amount: dec("500"), Date: time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)}
	_, err := svc.RecordDeployment(ctx, d, []domain.DeploymentSource{{ContributionID: c.ID, Amount: dec("300")}})
	if err == nil {
		t.Fatal("expected error: source sum != deployment amount")
	}

	deps, _ := s.Deployments.ListByBucket(ctx, b.ID)
	if len(deps) != 0 {
		t.Errorf("expected 0 deployments after error, got %d", len(deps))
	}
}

func TestRecordDeploymentWrongBucketSource(t *testing.T) {
	s := newTestStore(t)
	svc := service.NewDeploymentService(s)

	b1, _ := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "B1", Type: domain.BucketTypeFlat, TargetPct: dec("50")})
	b2, _ := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "B2", Type: domain.BucketTypeFlat, TargetPct: dec("50")})
	ev, _ := s.BudgetEvents.CreateBudgetEvent(ctx, domain.BudgetEvent{
		TotalAmount: dec("1000"), Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	// Contribution belongs to b2.
	c, _ := s.Contributions.CreateContribution(ctx, domain.Contribution{
		BucketID: b2.ID, Amount: dec("500"), Origination: domain.OriginationBudget,
		BudgetEventID: pInt64(ev.ID), Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})

	// Deploy from b1 but fund with b2's contribution.
	d := domain.Deployment{BucketID: b1.ID, Amount: dec("500"), Date: time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)}
	_, err := svc.RecordDeployment(ctx, d, []domain.DeploymentSource{{ContributionID: c.ID, Amount: dec("500")}})
	if err == nil {
		t.Fatal("expected error: source belongs to wrong bucket")
	}

	deps, _ := s.Deployments.ListByBucket(ctx, b1.ID)
	if len(deps) != 0 {
		t.Errorf("expected 0 deployments after error, got %d", len(deps))
	}
}

func TestRecordReinvestment(t *testing.T) {
	s := newTestStore(t)
	svc := service.NewDeploymentService(s)

	b, _ := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "Investing", Type: domain.BucketTypeDiversified, TargetPct: dec("100")})

	c, err := svc.RecordReinvestment(ctx, b.ID, dec("250"), time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("reinvestment: %v", err)
	}
	if c.ID == 0 {
		t.Fatal("expected non-zero contribution ID")
	}
	if c.Origination != domain.OriginationReinvestment {
		t.Errorf("expected origination=reinvestment, got %s", c.Origination)
	}
	if c.BudgetEventID != nil {
		t.Errorf("expected BudgetEventID=nil, got %v", c.BudgetEventID)
	}
	if !c.Amount.Equal(dec("250")) {
		t.Errorf("expected amount=250, got %s", c.Amount)
	}
}

// --- PoolService ---

func TestGetPoolBalance(t *testing.T) {
	s := newTestStore(t)
	svc := service.NewPoolService(s)

	b, _ := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "Investing", Type: domain.BucketTypeDiversified, TargetPct: dec("100")})
	ev, _ := s.BudgetEvents.CreateBudgetEvent(ctx, domain.BudgetEvent{
		TotalAmount: dec("1000"), Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	c, _ := s.Contributions.CreateContribution(ctx, domain.Contribution{
		BucketID: b.ID, Amount: dec("1000"), Origination: domain.OriginationBudget,
		BudgetEventID: pInt64(ev.ID), Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})

	_ = s.InTx(ctx, func(tx *db.Store) error {
		_, err := tx.Deployments.CreateDeployment(ctx,
			domain.Deployment{BucketID: b.ID, Amount: dec("300"), Date: time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC)},
			[]domain.DeploymentSource{{ContributionID: c.ID, Amount: dec("300")}},
		)
		return err
	})

	bal, err := svc.GetPoolBalance(ctx, b.ID)
	if err != nil {
		t.Fatalf("balance: %v", err)
	}
	if !bal.Equal(dec("700")) {
		t.Errorf("expected 700, got %s", bal)
	}
}

func TestGetPoolBreakdown(t *testing.T) {
	s := newTestStore(t)
	svc := service.NewPoolService(s)

	b, _ := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "Investing", Type: domain.BucketTypeDiversified, TargetPct: dec("100")})
	aprilEv, _ := s.BudgetEvents.CreateBudgetEvent(ctx, domain.BudgetEvent{
		TotalAmount: dec("500"), Date: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	})
	mayEv, _ := s.BudgetEvents.CreateBudgetEvent(ctx, domain.BudgetEvent{
		TotalAmount: dec("1000"), Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})

	// April contribution — must not appear in May breakdown.
	_, _ = s.Contributions.CreateContribution(ctx, domain.Contribution{
		BucketID: b.ID, Amount: dec("500"), Origination: domain.OriginationBudget,
		BudgetEventID: pInt64(aprilEv.ID), Date: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	})

	// May contributions: budget + reinvestment + slush.
	_, _ = s.Contributions.CreateContribution(ctx, domain.Contribution{
		BucketID: b.ID, Amount: dec("1000"), Origination: domain.OriginationBudget,
		BudgetEventID: pInt64(mayEv.ID), Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	_, _ = s.Contributions.CreateContribution(ctx, domain.Contribution{
		BucketID: b.ID, Amount: dec("150"), Origination: domain.OriginationReinvestment,
		Date: time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC),
	})
	_, _ = s.Contributions.CreateContribution(ctx, domain.Contribution{
		BucketID: b.ID, Amount: dec("500"), Origination: domain.OriginationSlush,
		BudgetEventID: pInt64(mayEv.ID), Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})

	bd, err := svc.GetPoolBreakdown(ctx, b.ID, 2026, 5)
	if err != nil {
		t.Fatalf("breakdown: %v", err)
	}
	if !bd.Budget.Equal(dec("1000")) {
		t.Errorf("budget: expected 1000, got %s", bd.Budget)
	}
	if !bd.Reinvestment.Equal(dec("150")) {
		t.Errorf("reinvestment: expected 150, got %s", bd.Reinvestment)
	}
	if !bd.Slush.Equal(dec("500")) {
		t.Errorf("slush: expected 500, got %s", bd.Slush)
	}
	if !bd.Total.Equal(dec("1650")) {
		t.Errorf("total: expected 1650, got %s", bd.Total)
	}
}

func TestGetMonthSummary(t *testing.T) {
	s := newTestStore(t)
	svc := service.NewPoolService(s)

	b1, _ := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "Investing", Type: domain.BucketTypeDiversified, TargetPct: dec("60")})
	b2, _ := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "Options", Type: domain.BucketTypeFlat, TargetPct: dec("40")})

	ev, _ := s.BudgetEvents.CreateBudgetEvent(ctx, domain.BudgetEvent{
		TotalAmount: dec("1000"), Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	c1, _ := s.Contributions.CreateContribution(ctx, domain.Contribution{
		BucketID: b1.ID, Amount: dec("600"), Origination: domain.OriginationBudget,
		BudgetEventID: pInt64(ev.ID), Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	_, _ = s.Contributions.CreateContribution(ctx, domain.Contribution{
		BucketID: b2.ID, Amount: dec("400"), Origination: domain.OriginationBudget,
		BudgetEventID: pInt64(ev.ID), Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	_ = s.InTx(ctx, func(tx *db.Store) error {
		_, err := tx.Deployments.CreateDeployment(ctx,
			domain.Deployment{BucketID: b1.ID, Amount: dec("200"), Date: time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)},
			[]domain.DeploymentSource{{ContributionID: c1.ID, Amount: dec("200")}},
		)
		return err
	})

	summary, err := svc.GetMonthSummary(ctx, 2026, 5)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}

	if len(summary.BudgetEvents) != 1 {
		t.Errorf("expected 1 budget event, got %d", len(summary.BudgetEvents))
	}
	if len(summary.Buckets) != 2 {
		t.Fatalf("expected 2 bucket summaries, got %d", len(summary.Buckets))
	}

	var b1s service.BucketMonthSummary
	for _, bs := range summary.Buckets {
		if bs.Bucket.ID == b1.ID {
			b1s = bs
		}
	}
	if len(b1s.Contributions) != 1 {
		t.Errorf("b1 contributions: expected 1, got %d", len(b1s.Contributions))
	}
	if len(b1s.Deployments) != 1 {
		t.Errorf("b1 deployments: expected 1, got %d", len(b1s.Deployments))
	}

	var b2s service.BucketMonthSummary
	for _, bs := range summary.Buckets {
		if bs.Bucket.ID == b2.ID {
			b2s = bs
		}
	}
	if len(b2s.Contributions) != 1 {
		t.Errorf("b2 contributions: expected 1, got %d", len(b2s.Contributions))
	}
	if len(b2s.Deployments) != 0 {
		t.Errorf("b2 deployments: expected 0, got %d", len(b2s.Deployments))
	}
}
