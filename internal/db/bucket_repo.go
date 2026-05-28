package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/swtsn/investor/internal/domain"
)

type sqliteBucketRepo struct {
	q querier
}

func (r *sqliteBucketRepo) ListBuckets(ctx context.Context) ([]domain.Bucket, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT id, name, type, target_pct FROM buckets ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []domain.Bucket
	for rows.Next() {
		var b domain.Bucket
		var targetPct string
		if err := rows.Scan(&b.ID, &b.Name, &b.Type, &targetPct); err != nil {
			return nil, err
		}
		if b.TargetPct, err = decimal.NewFromString(targetPct); err != nil {
			return nil, fmt.Errorf("parse target_pct: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *sqliteBucketRepo) GetBucket(ctx context.Context, id int64) (domain.Bucket, error) {
	var b domain.Bucket
	var targetPct string
	err := r.q.QueryRowContext(ctx, `SELECT id, name, type, target_pct FROM buckets WHERE id = ?`, id).
		Scan(&b.ID, &b.Name, &b.Type, &targetPct)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Bucket{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Bucket{}, err
	}
	if b.TargetPct, err = decimal.NewFromString(targetPct); err != nil {
		return domain.Bucket{}, fmt.Errorf("parse target_pct: %w", err)
	}
	return b, nil
}

func (r *sqliteBucketRepo) CreateBucket(ctx context.Context, b domain.Bucket) (domain.Bucket, error) {
	res, err := r.q.ExecContext(ctx,
		`INSERT INTO buckets (name, type, target_pct) VALUES (?, ?, ?)`,
		b.Name, string(b.Type), b.TargetPct.String(),
	)
	if err != nil {
		return domain.Bucket{}, err
	}
	b.ID, err = res.LastInsertId()
	return b, err
}

func (r *sqliteBucketRepo) UpdateBucket(ctx context.Context, b domain.Bucket) error {
	res, err := r.q.ExecContext(ctx,
		`UPDATE buckets SET name = ?, type = ?, target_pct = ? WHERE id = ?`,
		b.Name, string(b.Type), b.TargetPct.String(), b.ID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *sqliteBucketRepo) ListAllocations(ctx context.Context, bucketID int64) ([]domain.Allocation, error) {
	rows, err := r.q.QueryContext(ctx,
		`SELECT id, bucket_id, name, target_pct FROM allocations WHERE bucket_id = ? ORDER BY id`,
		bucketID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []domain.Allocation
	for rows.Next() {
		var a domain.Allocation
		var targetPct string
		if err := rows.Scan(&a.ID, &a.BucketID, &a.Name, &targetPct); err != nil {
			return nil, err
		}
		if a.TargetPct, err = decimal.NewFromString(targetPct); err != nil {
			return nil, fmt.Errorf("parse target_pct: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *sqliteBucketRepo) UpsertAllocation(ctx context.Context, a domain.Allocation) error {
	_, err := r.q.ExecContext(ctx,
		`INSERT INTO allocations (bucket_id, name, target_pct) VALUES (?, ?, ?)
		 ON CONFLICT(bucket_id, name) DO UPDATE SET target_pct = excluded.target_pct`,
		a.BucketID, a.Name, a.TargetPct.String(),
	)
	return err
}

func (r *sqliteBucketRepo) DeleteAllocation(ctx context.Context, id int64) error {
	res, err := r.q.ExecContext(ctx, `DELETE FROM allocations WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
