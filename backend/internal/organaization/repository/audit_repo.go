package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Firakef1/settle/backend/internal/organaization/dto"
	"github.com/Firakef1/settle/backend/internal/organaization/model"
)

type AuditWriter interface {
	WriteTx(ctx context.Context, tx *sql.Tx, log *model.AuditLog) error
	Write(ctx context.Context, db DBTX, log *model.AuditLog) error
}

type AuditRepository interface {
	AuditWriter
	Query(ctx context.Context, db DBTX, orgID string, filters dto.AuditFilters) ([]*dto.AuditLogItem, int, error)
}

type pgAuditRepo struct{}

func NewAuditWriter() AuditWriter {
	return &pgAuditRepo{}
}

func NewAuditRepository() AuditRepository {
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

func (r *pgAuditRepo) Query(ctx context.Context, db DBTX, orgID string, filters dto.AuditFilters) ([]*dto.AuditLogItem, int, error) {
	countQuery := `
		SELECT COUNT(*)
		FROM audit_log al
		WHERE al.org_id = $1
		  AND ($2 = '' OR al.action = $2)
		  AND ($3 = '' OR al.actor_id::text = $3)
		  AND ($4::timestamptz IS NULL OR al.created_at >= $4)
		  AND ($5::timestamptz IS NULL OR al.created_at <= $5)
	`
	var total int
	err := db.QueryRowContext(ctx, countQuery, orgID, filters.Action, filters.ActorID, filters.StartDate, filters.EndDate).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := filters.Offset
	if offset < 0 {
		offset = 0
	}
	if filters.Page > 1 && offset == 0 {
		offset = (filters.Page - 1) * limit
	}

	dataQuery := `
		SELECT al.id, al.org_id, al.actor_id, COALESCE(u.name, ''), al.action, al.target_id, al.metadata, al.created_at
		FROM audit_log al
		LEFT JOIN users u ON al.actor_id = u.id
		WHERE al.org_id = $1
		  AND ($2 = '' OR al.action = $2)
		  AND ($3 = '' OR al.actor_id::text = $3)
		  AND ($4::timestamptz IS NULL OR al.created_at >= $4)
		  AND ($5::timestamptz IS NULL OR al.created_at <= $5)
		ORDER BY al.created_at DESC
		LIMIT $6 OFFSET $7
	`
	rows, err := db.QueryContext(ctx, dataQuery, orgID, filters.Action, filters.ActorID, filters.StartDate, filters.EndDate, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var items []*dto.AuditLogItem
	for rows.Next() {
		item := &dto.AuditLogItem{}
		var metaRaw []byte
		var targetID sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.OrgID,
			&item.ActorID,
			&item.ActorName,
			&item.Action,
			&targetID,
			&metaRaw,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log row: %w", err)
		}
		if targetID.Valid {
			tid := targetID.String
			item.TargetID = &tid
		}
		item.Metadata = make(map[string]any)
		if len(metaRaw) > 0 {
			_ = json.Unmarshal(metaRaw, &item.Metadata)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating audit logs: %w", err)
	}

	if items == nil {
		items = []*dto.AuditLogItem{}
	}

	return items, total, nil
}
