package service

import (
	"context"
	"database/sql"
	"errors"
)

// TxRunner executes a function within a transaction.
type TxRunner interface {
	WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error
}

// SQLTxRunner implements TxRunner using standard *sql.DB.
type SQLTxRunner struct {
	db *sql.DB
}

// NewSQLTxRunner returns a new SQLTxRunner.
func NewSQLTxRunner(db *sql.DB) *SQLTxRunner {
	return &SQLTxRunner{db: db}
}

// WithTx runs fn within a transaction, committing on success and rolling back on error.
func (r *SQLTxRunner) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	if r.db == nil {
		return errors.New("database connection is nil")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// MockTxRunner executes fn directly for testing without a real database.
type MockTxRunner struct{}

func (m *MockTxRunner) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	return fn(nil)
}
