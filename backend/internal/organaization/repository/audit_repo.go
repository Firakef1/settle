package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Firakef1/settle/backend/internal/organaization/model"
)

type AuditWriter interface {
	WriteTx(ctx context.Context, tx *sql.Tx, log *model.AuditLog) error
	Write(ctx context.Context, db DBTX, log *model.AuditLog) error
}

type pgAuditRepo struct{}

func NewAuditWriter() AuditWriter {
	return &pgAuditRepo{}
}

func (r *pgAuditRepo) WriteTx(ctx context.Context, tx *sql.Tx, log *model.AuditLog) error {
	query := `
		INSERT INTO audit_log (id, org_id, actor_id, action, target_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7)
	`
	meta := log.Metadata
	if meta == "" {
		meta = "{}"
	}
	_, err := tx.ExecContext(ctx, query, log.ID, log.OrgID, log.ActorID, log.Action, log.TargetID, meta, log.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to write audit log in tx: %w", err)
	}
	return nil
}

func (r *pgAuditRepo) Write(ctx context.Context, db DBTX, log *model.AuditLog) error {
	query := `
		INSERT INTO audit_log (id, org_id, actor_id, action, target_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7)
	`
	meta := log.Metadata
	if meta == "" {
		meta = "{}"
	}
	_, err := db.ExecContext(ctx, query, log.ID, log.OrgID, log.ActorID, log.Action, log.TargetID, meta, log.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to write audit log: %w", err)
	}
	return nil
}
