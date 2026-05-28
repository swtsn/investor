package db_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/swtsn/investor/internal/db"
	"github.com/swtsn/investor/internal/domain"
)

func newTestStore(t *testing.T) *db.Store {
	t.Helper()
	s, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

var ctx = context.Background()

func dec(s string) decimal.Decimal {
	return decimal.RequireFromString(s)
}

func pDec(s string) *decimal.Decimal {
	v := dec(s)
	return &v
}

func pStr(s string) *string { return &s }

func pInt64(i int64) *int64 { return &i }

// --- Buckets ---

func TestBucketCRUD(t *testing.T) {
	s := newTestStore(t)

	b, err := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "Investing", Type: domain.BucketTypeDiversified, TargetPct: dec("60")})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if b.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	got, err := s.Buckets.GetBucket(ctx, b.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "Investing" || got.Type != domain.BucketTypeDiversified || !got.TargetPct.Equal(dec("60")) {
		t.Errorf("unexpected bucket: %+v", got)
	}

	b.Name = "Investing Updated"
	b.TargetPct = dec("70")
	if err := s.Buckets.UpdateBucket(ctx, b); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ = s.Buckets.GetBucket(ctx, b.ID)
	if got.Name != "Investing Updated" || !got.TargetPct.Equal(dec("70")) {
		t.Errorf("update not persisted: %+v", got)
	}

	list, err := s.Buckets.ListBuckets(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 bucket, got %d", len(list))
	}
}

func TestGetBucketNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Buckets.GetBucket(ctx, 999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestAllocationUpsertAndDelete(t *testing.T) {
	s := newTestStore(t)
	b, _ := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "Diversified", Type: domain.BucketTypeDiversified, TargetPct: dec("100")})

	if err := s.Buckets.UpsertAllocation(ctx, domain.Allocation{BucketID: b.ID, Name: "metals", TargetPct: dec("30")}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := s.Buckets.UpsertAllocation(ctx, domain.Allocation{BucketID: b.ID, Name: "metals", TargetPct: dec("40")}); err != nil {
		t.Fatalf("upsert update: %v", err)
	}

	allocs, err := s.Buckets.ListAllocations(ctx, b.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(allocs) != 1 || !allocs[0].TargetPct.Equal(dec("40")) {
		t.Errorf("unexpected allocations: %+v", allocs)
	}

	if err := s.Buckets.DeleteAllocation(ctx, allocs[0].ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	allocs, _ = s.Buckets.ListAllocations(ctx, b.ID)
	if len(allocs) != 0 {
		t.Errorf("expected 0 after delete, got %d", len(allocs))
	}
}

// --- Contributions ---

func TestContributionCRUD(t *testing.T) {
	s := newTestStore(t)
	b, _ := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "B", Type: domain.BucketTypeFlat, TargetPct: dec("100")})
	ev, _ := s.BudgetEvents.CreateBudgetEvent(ctx, domain.BudgetEvent{TotalAmount: dec("1000"), Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)})

	c, err := s.Contributions.CreateContribution(ctx, domain.Contribution{
		BucketID:      b.ID,
		Amount:        dec("500"),
		Origination:   domain.OriginationBudget,
		BudgetEventID: pInt64(ev.ID),
		Date:          time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if c.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	list, err := s.Contributions.ListByBucket(ctx, b.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || !list[0].Amount.Equal(dec("500")) {
		t.Errorf("unexpected: %+v", list)
	}

	list, _ = s.Contributions.ListByBucketAndMonth(ctx, b.ID, 2026, 5)
	if len(list) != 1 {
		t.Errorf("expected 1 in month 2026-05, got %d", len(list))
	}
	list, _ = s.Contributions.ListByBucketAndMonth(ctx, b.ID, 2026, 6)
	if len(list) != 0 {
		t.Errorf("expected 0 in month 2026-06, got %d", len(list))
	}
}

// --- BudgetEvents ---

func TestBudgetEventCRUD(t *testing.T) {
	s := newTestStore(t)
	ev, err := s.BudgetEvents.CreateBudgetEvent(ctx, domain.BudgetEvent{
		TotalAmount: dec("2000"),
		Date:        time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if ev.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	list, _ := s.BudgetEvents.ListByMonth(ctx, 2026, 5)
	if len(list) != 1 || !list[0].TotalAmount.Equal(dec("2000")) {
		t.Errorf("unexpected: %+v", list)
	}
	list, _ = s.BudgetEvents.ListByMonth(ctx, 2026, 4)
	if len(list) != 0 {
		t.Errorf("expected 0 in april, got %d", len(list))
	}
}

// --- Deployments ---

func TestDeploymentCreateAndList(t *testing.T) {
	s := newTestStore(t)
	b, _ := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "B", Type: domain.BucketTypeFlat, TargetPct: dec("100")})
	ev, _ := s.BudgetEvents.CreateBudgetEvent(ctx, domain.BudgetEvent{TotalAmount: dec("1000"), Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)})
	c, _ := s.Contributions.CreateContribution(ctx, domain.Contribution{
		BucketID: b.ID, Amount: dec("500"), Origination: domain.OriginationBudget,
		BudgetEventID: pInt64(ev.ID), Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})

	var d domain.Deployment
	var sources []domain.DeploymentSource
	err := s.InTx(ctx, func(tx *db.Store) error {
		var err error
		d, err = tx.Deployments.CreateDeployment(ctx,
			domain.Deployment{BucketID: b.ID, Amount: dec("200"), Date: time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)},
			[]domain.DeploymentSource{{ContributionID: c.ID, Amount: dec("200")}},
		)
		return err
	})
	if err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	if d.ID == 0 {
		t.Fatal("expected non-zero deployment ID")
	}

	sources, err = s.Deployments.ListSourcesByDeployment(ctx, d.ID)
	if err != nil {
		t.Fatalf("list sources: %v", err)
	}
	if len(sources) != 1 || !sources[0].Amount.Equal(dec("200")) {
		t.Errorf("unexpected sources: %+v", sources)
	}

	list, _ := s.Deployments.ListByBucket(ctx, b.ID)
	if len(list) != 1 {
		t.Errorf("expected 1 deployment, got %d", len(list))
	}

	list, _ = s.Deployments.ListByBucketAndMonth(ctx, b.ID, 2026, 5)
	if len(list) != 1 {
		t.Errorf("expected 1 in month 2026-05")
	}
	list, _ = s.Deployments.ListByBucketAndMonth(ctx, b.ID, 2026, 6)
	if len(list) != 0 {
		t.Errorf("expected 0 in month 2026-06")
	}
}

