package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

// TestAuditLogsInsertLandsInAuditLog covers the request insert
// INSERT INTO audit_logs (..., target_type, ...) landing in audit_log.metadata.
func TestAuditLogsInsertLandsInAuditLog(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Skipf("database unavailable: %v", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	const orgID = "org-audit-compat"
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO organizations (id, name, slug)
		VALUES ($1, 'Audit compat', $1)
		ON CONFLICT (id) DO NOTHING
	`, orgID); err != nil {
		t.Fatalf("insert organization: %v", err)
	}

	const (
		logID    = "AL-compat1"
		actorID  = "user-audit-compat"
		action   = "request_created"
		targetID = "REQ-abc123"
	)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO audit_logs (id, org_id, actor_id, action, target_type, target_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`, logID, orgID, actorID, action, "request", targetID)
	if err != nil {
		t.Fatalf("insert audit_logs: %v", err)
	}

	var gotAction, gotTarget, gotType string
	err = tx.QueryRowContext(ctx, `
		SELECT action, target_id, metadata->>'target_type'
		FROM audit_log
		WHERE id = $1
	`, logID).Scan(&gotAction, &gotTarget, &gotType)
	if err != nil {
		t.Fatalf("read audit_log: %v", err)
	}
	if gotAction != action || gotTarget != targetID || gotType != "request" {
		t.Fatalf("audit_log row = action %q target %q type %q", gotAction, gotTarget, gotType)
	}
}
