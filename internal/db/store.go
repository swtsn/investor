package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/swtsn/investor/internal/domain"
	_ "modernc.org/sqlite"
)

var (
	_ domain.BucketRepository       = (*sqliteBucketRepo)(nil)
	_ domain.ContributionRepository = (*sqliteContributionRepo)(nil)
	_ domain.DeploymentRepository   = (*sqliteDeploymentRepo)(nil)
	_ domain.BudgetEventRepository  = (*sqliteBudgetEventRepo)(nil)
)

// Store holds all repository implementations backed by a single SQLite connection.
type Store struct {
	db            *sql.DB
	Buckets       domain.BucketRepository
	Contributions domain.ContributionRepository
	Deployments   domain.DeploymentRepository
	BudgetEvents  domain.BudgetEventRepository
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	// SQLite is single-writer; one connection ensures pragmas set below remain in effect
	// for every subsequent query and prevents SQLITE_BUSY under concurrent goroutine access.
	db.SetMaxOpenConns(1)

	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set foreign_keys pragma: %w", err)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA journal_mode = WAL`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set journal_mode pragma: %w", err)
	}

	if err := migrate(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &Store{
		db:            db,
		Buckets:       &sqliteBucketRepo{q: db},
		Contributions: &sqliteContributionRepo{q: db},
		Deployments:   &sqliteDeploymentRepo{q: db},
		BudgetEvents:  &sqliteBudgetEventRepo{q: db},
	}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying *sql.DB. Intended for integration tests that need raw SQL access.
func (s *Store) DB() *sql.DB { return s.db }

// InTx runs fn inside a transaction. All repos on the passed *Store share the same *sql.Tx.
// Rolls back on error, commits on success.
// Must not be called on the Store passed into an InTx callback — nested InTx is not supported.
func (s *Store) InTx(ctx context.Context, fn func(*Store) error) error {
	if s.db == nil {
		panic("db.Store: InTx called on a transaction-scoped Store")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	// db is nil so the callback Store cannot start a nested transaction.
	txStore := &Store{
		db:            nil,
		Buckets:       &sqliteBucketRepo{q: tx},
		Contributions: &sqliteContributionRepo{q: tx},
		Deployments:   &sqliteDeploymentRepo{q: tx},
		BudgetEvents:  &sqliteBudgetEventRepo{q: tx},
	}
	if err := fn(txStore); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
