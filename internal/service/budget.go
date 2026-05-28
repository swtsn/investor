package service

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/swtsn/investor/internal/db"
	"github.com/swtsn/investor/internal/domain"
)

type BucketAllocation struct {
	Bucket domain.Bucket
	Amount decimal.Decimal
}

type BudgetService struct{ store *db.Store }

func NewBudgetService(store *db.Store) *BudgetService {
	return &BudgetService{store: store}
}

// PreviewBudget splits totalAmount by each bucket's TargetPct. Read-only.
// Truncates each bucket to 2dp; last bucket absorbs the remainder.
func (s *BudgetService) PreviewBudget(ctx context.Context, totalAmount decimal.Decimal) ([]BucketAllocation, error) {
	buckets, err := s.store.Buckets.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}
	return splitByTargetPct(buckets, totalAmount), nil
}

// ApplyBudget writes a BudgetEvent and per-bucket Contributions in a single transaction.
// On the first ApplyBudget of a new month (month-opening), a slush Contribution is written
// per bucket if the running pool balance is positive. Subsequent calls in the same month
// (top-up) skip slush and write only budget Contributions.
func (s *BudgetService) ApplyBudget(ctx context.Context, totalAmount decimal.Decimal, date time.Time) (domain.BudgetEvent, error) {
	// Check month-opening before opening the transaction.
	existing, err := s.store.BudgetEvents.ListByMonth(ctx, date.Year(), int(date.Month()))
	if err != nil {
		return domain.BudgetEvent{}, err
	}
	isMonthOpening := len(existing) == 0

	buckets, err := s.store.Buckets.ListBuckets(ctx)
	if err != nil {
		return domain.BudgetEvent{}, err
	}
	allocs := splitByTargetPct(buckets, totalAmount)

	var balances map[int64]decimal.Decimal
	if isMonthOpening {
		balances = make(map[int64]decimal.Decimal, len(buckets))
		for _, b := range buckets {
			bal, err := poolBalance(ctx, s.store, b.ID)
			if err != nil {
				return domain.BudgetEvent{}, err
			}
			balances[b.ID] = bal
		}
	}

	var event domain.BudgetEvent
	err = s.store.InTx(ctx, func(tx *db.Store) error {
		var txErr error
		event, txErr = tx.BudgetEvents.CreateBudgetEvent(ctx, domain.BudgetEvent{
			TotalAmount: totalAmount,
			Date:        date,
		})
		if txErr != nil {
			return txErr
		}

		for _, alloc := range allocs {
			if isMonthOpening {
				if bal := balances[alloc.Bucket.ID]; bal.IsPositive() {
					_, txErr = tx.Contributions.CreateContribution(ctx, domain.Contribution{
						BucketID:      alloc.Bucket.ID,
						Amount:        bal,
						Origination:   domain.OriginationSlush,
						BudgetEventID: &event.ID,
						Date:          date,
					})
					if txErr != nil {
						return txErr
					}
				}
			}
			_, txErr = tx.Contributions.CreateContribution(ctx, domain.Contribution{
				BucketID:      alloc.Bucket.ID,
				Amount:        alloc.Amount,
				Origination:   domain.OriginationBudget,
				BudgetEventID: &event.ID,
				Date:          date,
			})
			if txErr != nil {
				return txErr
			}
		}
		return nil
	})
	if err != nil {
		return domain.BudgetEvent{}, err
	}
	return event, nil
}

// splitByTargetPct truncates each bucket's share to 2dp; last bucket absorbs the remainder.
func splitByTargetPct(buckets []domain.Bucket, totalAmount decimal.Decimal) []BucketAllocation {
	hundred := decimal.NewFromInt(100)
	result := make([]BucketAllocation, len(buckets))
	var sum decimal.Decimal
	for i, b := range buckets {
		if i < len(buckets)-1 {
			amt := totalAmount.Mul(b.TargetPct).Div(hundred).Truncate(2)
			result[i] = BucketAllocation{Bucket: b, Amount: amt}
			sum = sum.Add(amt)
		} else {
			result[i] = BucketAllocation{Bucket: b, Amount: totalAmount.Sub(sum)}
		}
	}
	return result
}
