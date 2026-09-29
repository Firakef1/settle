package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Firakef1/settle/backend/internal/approvals/model"
	"github.com/Firakef1/settle/backend/internal/approvals/repository"
	"github.com/Firakef1/settle/backend/internal/approvals/validator"
)

type memTx struct{}

func (memTx) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	return fn(nil)
}

type fakeMembers struct {
	role string
}

func (f fakeMembers) ActiveRole(ctx context.Context, db repository.DBTX, orgID, userID string) (string, error) {
	if f.role == "" {
		return "", repository.ErrNotFound
	}
	return f.role, nil
}

type fakeRequests struct {
	status  string
	missing bool
}

func (f *fakeRequests) GetByID(ctx context.Context, db repository.DBTX, orgID, requestID string, forUpdate bool) (*model.RequestSnapshot, error) {
	if f.missing {
		return nil, repository.ErrNotFound
	}
	return &model.RequestSnapshot{ID: requestID, OrgID: orgID, Status: f.status}, nil
}

func (f *fakeRequests) UpdateStatus(ctx context.Context, db repository.DBTX, orgID, requestID, fromStatus, toStatus string) error {
	if f.status != fromStatus {
		return repository.ErrConflict
	}
	f.status = toStatus
	return nil
}

type fakeApprovals struct {
	row       *model.Approval
	createErr error
}

func (f *fakeApprovals) Create(ctx context.Context, db repository.DBTX, approval *model.Approval) error {
	if f.createErr != nil {
		return f.createErr
	}
	copied := *approval
	f.row = &copied
	return nil
}

func (f *fakeApprovals) GetByRequestID(ctx context.Context, db repository.DBTX, orgID, requestID string, forUpdate bool) (*model.Approval, error) {
	if f.row == nil {
		return nil, repository.ErrNotFound
	}
	copied := *f.row
	return &copied, nil
}

func (f *fakeApprovals) UpdatePayment(ctx context.Context, db repository.DBTX, approval *model.Approval) error {
	copied := *approval
	f.row = &copied
	return nil
}

type fakeAudit struct {
	actions []string
}

func (f *fakeAudit) Write(ctx context.Context, db repository.DBTX, entry model.AuditEntry) error {
	f.actions = append(f.actions, entry.Action)
	return nil
}

type harness struct {
	svc       ApprovalService
	requests  *fakeRequests
	approvals *fakeApprovals
	audit     *fakeAudit
}

func newHarness(role, status string) harness {
	requests := &fakeRequests{status: status}
	approvals := &fakeApprovals{}
	audit := &fakeAudit{}
	svc := NewApprovalServiceWithRunner(memTx{}, nil, approvals, requests, fakeMembers{role: role}, audit)
	return harness{svc: svc, requests: requests, approvals: approvals, audit: audit}
}

func TestApprove_PendingRequest(t *testing.T) {
	h := newHarness(model.RoleFinance, model.StatusPending)

	resp, err := h.svc.Approve(context.Background(), "org-1", "REQ-000001", "user-1", " looks good ")
	require.NoError(t, err)
	require.Equal(t, model.StatusApproved, resp.RequestStatus)
	require.Equal(t, model.DecisionApproved, resp.Decision)
	require.Equal(t, model.PaymentPending, resp.PaymentStatus)
	require.Equal(t, "looks good", resp.DecisionNote)
	require.Equal(t, model.StatusApproved, h.requests.status)
	require.Equal(t, []string{model.ActionApproved}, h.audit.actions)
}

func TestApprove_NotPending(t *testing.T) {
	h := newHarness(model.RoleFinance, model.StatusApproved)

	_, err := h.svc.Approve(context.Background(), "org-1", "REQ-000001", "user-1", "")
	require.ErrorIs(t, err, ErrNotPending)
	require.Nil(t, h.approvals.row)
	require.Empty(t, h.audit.actions)
}

func TestApprove_RequestMissing(t *testing.T) {
	h := newHarness(model.RoleOrgAdmin, model.StatusPending)
	h.requests.missing = true

	_, err := h.svc.Approve(context.Background(), "org-1", "REQ-000001", "user-1", "")
	require.ErrorIs(t, err, ErrRequestNotFound)
}

func TestApprove_StaffForbidden(t *testing.T) {
	h := newHarness("staff", model.StatusPending)

	_, err := h.svc.Approve(context.Background(), "org-1", "REQ-000001", "user-1", "")
	require.ErrorIs(t, err, ErrForbidden)
	require.Equal(t, model.StatusPending, h.requests.status)
}

