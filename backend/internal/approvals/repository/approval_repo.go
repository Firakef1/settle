package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"github.com/Firakef1/settle/backend/internal/approvals/model"
)

type pgApprovalRepo struct{}

// NewApprovalRepository returns the PostgreSQL approval repository.
func NewApprovalRepository() ApprovalRepository {
	return &pgApprovalRepo{}
}

func (r *pgApprovalRepo) Create(ctx context.Context, db DBTX, approval *model.Approval) error {
	if db == nil {
		return errors.New("database executor is nil")
	}
	const query = `
		INSERT INTO approvals (
			id, request_id, org_id, approver_id, decision, decision_note,
			payment_status, payment_method, payment_at, failure_reason, failure_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13
		)
	`
	_, err := db.ExecContext(
		ctx,
		query,
		approval.ID,
		approval.RequestID,
		approval.OrgID,
		approval.ApproverID,
		approval.Decision,
		approval.DecisionNote,
		approval.PaymentStatus,
		approval.PaymentMethod,
		approval.PaymentAt,
		approval.FailureReason,
		approval.FailureAt,
		approval.CreatedAt,
		approval.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return fmt.Errorf("insert approval: %w", err)
	}
	return nil
}

func (r *pgApprovalRepo) GetByRequestID(ctx context.Context, db DBTX, orgID, requestID string, forUpdate bool) (*model.Approval, error) {
	if db == nil {
		return nil, errors.New("database executor is nil")
	}
	query := `
		SELECT id, request_id, org_id, approver_id, decision, decision_note,
		       payment_status, payment_method, payment_at, failure_reason, failure_at,
		       created_at, updated_at
		FROM approvals
		WHERE org_id = $1 AND request_id = $2
	`
	if forUpdate {
		query += ` FOR UPDATE`
	}

	approval := &model.Approval{}
	err := db.QueryRowContext(ctx, query, orgID, requestID).Scan(
		&approval.ID,
		&approval.RequestID,
		&approval.OrgID,
		&approval.ApproverID,
		&approval.Decision,
		&approval.DecisionNote,
		&approval.PaymentStatus,
		&approval.PaymentMethod,
		&approval.PaymentAt,
		&approval.FailureReason,
		&approval.FailureAt,
		&approval.CreatedAt,
		&approval.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get approval: %w", err)
	}
	return approval, nil
}

func (r *pgApprovalRepo) UpdatePayment(ctx context.Context, db DBTX, approval *model.Approval) error {
	if db == nil {
		return errors.New("database executor is nil")
	}
	const query = `
		UPDATE approvals
		SET payment_status = $1,
		    payment_method = $2,
		    payment_at = $3,
		    failure_reason = $4,
		    failure_at = $5,
		    updated_at = $6
		WHERE id = $7 AND org_id = $8
	`
	res, err := db.ExecContext(
		ctx,
		query,
		approval.PaymentStatus,
		approval.PaymentMethod,
		approval.PaymentAt,
		approval.FailureReason,
		approval.FailureAt,
		approval.UpdatedAt,
		approval.ID,
		approval.OrgID,
	)
	if err != nil {
		return fmt.Errorf("update approval payment: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update approval payment rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
