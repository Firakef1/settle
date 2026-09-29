package repository

import (
	"context"
	"database/sql"
	"errors"
)

// ErrNotFound means the requested row does not exist in this organization.
var ErrNotFound = errors.New("record not found")

// ErrConflict means the row changed and the expected state no longer matches.
var ErrConflict = errors.New("state conflict")

// DBTX is the subset of *sql.DB and *sql.Tx used by repositories.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
