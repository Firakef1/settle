package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Firakef1/settle/backend/internal/approvals/model"
)

type pgRequestReader struct{}

// NewRequestReader returns a SQL RequestReader limited to status reads and updates.
func NewRequestReader() RequestReader {
	return &pgRequestReader{}
}

func (r *pgRequestReader) GetByID(ctx context.Context, db DBTX, orgID, requestID string, forUpdate bool) (*model.RequestSnapshot, error) {
	if db == nil {
		return nil, errors.New("database executor is nil")
	}
	query := `
		SELECT id, org_id, status
		FROM requests
		WHERE id = $1 AND org_id = $2
	`
	if forUpdate {
		query += ` FOR UPDATE`
	}

	snapshot := &model.RequestSnapshot{}
	err := db.QueryRowContext(ctx, query, requestID, orgID).Scan(&snapshot.ID, &snapshot.OrgID, &snapshot.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get request: %w", err)
	}
	return snapshot, nil
}

func (r *pgRequestReader) UpdateStatus(ctx context.Context, db DBTX, orgID, requestID, fromStatus, toStatus string) error {
	if db == nil {
		return errors.New("database executor is nil")
	}
	const query = `
		UPDATE requests
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND org_id = $3 AND status = $4
	`
	res, err := db.ExecContext(ctx, query, toStatus, requestID, orgID, fromStatus)
	if err != nil {
		return fmt.Errorf("update request status: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update request status rows: %w", err)
	}
	if n == 0 {
		return ErrConflict
	}
	return nil
}
