package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Firakef1/settle/backend/internal/approvals/dto"
	"github.com/Firakef1/settle/backend/internal/approvals/model"
	"github.com/Firakef1/settle/backend/internal/approvals/repository"
	"github.com/Firakef1/settle/backend/internal/approvals/validator"
)

// ApprovalService is the finance decision engine.
type ApprovalService interface {
	Approve(ctx context.Context, orgID, requestID, approverID, note string) (*dto.ApprovalResponse, error)
	Reject(ctx context.Context, orgID, requestID, approverID, reason string) (*dto.ApprovalResponse, error)
	MarkPaid(ctx context.Context, orgID, requestID, approverID, method string) (*dto.ApprovalResponse, error)
	MarkFailed(ctx context.Context, orgID, requestID, approverID, reason string) (*dto.ApprovalResponse, error)
}

type approvalService struct {
	readDB    repository.DBTX
	tx        TxRunner
	approvals repository.ApprovalRepository
	requests  repository.RequestReader
	members   repository.MemberReader
	audit     repository.AuditWriter
}

// NewApprovalService injects the approval repositories and shared readers.
func NewApprovalService(
	db *sql.DB,
	approvals repository.ApprovalRepository,
	requests repository.RequestReader,
	members repository.MemberReader,
	audit repository.AuditWriter,
) ApprovalService {
	var runner TxRunner
	if db != nil {
		runner = NewSQLTxRunner(db)
	}
	return &approvalService{
		readDB:    db,
		tx:        runner,
		approvals: approvals,
		requests:  requests,
		members:   members,
		audit:     audit,
	}
}

// NewApprovalServiceWithRunner injects a transaction runner. Tests use this.
func NewApprovalServiceWithRunner(
	runner TxRunner,
	readDB repository.DBTX,
	approvals repository.ApprovalRepository,
	requests repository.RequestReader,
	members repository.MemberReader,
	audit repository.AuditWriter,
) ApprovalService {
	return &approvalService{
		readDB:    readDB,
		tx:        runner,
		approvals: approvals,
		requests:  requests,
		members:   members,
		audit:     audit,
	}
}

func (s *approvalService) Approve(ctx context.Context, orgID, requestID, approverID, note string) (*dto.ApprovalResponse, error) {
	if err := s.authorize(ctx, orgID, approverID); err != nil {
		return nil, err
	}
	note = strings.TrimSpace(note)

	var response *dto.ApprovalResponse
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		req, err := s.requests.GetByID(ctx, tx, orgID, requestID, true)
		if err != nil {
			return mapRequestErr(err)
		}
		if req.Status != model.StatusPending {
			return ErrNotPending
		}

		now := time.Now().UTC()
		approval := &model.Approval{
			ID:            uuid.New().String(),
			RequestID:     requestID,
			OrgID:         orgID,
			ApproverID:    approverID,
			Decision:      model.DecisionApproved,
			DecisionNote:  nullString(note),
			PaymentStatus: nullString(model.PaymentPending),
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := s.approvals.Create(ctx, tx, approval); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				return ErrNotPending
			}
			return err
		}
		if err := s.requests.UpdateStatus(ctx, tx, orgID, requestID, model.StatusPending, model.StatusApproved); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				return ErrNotPending
			}
			return err
		}
		if err := s.writeAudit(ctx, tx, orgID, approverID, requestID, model.ActionApproved, model.StatusPending, model.StatusApproved, note); err != nil {
			return err
		}
		response = toResponse(approval, model.StatusApproved)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (s *approvalService) Reject(ctx context.Context, orgID, requestID, approverID, reason string) (*dto.ApprovalResponse, error) {
	if err := validator.ValidateReason(reason); err != nil {
		return nil, err
	}
	if err := s.authorize(ctx, orgID, approverID); err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)

	var response *dto.ApprovalResponse
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		req, err := s.requests.GetByID(ctx, tx, orgID, requestID, true)
		if err != nil {
			return mapRequestErr(err)
		}
		if req.Status != model.StatusPending {
			return ErrNotPending
		}

		now := time.Now().UTC()
		approval := &model.Approval{
			ID:           uuid.New().String(),
			RequestID:    requestID,
			OrgID:        orgID,
			ApproverID:   approverID,
			Decision:     model.DecisionRejected,
			DecisionNote: nullString(reason),
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := s.approvals.Create(ctx, tx, approval); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				return ErrNotPending
			}
			return err
		}
		if err := s.requests.UpdateStatus(ctx, tx, orgID, requestID, model.StatusPending, model.StatusRejected); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				return ErrNotPending
			}
			return err
		}
		if err := s.writeAudit(ctx, tx, orgID, approverID, requestID, model.ActionRejected, model.StatusPending, model.StatusRejected, reason); err != nil {
			return err
		}
		response = toResponse(approval, model.StatusRejected)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (s *approvalService) MarkPaid(ctx context.Context, orgID, requestID, approverID, method string) (*dto.ApprovalResponse, error) {
	normalized, err := validator.NormalizePaymentMethod(method)
	if err != nil {
		return nil, err
	}
	if err := s.authorize(ctx, orgID, approverID); err != nil {
		return nil, err
	}

	var response *dto.ApprovalResponse
	err = s.withTx(ctx, func(tx *sql.Tx) error {
		req, err := s.requests.GetByID(ctx, tx, orgID, requestID, true)
		if err != nil {
			return mapRequestErr(err)
		}
		if req.Status != model.StatusApproved {
			return ErrNotApproved
		}
		approval, err := s.approvals.GetByRequestID(ctx, tx, orgID, requestID, true)
		if err != nil {
			return mapApprovalErr(err)
		}
		if err := ensurePaymentOpen(approval); err != nil {
			return err
		}

		now := time.Now().UTC()
		approval.PaymentStatus = nullString(model.PaymentPaid)
		approval.PaymentMethod = nullString(normalized)
		approval.PaymentAt = sql.NullTime{Time: now, Valid: true}
		approval.FailureReason = sql.NullString{}
		approval.FailureAt = sql.NullTime{}
		approval.UpdatedAt = now

		if err := s.approvals.UpdatePayment(ctx, tx, approval); err != nil {
			return mapApprovalErr(err)
		}
		if err := s.requests.UpdateStatus(ctx, tx, orgID, requestID, model.StatusApproved, model.StatusPaid); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				return ErrNotApproved
			}
			return err
		}
		if err := s.writeAudit(ctx, tx, orgID, approverID, requestID, model.ActionPaid, model.StatusApproved, model.StatusPaid, normalized); err != nil {
			return err
		}
		response = toResponse(approval, model.StatusPaid)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (s *approvalService) MarkFailed(ctx context.Context, orgID, requestID, approverID, reason string) (*dto.ApprovalResponse, error) {
	if err := validator.ValidateReason(reason); err != nil {
		return nil, err
	}
	if err := s.authorize(ctx, orgID, approverID); err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)

	var response *dto.ApprovalResponse
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		req, err := s.requests.GetByID(ctx, tx, orgID, requestID, true)
		if err != nil {
			return mapRequestErr(err)
		}
		if req.Status != model.StatusApproved {
			return ErrNotApproved
		}
		approval, err := s.approvals.GetByRequestID(ctx, tx, orgID, requestID, true)
		if err != nil {
			return mapApprovalErr(err)
		}
		if err := ensurePaymentOpen(approval); err != nil {
			return err
		}

		now := time.Now().UTC()
		approval.PaymentStatus = nullString(model.PaymentFailed)
		approval.FailureReason = nullString(reason)
		approval.FailureAt = sql.NullTime{Time: now, Valid: true}
		approval.UpdatedAt = now

		if err := s.approvals.UpdatePayment(ctx, tx, approval); err != nil {
			return mapApprovalErr(err)
		}
		if err := s.requests.UpdateStatus(ctx, tx, orgID, requestID, model.StatusApproved, model.StatusFailed); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				return ErrNotApproved
			}
			return err
		}
		if err := s.writeAudit(ctx, tx, orgID, approverID, requestID, model.ActionFailed, model.StatusApproved, model.StatusFailed, reason); err != nil {
			return err
		}
		response = toResponse(approval, model.StatusFailed)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (s *approvalService) authorize(ctx context.Context, orgID, userID string) error {
	role, err := s.members.ActiveRole(ctx, s.readDB, orgID, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrForbidden
	}
	if err != nil {
		return err
	}
	if !strings.EqualFold(role, model.RoleFinance) && !strings.EqualFold(role, model.RoleOrgAdmin) {
		return ErrForbidden
	}
	return nil
}

