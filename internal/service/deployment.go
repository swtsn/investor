package service

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/swtsn/investor/internal/db"
	"github.com/swtsn/investor/internal/domain"
)

type AllocationSplit struct {
	Allocation domain.Allocation
	Amount     decimal.Decimal
}

type DeploymentService struct{ store *db.Store }

func NewDeploymentService(store *db.Store) *DeploymentService {
	return &DeploymentService{store: store}
}

// PreviewDeployment splits amount by the bucket's allocation percentages. Read-only.
// Returns an error for flat buckets.
// Truncates each allocation to 2dp; last allocation absorbs the remainder.
func (s *DeploymentService) PreviewDeployment(ctx context.Context, bucketID int64, amount decimal.Decimal) ([]AllocationSplit, error) {
	bucket, err := s.store.Buckets.GetBucket(ctx, bucketID)
	if err != nil {
		return nil, err
	}
	if bucket.Type == domain.BucketTypeFlat {
		return nil, fmt.Errorf("cannot preview deployment split for flat bucket")
	}
	allocs, err := s.store.Buckets.ListAllocations(ctx, bucketID)
	if err != nil {
		return nil, err
	}
	return splitAllocations(allocs, amount), nil
}

// RecordDeployment validates and persists a Deployment with its sources in a single transaction.
// Validation order: GetBucket → d.Validate(bucketType) → Σ sources == d.Amount →
// GetContribution per source (verify each belongs to d.BucketID) → InTx write.
func (s *DeploymentService) RecordDeployment(ctx context.Context, d domain.Deployment, sources []domain.DeploymentSource) (domain.Deployment, error) {
	bucket, err := s.store.Buckets.GetBucket(ctx, d.BucketID)
	if err != nil {
		return domain.Deployment{}, err
	}
	if err := d.Validate(bucket.Type); err != nil {
		return domain.Deployment{}, err
	}

	var sourceSum decimal.Decimal
	for _, src := range sources {
		sourceSum = sourceSum.Add(src.Amount)
	}
	if !sourceSum.Equal(d.Amount) {
		return domain.Deployment{}, fmt.Errorf("sum of source amounts %s does not equal deployment amount %s", sourceSum, d.Amount)
	}

	for _, src := range sources {
		if err := src.Validate(); err != nil {
			return domain.Deployment{}, fmt.Errorf("invalid source: %w", err)
		}
		c, err := s.store.Contributions.GetContribution(ctx, src.ContributionID)
		if err != nil {
			return domain.Deployment{}, err
		}
		if c.BucketID != d.BucketID {
			return domain.Deployment{}, fmt.Errorf("contribution %d belongs to bucket %d, not bucket %d", src.ContributionID, c.BucketID, d.BucketID)
		}
	}

	var result domain.Deployment
	err = s.store.InTx(ctx, func(tx *db.Store) error {
		var txErr error
		result, txErr = tx.Deployments.CreateDeployment(ctx, d, sources)
		return txErr
	})
	if err != nil {
		return domain.Deployment{}, err
	}
	return result, nil
}

// RecordReinvestment writes a Contribution with origination=reinvestment and BudgetEventID=nil.
func (s *DeploymentService) RecordReinvestment(ctx context.Context, bucketID int64, amount decimal.Decimal, date time.Time) (domain.Contribution, error) {
	_, err := s.store.Buckets.GetBucket(ctx, bucketID)
	if err != nil {
		return domain.Contribution{}, err
	}
	c := domain.Contribution{
		BucketID:    bucketID,
		Amount:      amount,
		Origination: domain.OriginationReinvestment,
		Date:        date,
	}
	if err := c.Validate(); err != nil {
		return domain.Contribution{}, err
	}
	return s.store.Contributions.CreateContribution(ctx, c)
}

// splitAllocations truncates each allocation's share to 2dp; last allocation absorbs the remainder.
func splitAllocations(allocs []domain.Allocation, totalAmount decimal.Decimal) []AllocationSplit {
	hundred := decimal.NewFromInt(100)
	result := make([]AllocationSplit, len(allocs))
	var sum decimal.Decimal
	for i, a := range allocs {
		if i < len(allocs)-1 {
			amt := totalAmount.Mul(a.TargetPct).Div(hundred).Truncate(2)
			result[i] = AllocationSplit{Allocation: a, Amount: amt}
			sum = sum.Add(amt)
		} else {
			result[i] = AllocationSplit{Allocation: a, Amount: totalAmount.Sub(sum)}
		}
	}
	return result
}
