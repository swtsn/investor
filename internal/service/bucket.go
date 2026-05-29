package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/shopspring/decimal"
	"github.com/swtsn/investor/internal/db"
	"github.com/swtsn/investor/internal/domain"
)

type BucketService struct{ store *db.Store }

func NewBucketService(store *db.Store) *BucketService {
	return &BucketService{store: store}
}

func (s *BucketService) CreateBucket(ctx context.Context, name string, bucketType domain.BucketType, targetPct decimal.Decimal) (domain.Bucket, error) {
	if name == "" {
		return domain.Bucket{}, errors.New("name is required")
	}
	if bucketType != domain.BucketTypeFlat && bucketType != domain.BucketTypeDiversified {
		return domain.Bucket{}, errors.New("type must be flat or diversified")
	}
	if !targetPct.IsPositive() {
		return domain.Bucket{}, errors.New("target_pct must be > 0")
	}
	return s.store.Buckets.CreateBucket(ctx, domain.Bucket{
		Name:      name,
		Type:      bucketType,
		TargetPct: targetPct,
	})
}

func (s *BucketService) UpdateBucket(ctx context.Context, id int64, name string, targetPct decimal.Decimal) (domain.Bucket, error) {
	if name == "" {
		return domain.Bucket{}, errors.New("name is required")
	}
	if !targetPct.IsPositive() {
		return domain.Bucket{}, errors.New("target_pct must be > 0")
	}
	existing, err := s.store.Buckets.GetBucket(ctx, id)
	if err != nil {
		return domain.Bucket{}, err
	}
	if err := s.store.Buckets.UpdateBucket(ctx, domain.Bucket{
		ID:        id,
		Name:      name,
		Type:      existing.Type,
		TargetPct: targetPct,
	}); err != nil {
		return domain.Bucket{}, err
	}
	return s.store.Buckets.GetBucket(ctx, id)
}

func (s *BucketService) UpsertAllocation(ctx context.Context, bucketID int64, name string, targetPct decimal.Decimal) (domain.Allocation, error) {
	if name == "" {
		return domain.Allocation{}, errors.New("name is required")
	}
	if !targetPct.IsPositive() {
		return domain.Allocation{}, errors.New("target_pct must be > 0")
	}
	bucket, err := s.store.Buckets.GetBucket(ctx, bucketID)
	if err != nil {
		return domain.Allocation{}, err
	}
	if bucket.Type == domain.BucketTypeFlat {
		return domain.Allocation{}, domain.ErrBucketIsFlat
	}
	if err := s.store.Buckets.UpsertAllocation(ctx, domain.Allocation{
		BucketID:  bucketID,
		Name:      name,
		TargetPct: targetPct,
	}); err != nil {
		return domain.Allocation{}, err
	}
	allocs, err := s.store.Buckets.ListAllocations(ctx, bucketID)
	if err != nil {
		return domain.Allocation{}, err
	}
	for _, a := range allocs {
		if a.Name == name {
			return a, nil
		}
	}
	return domain.Allocation{}, domain.ErrNotFound
}

func (s *BucketService) DeleteAllocation(ctx context.Context, id int64) error {
	var exists int
	err := s.store.DB().QueryRowContext(ctx,
		`SELECT 1 FROM deployments WHERE allocation_id = ? LIMIT 1`, id).Scan(&exists)
	if err == nil {
		return domain.ErrAllocationHasDeployments
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return s.store.Buckets.DeleteAllocation(ctx, id)
}