func (s *approvalService) withTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	if s.tx == nil {
		return errors.New("transaction runner is not configured")
	}
	return s.tx.WithTx(ctx, fn)
}

func (s *approvalService) writeAudit(ctx context.Context, tx *sql.Tx, orgID, actorID, requestID, action, oldStatus, newStatus, note string) error {
	payload := map[string]string{
		"request_id": requestID,
		"old_status": oldStatus,
		"new_status": newStatus,
	}
	if note != "" {
		payload["note"] = note
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.audit.Write(ctx, tx, model.AuditEntry{
		ID:        uuid.New().String(),
		OrgID:     orgID,
		ActorID:   actorID,
		Action:    action,
		TargetID:  requestID,
		Metadata:  string(raw),
		CreatedAt: time.Now().UTC(),
	})
}

func ensurePaymentOpen(approval *model.Approval) error {
	if !approval.PaymentStatus.Valid {
		return ErrNotApproved
	}
	switch approval.PaymentStatus.String {
	case model.PaymentPending:
		return nil
	case model.PaymentPaid, model.PaymentFailed:
		return ErrPaymentAlreadySettled
	default:
		return ErrNotApproved
	}
}

func mapRequestErr(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return ErrRequestNotFound
	}
	return err
}

func mapApprovalErr(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return ErrApprovalNotFound
	}
	return err
}

func nullString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func toResponse(approval *model.Approval, requestStatus string) *dto.ApprovalResponse {
	resp := &dto.ApprovalResponse{
		ID:            approval.ID,
		RequestID:     approval.RequestID,
		OrgID:         approval.OrgID,
		ApproverID:    approval.ApproverID,
		Decision:      approval.Decision,
		DecisionNote:  approval.DecisionNote.String,
		PaymentStatus: approval.PaymentStatus.String,
		PaymentMethod: approval.PaymentMethod.String,
		FailureReason: approval.FailureReason.String,
		RequestStatus: requestStatus,
		CreatedAt:     approval.CreatedAt,
		UpdatedAt:     approval.UpdatedAt,
	}
	if approval.PaymentAt.Valid {
		paidAt := approval.PaymentAt.Time
		resp.PaymentAt = &paidAt
	}
	if approval.FailureAt.Valid {
		failedAt := approval.FailureAt.Time
		resp.FailureAt = &failedAt
	}
	return resp
}
