package handler

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Firakef1/settle/backend/internal/approvals/dto"
	"github.com/Firakef1/settle/backend/internal/approvals/service"
	"github.com/Firakef1/settle/backend/internal/approvals/validator"
	"github.com/Firakef1/settle/backend/internal/shared/middleware"
)

// ApprovalHandler handles finance decisions on payout requests.
type ApprovalHandler struct {
	approvalService service.ApprovalService
}

// NewApprovalHandler injects the approval service.
func NewApprovalHandler(approvalService service.ApprovalService) *ApprovalHandler {
	return &ApprovalHandler{approvalService: approvalService}
}

// Approve godoc
// @Summary      Approve a pending request
// @Description  Finance or org admin approves a pending payout request. Creates an approval with payment_status pending_payment, sets the request status to approved, and writes an audit row in one transaction.
// @Tags         approvals
// @Accept       json
// @Produce      json
// @Param        id path string true "Request ID"
// @Param        org_id query string false "Organization ID when the JWT has no org claim"
// @Param        body body dto.ApproveRequest false "Optional decision note"
// @Success      200 {object} map[string]dto.ApprovalResponse
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /v1/requests/{id}/approve [put]
func (h *ApprovalHandler) Approve(c *gin.Context) {
	orgID, requestID, approverID, ok := approvalContext(c)
	if !ok {
		return
	}

	var req dto.ApproveRequest
	if err := bindJSON(c, &req, false); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	resp, err := h.approvalService.Approve(c.Request.Context(), orgID, requestID, approverID, req.Note)
	if err != nil {
		writeApprovalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// Reject godoc
// @Summary      Reject a pending request
// @Description  Finance or org admin rejects a pending payout request. A reason is required. Sets the request status to rejected and writes an audit row in one transaction.
// @Tags         approvals
// @Accept       json
// @Produce      json
// @Param        id path string true "Request ID"
// @Param        org_id query string false "Organization ID when the JWT has no org claim"
// @Param        body body dto.RejectRequest true "Rejection reason"
// @Success      200 {object} map[string]dto.ApprovalResponse
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /v1/requests/{id}/reject [put]
func (h *ApprovalHandler) Reject(c *gin.Context) {
	orgID, requestID, approverID, ok := approvalContext(c)
	if !ok {
		return
	}

	var req dto.RejectRequest
	if err := bindJSON(c, &req, true); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	resp, err := h.approvalService.Reject(c.Request.Context(), orgID, requestID, approverID, req.Reason)
	if err != nil {
		writeApprovalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// MarkPaid godoc
// @Summary      Mark an approved request as paid
// @Description  Finance or org admin records a successful payment. Allowed only when the request is approved and payment is still pending. payment_method must be bank_transfer, check, cash, or other.
// @Tags         approvals
// @Accept       json
// @Produce      json
// @Param        id path string true "Request ID"
// @Param        org_id query string false "Organization ID when the JWT has no org claim"
// @Param        body body dto.MarkPaidRequest true "Payment method"
// @Success      200 {object} map[string]dto.ApprovalResponse
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /v1/requests/{id}/mark-paid [put]
func (h *ApprovalHandler) MarkPaid(c *gin.Context) {
	orgID, requestID, approverID, ok := approvalContext(c)
	if !ok {
		return
	}

	var req dto.MarkPaidRequest
	if err := bindJSON(c, &req, true); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	resp, err := h.approvalService.MarkPaid(c.Request.Context(), orgID, requestID, approverID, req.PaymentMethod)
	if err != nil {
		writeApprovalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// MarkFailed godoc
// @Summary      Record a failed payment
// @Description  Finance or org admin records a payment failure on an approved request. failure_reason is required. Allowed only while payment is still pending.
// @Tags         approvals
// @Accept       json
// @Produce      json
// @Param        id path string true "Request ID"
// @Param        org_id query string false "Organization ID when the JWT has no org claim"
// @Param        body body dto.PaymentFailedRequest true "Failure reason"
// @Success      200 {object} map[string]dto.ApprovalResponse
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /v1/requests/{id}/payment-failed [put]
func (h *ApprovalHandler) MarkFailed(c *gin.Context) {
	orgID, requestID, approverID, ok := approvalContext(c)
	if !ok {
		return
	}

	var req dto.PaymentFailedRequest
	if err := bindJSON(c, &req, true); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	resp, err := h.approvalService.MarkFailed(c.Request.Context(), orgID, requestID, approverID, req.FailureReason)
	if err != nil {
		writeApprovalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func approvalContext(c *gin.Context) (orgID, requestID, approverID string, ok bool) {
	approverID = middleware.GetUserID(c)
	if approverID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return "", "", "", false
	}

	requestID = strings.TrimSpace(c.Param("id"))
	if requestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request id is required"})
		return "", "", "", false
	}

	orgID = middleware.GetOrgID(c)
	if orgID == "" {
		orgID = strings.TrimSpace(c.Query("org_id"))
	}
	if orgID == "" {
		orgID = strings.TrimSpace(c.GetHeader("X-Organization-Id"))
	}
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "organization context is required (in token or org_id parameter)"})
		return "", "", "", false
	}
	return orgID, requestID, approverID, true
}

func bindJSON(c *gin.Context, dest any, required bool) error {
	if !required && (c.Request.Body == nil || c.Request.ContentLength == 0) {
		return nil
	}
	err := c.ShouldBindJSON(dest)
	if !required && errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

func writeApprovalError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, validator.ErrReasonRequired), errors.Is(err, validator.ErrInvalidPaymentMethod):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrRequestNotFound), errors.Is(err, service.ErrApprovalNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrNotPending), errors.Is(err, service.ErrNotApproved), errors.Is(err, service.ErrPaymentAlreadySettled):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process approval"})
	}
}
