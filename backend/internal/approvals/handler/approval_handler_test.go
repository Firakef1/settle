package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/Firakef1/settle/backend/internal/approvals/dto"
	"github.com/Firakef1/settle/backend/internal/approvals/service"
	"github.com/Firakef1/settle/backend/internal/shared/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type stubApprovalService struct {
	err  error
	resp *dto.ApprovalResponse
}

func (s stubApprovalService) Approve(ctx context.Context, orgID, requestID, approverID, note string) (*dto.ApprovalResponse, error) {
	return s.resp, s.err
}

func (s stubApprovalService) Reject(ctx context.Context, orgID, requestID, approverID, reason string) (*dto.ApprovalResponse, error) {
	return s.resp, s.err
}

func (s stubApprovalService) MarkPaid(ctx context.Context, orgID, requestID, approverID, method string) (*dto.ApprovalResponse, error) {
	return s.resp, s.err
}

func (s stubApprovalService) MarkFailed(ctx context.Context, orgID, requestID, approverID, reason string) (*dto.ApprovalResponse, error) {
	return s.resp, s.err
}

func perform(h *ApprovalHandler, method, path string, body string, userID, orgID string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r := gin.New()
	r.PUT("/requests/:id/approve", func(c *gin.Context) {
		if userID != "" {
			c.Set(middleware.ContextUserIDKey, userID)
		}
		if orgID != "" {
			c.Set(middleware.ContextOrgIDKey, orgID)
		}
		h.Approve(c)
	})
	r.PUT("/requests/:id/reject", func(c *gin.Context) {
		if userID != "" {
			c.Set(middleware.ContextUserIDKey, userID)
		}
		if orgID != "" {
			c.Set(middleware.ContextOrgIDKey, orgID)
		}
		h.Reject(c)
	})
	r.PUT("/requests/:id/mark-paid", func(c *gin.Context) {
		if userID != "" {
			c.Set(middleware.ContextUserIDKey, userID)
		}
		if orgID != "" {
			c.Set(middleware.ContextOrgIDKey, orgID)
		}
		h.MarkPaid(c)
	})

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestApprove_Unauthorized(t *testing.T) {
	h := NewApprovalHandler(stubApprovalService{})
	w := perform(h, http.MethodPut, "/requests/REQ-1/approve", `{}`, "", "org-1")
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestApprove_MissingOrg(t *testing.T) {
	h := NewApprovalHandler(stubApprovalService{})
	w := perform(h, http.MethodPut, "/requests/REQ-1/approve", `{}`, "user-1", "")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReject_InvalidJSON(t *testing.T) {
	h := NewApprovalHandler(stubApprovalService{})
	w := perform(h, http.MethodPut, "/requests/REQ-1/reject", `{`, "user-1", "org-1")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReject_MapsConflict(t *testing.T) {
	h := NewApprovalHandler(stubApprovalService{err: service.ErrNotPending})
	w := perform(h, http.MethodPut, "/requests/REQ-1/reject", `{"reason":"no receipt"}`, "user-1", "org-1")
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestMarkPaid_SuccessEnvelope(t *testing.T) {
	h := NewApprovalHandler(stubApprovalService{
		resp: &dto.ApprovalResponse{
			ID:            "appr-1",
			RequestID:     "REQ-1",
			RequestStatus: "paid",
			PaymentMethod: "cash",
		},
	})
	w := perform(h, http.MethodPut, "/requests/REQ-1/mark-paid", `{"payment_method":"cash"}`, "user-1", "org-1")
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]dto.ApprovalResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "paid", body["data"].RequestStatus)
	require.Equal(t, "cash", body["data"].PaymentMethod)
}
