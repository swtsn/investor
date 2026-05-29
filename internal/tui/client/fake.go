package client

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/swtsn/investor/internal/domain"
)

// Fake is an in-memory Client for use in tests.
type Fake struct {
	Buckets           []domain.Bucket
	Allocations       map[int64][]domain.Allocation
	Dashboard         []BucketDashboard
	Summary           MonthSummary
	DeployableSources map[int64][]DeployableSource
	BudgetPreview     []BucketAllocation
	DeployPreview     map[int64][]AllocationSplit

	// Err causes all calls to return this error if non-nil.
	Err error
}

func (f *Fake) ListBuckets(_ context.Context) ([]domain.Bucket, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Buckets, nil
}

func (f *Fake) ListAllocations(_ context.Context, bucketID int64) ([]domain.Allocation, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Allocations[bucketID], nil
}

func (f *Fake) GetDashboard(_ context.Context) ([]BucketDashboard, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Dashboard, nil
}

func (f *Fake) GetMonthSummary(_ context.Context, _, _ int) (MonthSummary, error) {
	if f.Err != nil {
		return MonthSummary{}, f.Err
	}
	return f.Summary, nil
}

func (f *Fake) ListDeployableSources(_ context.Context, bucketID int64) ([]DeployableSource, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	return f.DeployableSources[bucketID], nil
}

func (f *Fake) PreviewBudget(_ context.Context, _ decimal.Decimal) ([]BucketAllocation, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	return f.BudgetPreview, nil
}

func (f *Fake) ApplyBudget(_ context.Context, _ decimal.Decimal, _ time.Time) error {
	return f.Err
}

func (f *Fake) PreviewDeployment(_ context.Context, bucketID int64, _ decimal.Decimal) ([]AllocationSplit, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	return f.DeployPreview[bucketID], nil
}

func (f *Fake) RecordDeployment(_ context.Context, _ domain.Deployment, _ []domain.DeploymentSource) error {
	return f.Err
}

func (f *Fake) RecordReinvestment(_ context.Context, _ int64, _ decimal.Decimal, _ time.Time) error {
	return f.Err
}

func (f *Fake) CreateBucket(_ context.Context, _ string, _ domain.BucketType, _ decimal.Decimal) (domain.Bucket, error) {
	return domain.Bucket{}, f.Err
}

func (f *Fake) UpdateBucket(_ context.Context, _ int64, _ string, _ decimal.Decimal) (domain.Bucket, error) {
	return domain.Bucket{}, f.Err
}

func (f *Fake) UpsertAllocation(_ context.Context, _ int64, _ string, _ decimal.Decimal) (domain.Allocation, error) {
	return domain.Allocation{}, f.Err
}

func (f *Fake) DeleteAllocation(_ context.Context, _ int64) error {
	return f.Err
}

func (f *Fake) Close() error { return nil }

// Compile-time check that Fake implements Client.
var _ Client = (*Fake)(nil)
