package service

import (
	"context"
	"database/sql"
	"errors"
)

// TxRunner executes a function inside a database transaction.
type TxRunner interface {
	WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error
}

// SQLTxRunner implements TxRunner with *sql.DB.
type SQLTxRunner struct {
	db *sql.DB
}

// NewSQLTxRunner returns a transaction runner for the given pool.
func NewSQLTxRunner(db *sql.DB) *SQLTxRunner {
	return &SQLTxRunner{db: db}
}

// WithTx commits when fn returns nil and rolls back otherwise.
func (r *SQLTxRunner) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	if r == nil || r.db == nil {
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