func TestDeploymentWithSharesAndSymbol(t *testing.T) {
	s := newTestStore(t)
	b, _ := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "Div", Type: domain.BucketTypeDiversified, TargetPct: dec("100")})
	_ = s.Buckets.UpsertAllocation(ctx, domain.Allocation{BucketID: b.ID, Name: "metals", TargetPct: dec("100")})
	allocs, _ := s.Buckets.ListAllocations(ctx, b.ID)
	ev, _ := s.BudgetEvents.CreateBudgetEvent(ctx, domain.BudgetEvent{TotalAmount: dec("1000"), Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)})
	c, _ := s.Contributions.CreateContribution(ctx, domain.Contribution{
		BucketID: b.ID, Amount: dec("1000"), Origination: domain.OriginationBudget,
		BudgetEventID: pInt64(ev.ID), Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})

	err := s.InTx(ctx, func(tx *db.Store) error {
		_, err := tx.Deployments.CreateDeployment(ctx,
			domain.Deployment{
				BucketID:      b.ID,
				AllocationID:  pInt64(allocs[0].ID),
				Symbol:        pStr("GLD"),
				Shares:        pDec("10"),
				PricePerShare: pDec("100"),
				Amount:        dec("1000"),
				Date:          time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC),
			},
			[]domain.DeploymentSource{{ContributionID: c.ID, Amount: dec("1000")}},
		)
		return err
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, _ := s.Deployments.ListByBucket(ctx, b.ID)
	if len(got) != 1 {
		t.Fatalf("expected 1")
	}
	if got[0].Symbol == nil || *got[0].Symbol != "GLD" {
		t.Errorf("symbol mismatch: %v", got[0].Symbol)
	}
	if got[0].Shares == nil || !got[0].Shares.Equal(dec("10")) {
		t.Errorf("shares mismatch: %v", got[0].Shares)
	}
}

// --- InTx rollback ---

func TestInTxRollback(t *testing.T) {
	s := newTestStore(t)
	b, _ := s.Buckets.CreateBucket(ctx, domain.Bucket{Name: "B", Type: domain.BucketTypeFlat, TargetPct: dec("100")})
	ev, _ := s.BudgetEvents.CreateBudgetEvent(ctx, domain.BudgetEvent{TotalAmount: dec("1000"), Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)})
	c, _ := s.Contributions.CreateContribution(ctx, domain.Contribution{
		BucketID: b.ID, Amount: dec("500"), Origination: domain.OriginationBudget,
		BudgetEventID: pInt64(ev.ID), Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})

	sentinelErr := errors.New("abort")
	err := s.InTx(ctx, func(tx *db.Store) error {
		_, _ = tx.Deployments.CreateDeployment(ctx,
			domain.Deployment{BucketID: b.ID, Amount: dec("100"), Date: time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)},
			[]domain.DeploymentSource{{ContributionID: c.ID, Amount: dec("100")}},
		)
		return sentinelErr
	})
	if !errors.Is(err, sentinelErr) {
		t.Fatalf("expected sentinelErr, got %v", err)
	}

	list, _ := s.Deployments.ListByBucket(ctx, b.ID)
	if len(list) != 0 {
		t.Errorf("deployment should have been rolled back, got %d", len(list))
	}
}
