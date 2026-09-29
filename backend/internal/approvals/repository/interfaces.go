package repository

import (
	"context"

	"github.com/Firakef1/settle/backend/internal/approvals/model"
)

// ApprovalRepository stores finance decisions. It does not query the requests table.
type ApprovalRepository interface {
	Create(ctx context.Context, db DBTX, approval *model.Approval) error
	GetByRequestID(ctx context.Context, db DBTX, orgID, requestID string, forUpdate bool) (*model.Approval, error)
	UpdatePayment(ctx context.Context, db DBTX, approval *model.Approval) error
}

// RequestReader is the injected contract for request status.
// The requests domain can replace the SQL implementation without changing this service.
type RequestReader interface {
	GetByID(ctx context.Context, db DBTX, orgID, requestID string, forUpdate bool) (*model.RequestSnapshot, error)
	UpdateStatus(ctx context.Context, db DBTX, orgID, requestID, fromStatus, toStatus string) error
}

// MemberReader resolves the caller's active role inside an organization.
type MemberReader interface {
	ActiveRole(ctx context.Context, db DBTX, orgID, userID string) (string, error)
}

// AuditWriter appends an immutable audit row on the caller's transaction.
type AuditWriter interface {
	Write(ctx context.Context, db DBTX, entry model.AuditEntry) error
}
