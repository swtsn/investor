package service

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/swtsn/investor/internal/db"
	"github.com/swtsn/investor/internal/domain"
)

// poolBalance computes Σ contributions − Σ deployments for a bucket.
// Shared by BudgetService and PoolService.
func poolBalance(ctx context.Context, store *db.Store, bucketID int64) (decimal.Decimal, error) {
	contribs, err := store.Contributions.ListByBucket(ctx, bucketID)
	if err != nil {
		return decimal.Zero, err
	}
	deps, err := store.Deployments.ListByBucket(ctx, bucketID)
	if err != nil {
		return decimal.Zero, err
	}
	var total decimal.Decimal
	for _, c := range contribs {
		total = total.Add(c.Amount)
	}
	for _, d := range deps {
		total = total.Sub(d.Amount)
	}
	return total, nil
}

type PoolBreakdown struct {
	Budget       decimal.Decimal
	Reinvestment decimal.Decimal
	Slush        decimal.Decimal
	Total        decimal.Decimal
}

type BucketMonthSummary struct {
	Bucket        domain.Bucket
	Contributions []domain.Contribution
	Deployments   []domain.Deployment
}

type MonthSummary struct {
	BudgetEvents []domain.BudgetEvent
	Buckets      []BucketMonthSummary
}

type PoolService struct{ store *db.Store }

func NewPoolService(store *db.Store) *PoolService {
	return &PoolService{store: store}
}

func (s *PoolService) GetPoolBalance(ctx context.Context, bucketID int64) (decimal.Decimal, error) {
	return poolBalance(ctx, s.store, bucketID)
}

func (s *PoolService) GetPoolBreakdown(ctx context.Context, bucketID int64, year, month int) (PoolBreakdown, error) {
	contribs, err := s.store.Contributions.ListByBucketAndMonth(ctx, bucketID, year, month)
	if err != nil {
		return PoolBreakdown{}, err
	}
	var bd PoolBreakdown
	for _, c := range contribs {
		switch c.Origination {
		case domain.OriginationBudget:
			bd.Budget = bd.Budget.Add(c.Amount)
		case domain.OriginationReinvestment:
			bd.Reinvestment = bd.Reinvestment.Add(c.Amount)
		case domain.OriginationSlush:
			bd.Slush = bd.Slush.Add(c.Amount)
		default:
			return PoolBreakdown{}, fmt.Errorf("unknown origination type: %s", c.Origination)
		}
	}
	bd.Total = bd.Budget.Add(bd.Reinvestment).Add(bd.Slush)
	return bd, nil
}

func (s *PoolService) GetMonthSummary(ctx context.Context, year, month int) (MonthSummary, error) {
	events, err := s.store.BudgetEvents.ListByMonth(ctx, year, month)
	if err != nil {
		return MonthSummary{}, err
	}
	buckets, err := s.store.Buckets.ListBuckets(ctx)
	if err != nil {
		return MonthSummary{}, err
	}
	summary := MonthSummary{BudgetEvents: events}
	for _, b := range buckets {
		contribs, err := s.store.Contributions.ListByBucketAndMonth(ctx, b.ID, year, month)
		if err != nil {
			return MonthSummary{}, err
		}
		deps, err := s.store.Deployments.ListByBucketAndMonth(ctx, b.ID, year, month)
		if err != nil {
			return MonthSummary{}, err
		}
		summary.Buckets = append(summary.Buckets, BucketMonthSummary{
			Bucket:        b,
			Contributions: contribs,
			Deployments:   deps,
		})
	}
	return summary, nil
}
