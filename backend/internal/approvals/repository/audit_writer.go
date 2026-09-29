package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Firakef1/settle/backend/internal/approvals/model"
)

type pgAuditWriter struct{}

// NewAuditWriter returns a SQL writer for the existing audit_log table.
func NewAuditWriter() AuditWriter {
	return &pgAuditWriter{}
}

func (w *pgAuditWriter) Write(ctx context.Context, db DBTX, entry model.AuditEntry) error {
	if db == nil {
		return errors.New("database executor is nil")
	}
	meta := entry.Metadata
	if meta == "" {
		meta = "{}"
	}
	const query = `
		INSERT INTO audit_log (id, org_id, actor_id, action, target_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7)
	`
	_, err := db.ExecContext(ctx, query, entry.ID, entry.OrgID, entry.ActorID, entry.Action, entry.TargetID, meta, entry.CreatedAt)
	if err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return nil
}
