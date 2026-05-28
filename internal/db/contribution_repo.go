package db

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/swtsn/investor/internal/domain"
)

type sqliteContributionRepo struct {
	q querier
}

func (r *sqliteContributionRepo) CreateContribution(ctx context.Context, c domain.Contribution) (domain.Contribution, error) {
	res, err := r.q.ExecContext(ctx,
		`INSERT INTO contributions (bucket_id, amount, origination, budget_event_id, date)
		 VALUES (?, ?, ?, ?, ?)`,
		c.BucketID, c.Amount.String(), string(c.Origination), c.BudgetEventID, c.Date.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return domain.Contribution{}, err
	}
	c.ID, err = res.LastInsertId()
	return c, err
}

func (r *sqliteContributionRepo) ListByBucket(ctx context.Context, bucketID int64) ([]domain.Contribution, error) {
	rows, err := r.q.QueryContext(ctx,
		`SELECT id, bucket_id, amount, origination, budget_event_id, date
		 FROM contributions WHERE bucket_id = ? ORDER BY date, id`,
		bucketID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanContributions(rows)
}

func (r *sqliteContributionRepo) ListByBucketAndMonth(ctx context.Context, bucketID int64, year, month int) ([]domain.Contribution, error) {
	start, end := monthBounds(year, month)
	rows, err := r.q.QueryContext(ctx,
		`SELECT id, bucket_id, amount, origination, budget_event_id, date
		 FROM contributions WHERE bucket_id = ? AND date >= ? AND date < ? ORDER BY date, id`,
		bucketID, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanContributions(rows)
}

func scanContributions(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]domain.Contribution, error) {
	var out []domain.Contribution
	for rows.Next() {
		var c domain.Contribution
		var amount, dateStr string
		if err := rows.Scan(&c.ID, &c.BucketID, &amount, &c.Origination, &c.BudgetEventID, &dateStr); err != nil {
			return nil, err
		}
		var err error
		if c.Amount, err = decimal.NewFromString(amount); err != nil {
			return nil, fmt.Errorf("parse amount: %w", err)
		}
		if c.Date, err = time.Parse(time.RFC3339, dateStr); err != nil {
			return nil, fmt.Errorf("parse date: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func monthBounds(year, month int) (string, string) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	return start.Format(time.RFC3339), end.Format(time.RFC3339)
}
