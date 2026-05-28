package db

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/swtsn/investor/internal/domain"
)

type sqliteDeploymentRepo struct {
	q querier
}

// CreateDeployment inserts the deployment and all sources. No internal transaction — caller must wrap in InTx.
func (r *sqliteDeploymentRepo) CreateDeployment(ctx context.Context, d domain.Deployment, sources []domain.DeploymentSource) (domain.Deployment, error) {
	var shares, pricePerShare *string
	if d.Shares != nil {
		s := d.Shares.String()
		shares = &s
	}
	if d.PricePerShare != nil {
		p := d.PricePerShare.String()
		pricePerShare = &p
	}

	res, err := r.q.ExecContext(ctx,
		`INSERT INTO deployments (bucket_id, allocation_id, symbol, shares, price_per_share, amount, date)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		d.BucketID, d.AllocationID, d.Symbol, shares, pricePerShare,
		d.Amount.String(), d.Date.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return domain.Deployment{}, err
	}
	d.ID, err = res.LastInsertId()
	if err != nil {
		return domain.Deployment{}, err
	}

	for _, src := range sources {
		src.DeploymentID = d.ID
		if _, err := r.q.ExecContext(ctx,
			`INSERT INTO deployment_sources (deployment_id, contribution_id, amount) VALUES (?, ?, ?)`,
			src.DeploymentID, src.ContributionID, src.Amount.String(),
		); err != nil {
			return domain.Deployment{}, fmt.Errorf("insert deployment_source: %w", err)
		}
	}
	return d, nil
}

func (r *sqliteDeploymentRepo) ListByBucket(ctx context.Context, bucketID int64) ([]domain.Deployment, error) {
	rows, err := r.q.QueryContext(ctx,
		`SELECT id, bucket_id, allocation_id, symbol, shares, price_per_share, amount, date
		 FROM deployments WHERE bucket_id = ? ORDER BY date, id`,
		bucketID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanDeployments(rows)
}

func (r *sqliteDeploymentRepo) ListByBucketAndMonth(ctx context.Context, bucketID int64, year, month int) ([]domain.Deployment, error) {
	start, end := monthBounds(year, month)
	rows, err := r.q.QueryContext(ctx,
		`SELECT id, bucket_id, allocation_id, symbol, shares, price_per_share, amount, date
		 FROM deployments WHERE bucket_id = ? AND date >= ? AND date < ? ORDER BY date, id`,
		bucketID, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanDeployments(rows)
}

func (r *sqliteDeploymentRepo) ListSourcesByDeployment(ctx context.Context, deploymentID int64) ([]domain.DeploymentSource, error) {
	rows, err := r.q.QueryContext(ctx,
		`SELECT deployment_id, contribution_id, amount FROM deployment_sources WHERE deployment_id = ?`,
		deploymentID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []domain.DeploymentSource
	for rows.Next() {
		var s domain.DeploymentSource
		var amount string
		if err := rows.Scan(&s.DeploymentID, &s.ContributionID, &amount); err != nil {
			return nil, err
		}
		var parseErr error
		if s.Amount, parseErr = decimal.NewFromString(amount); parseErr != nil {
			return nil, fmt.Errorf("parse amount: %w", parseErr)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func scanDeployments(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]domain.Deployment, error) {
	var out []domain.Deployment
	for rows.Next() {
		var d domain.Deployment
		var amount, dateStr string
		var shares, pricePerShare *string
		if err := rows.Scan(&d.ID, &d.BucketID, &d.AllocationID, &d.Symbol, &shares, &pricePerShare, &amount, &dateStr); err != nil {
			return nil, err
		}
		var err error
		if d.Amount, err = decimal.NewFromString(amount); err != nil {
			return nil, fmt.Errorf("parse amount: %w", err)
		}
		if d.Date, err = time.Parse(time.RFC3339, dateStr); err != nil {
			return nil, fmt.Errorf("parse date: %w", err)
		}
		if shares != nil {
			v, err := decimal.NewFromString(*shares)
			if err != nil {
				return nil, fmt.Errorf("parse shares: %w", err)
			}
			d.Shares = &v
		}
		if pricePerShare != nil {
			v, err := decimal.NewFromString(*pricePerShare)
			if err != nil {
				return nil, fmt.Errorf("parse price_per_share: %w", err)
			}
			d.PricePerShare = &v
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
