package db

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/swtsn/investor/internal/domain"
)

type sqliteBudgetEventRepo struct {
	q querier
}

func (r *sqliteBudgetEventRepo) CreateBudgetEvent(ctx context.Context, e domain.BudgetEvent) (domain.BudgetEvent, error) {
	res, err := r.q.ExecContext(ctx,
		`INSERT INTO budget_events (total_amount, date) VALUES (?, ?)`,
		e.TotalAmount.String(), e.Date.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return domain.BudgetEvent{}, err
	}
	e.ID, err = res.LastInsertId()
	return e, err
}

func (r *sqliteBudgetEventRepo) ListByMonth(ctx context.Context, year, month int) ([]domain.BudgetEvent, error) {
	start, end := monthBounds(year, month)
	rows, err := r.q.QueryContext(ctx,
		`SELECT id, total_amount, date FROM budget_events WHERE date >= ? AND date < ? ORDER BY date, id`,
		start, end,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []domain.BudgetEvent
	for rows.Next() {
		var e domain.BudgetEvent
		var totalAmount, dateStr string
		if err := rows.Scan(&e.ID, &totalAmount, &dateStr); err != nil {
			return nil, err
		}
		var parseErr error
		if e.TotalAmount, parseErr = decimal.NewFromString(totalAmount); parseErr != nil {
			return nil, fmt.Errorf("parse total_amount: %w", parseErr)
		}
		if e.Date, parseErr = time.Parse(time.RFC3339, dateStr); parseErr != nil {
			return nil, fmt.Errorf("parse date: %w", parseErr)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