func TestReject_RequiresReason(t *testing.T) {
	h := newHarness(model.RoleFinance, model.StatusPending)

	_, err := h.svc.Reject(context.Background(), "org-1", "REQ-000001", "user-1", "  ")
	require.ErrorIs(t, err, validator.ErrReasonRequired)
	require.Equal(t, model.StatusPending, h.requests.status)
}

func TestReject_PendingRequest(t *testing.T) {
	h := newHarness(model.RoleFinance, model.StatusPending)

	resp, err := h.svc.Reject(context.Background(), "org-1", "REQ-000001", "user-1", "missing receipt")
	require.NoError(t, err)
	require.Equal(t, model.StatusRejected, resp.RequestStatus)
	require.Equal(t, model.DecisionRejected, resp.Decision)
	require.Empty(t, resp.PaymentStatus)
	require.Equal(t, model.StatusRejected, h.requests.status)
	require.Equal(t, []string{model.ActionRejected}, h.audit.actions)
}

func TestMarkPaid_InvalidMethod(t *testing.T) {
	h := newHarness(model.RoleFinance, model.StatusApproved)

	_, err := h.svc.MarkPaid(context.Background(), "org-1", "REQ-000001", "user-1", "wire")
	require.ErrorIs(t, err, validator.ErrInvalidPaymentMethod)
}

func TestMarkPaid_RequiresApprovedRequest(t *testing.T) {
	h := newHarness(model.RoleFinance, model.StatusPending)

	_, err := h.svc.MarkPaid(context.Background(), "org-1", "REQ-000001", "user-1", "Cash")
	require.ErrorIs(t, err, ErrNotApproved)
}

func TestMarkPaid_Success(t *testing.T) {
	h := newHarness(model.RoleFinance, model.StatusApproved)
	h.approvals.row = &model.Approval{
		ID:            "appr-1",
		RequestID:     "REQ-000001",
		OrgID:         "org-1",
		Decision:      model.DecisionApproved,
		PaymentStatus: nullString(model.PaymentPending),
	}

	resp, err := h.svc.MarkPaid(context.Background(), "org-1", "REQ-000001", "user-1", "Cash")
	require.NoError(t, err)
	require.Equal(t, model.StatusPaid, resp.RequestStatus)
	require.Equal(t, model.PaymentPaid, resp.PaymentStatus)
	require.Equal(t, "cash", resp.PaymentMethod)
	require.NotNil(t, resp.PaymentAt)
	require.Equal(t, model.StatusPaid, h.requests.status)
	require.Equal(t, []string{model.ActionPaid}, h.audit.actions)
}

func TestMarkPaid_AlreadySettled(t *testing.T) {
	h := newHarness(model.RoleFinance, model.StatusApproved)
	h.approvals.row = &model.Approval{
		ID:            "appr-1",
		PaymentStatus: nullString(model.PaymentPaid),
	}

	_, err := h.svc.MarkPaid(context.Background(), "org-1", "REQ-000001", "user-1", "cash")
	require.ErrorIs(t, err, ErrPaymentAlreadySettled)
	require.Equal(t, model.StatusApproved, h.requests.status)
}

func TestMarkFailed_RequiresReason(t *testing.T) {
	h := newHarness(model.RoleFinance, model.StatusApproved)

	_, err := h.svc.MarkFailed(context.Background(), "org-1", "REQ-000001", "user-1", "")
	require.ErrorIs(t, err, validator.ErrReasonRequired)
}

func TestMarkFailed_Success(t *testing.T) {
	h := newHarness(model.RoleOrgAdmin, model.StatusApproved)
	h.approvals.row = &model.Approval{
		ID:            "appr-1",
		RequestID:     "REQ-000001",
		OrgID:         "org-1",
		Decision:      model.DecisionApproved,
		PaymentStatus: nullString(model.PaymentPending),
	}

	resp, err := h.svc.MarkFailed(context.Background(), "org-1", "REQ-000001", "user-1", "bank declined")
	require.NoError(t, err)
	require.Equal(t, model.StatusFailed, resp.RequestStatus)
	require.Equal(t, model.PaymentFailed, resp.PaymentStatus)
	require.Equal(t, "bank declined", resp.FailureReason)
	require.NotNil(t, resp.FailureAt)
	require.Equal(t, []string{model.ActionFailed}, h.audit.actions)
}

func TestMarkFailed_MissingApproval(t *testing.T) {
	h := newHarness(model.RoleFinance, model.StatusApproved)

	_, err := h.svc.MarkFailed(context.Background(), "org-1", "REQ-000001", "user-1", "bank declined")
	require.ErrorIs(t, err, ErrApprovalNotFound)
}
