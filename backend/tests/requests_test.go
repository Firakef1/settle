package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	approvalDto "github.com/Firakef1/settle/backend/internal/approvals/dto"
	approvalHandler "github.com/Firakef1/settle/backend/internal/approvals/handler"
	authDto "github.com/Firakef1/settle/backend/internal/auth/dto"
	authHandler "github.com/Firakef1/settle/backend/internal/auth/handler"
	authModel "github.com/Firakef1/settle/backend/internal/auth/model"
	authRepo "github.com/Firakef1/settle/backend/internal/auth/repository"
	authService "github.com/Firakef1/settle/backend/internal/auth/service"
	orgDto "github.com/Firakef1/settle/backend/internal/organaization/dto"
	orghandler "github.com/Firakef1/settle/backend/internal/organaization/handler"
	orgService "github.com/Firakef1/settle/backend/internal/organaization/service"
	reqDtoPkg "github.com/Firakef1/settle/backend/internal/requests/dto"
	reqHandler "github.com/Firakef1/settle/backend/internal/requests/handler"
	reqRepoPkg "github.com/Firakef1/settle/backend/internal/requests/repository"
	reqServicePkg "github.com/Firakef1/settle/backend/internal/requests/service"
	"github.com/Firakef1/settle/backend/internal/router"
	"github.com/Firakef1/settle/backend/internal/shared/config"
	sharedService "github.com/Firakef1/settle/backend/internal/shared/service"
)

type inMemoryApprovalService struct {
	reqRepo reqRepoPkg.RequestRepository
}

func (m *inMemoryApprovalService) Approve(ctx context.Context, orgID, requestID, approverID, note string) (*approvalDto.ApprovalResponse, error) {
	req, err := m.reqRepo.GetByID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if req.OrgID != orgID {
		return nil, reqRepoPkg.ErrRequestNotFound
	}
	if err := m.reqRepo.UpdateStatus(ctx, requestID, "approved"); err != nil {
		return nil, err
	}
	now := time.Now()
	return &approvalDto.ApprovalResponse{
		ID:            "app-" + requestID,
		RequestID:     requestID,
		OrgID:         orgID,
		ApproverID:    approverID,
		Decision:      "approved",
		DecisionNote:  note,
		PaymentStatus: "pending_payment",
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func (m *inMemoryApprovalService) Reject(ctx context.Context, orgID, requestID, approverID, reason string) (*approvalDto.ApprovalResponse, error) {
	req, err := m.reqRepo.GetByID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if req.OrgID != orgID {
		return nil, reqRepoPkg.ErrRequestNotFound
	}
	if err := m.reqRepo.UpdateStatus(ctx, requestID, "rejected"); err != nil {
		return nil, err
	}
	now := time.Now()
	return &approvalDto.ApprovalResponse{
		ID:           "app-" + requestID,
		RequestID:    requestID,
		OrgID:        orgID,
		ApproverID:   approverID,
		Decision:     "rejected",
		DecisionNote: reason,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (m *inMemoryApprovalService) MarkPaid(ctx context.Context, orgID, requestID, approverID, method string) (*approvalDto.ApprovalResponse, error) {
	req, err := m.reqRepo.GetByID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if req.OrgID != orgID {
		return nil, reqRepoPkg.ErrRequestNotFound
	}
	if err := m.reqRepo.UpdateStatus(ctx, requestID, "paid"); err != nil {
		return nil, err
	}
	now := time.Now()
	return &approvalDto.ApprovalResponse{
		ID:            "app-" + requestID,
		RequestID:     requestID,
		OrgID:         orgID,
		ApproverID:    approverID,
		Decision:      "approved",
		PaymentStatus: "paid",
		PaymentMethod: method,
		PaymentAt:     &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func (m *inMemoryApprovalService) MarkFailed(ctx context.Context, orgID, requestID, approverID, reason string) (*approvalDto.ApprovalResponse, error) {
	req, err := m.reqRepo.GetByID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if req.OrgID != orgID {
		return nil, reqRepoPkg.ErrRequestNotFound
	}
	if err := m.reqRepo.UpdateStatus(ctx, requestID, "failed"); err != nil {
		return nil, err
	}
	now := time.Now()
	return &approvalDto.ApprovalResponse{
		ID:            "app-" + requestID,
		RequestID:     requestID,
		OrgID:         orgID,
		ApproverID:    approverID,
		Decision:      "approved",
		PaymentStatus: "failed",
		FailureReason: reason,
		FailureAt:     &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

type testUserProviderAdapter struct {
	repo authRepo.UserRepository
}

func (a *testUserProviderAdapter) GetUserBasicInfo(ctx context.Context, userID string) (reqDtoPkg.RequesterResponse, error) {
	u, err := a.repo.FindByID(ctx, userID)
	if err != nil {
		return reqDtoPkg.RequesterResponse{ID: userID}, err
	}
	return reqDtoPkg.RequesterResponse{
		ID:    u.ID,
		Name:  u.Name,
		Email: u.Email,
	}, nil
}

func setupRequestTestApp(t *testing.T) (*gin.Engine, *authRepo.UserRepo, *reqRepoPkg.RequestRepo) {
	gin.SetMode(gin.TestMode)

	config.AppConfig.SecretKey = "test_integration_secret_key_1234567890"
	config.AppConfig.AccessTokenTTL = 24 * time.Hour
	config.AppConfig.RefreshTokenTTL = 7 * 24 * time.Hour
	config.AppConfig.UploadDir = t.TempDir()
	config.AppConfig.Email = config.EmailConfig{
		LogOnly:     true,
		CodeSecret:  "12345678901234567890123456789012",
		CodeTTL:     15 * time.Minute,
		SendTimeout: 10 * time.Second,
	}

	// 1. Auth handlers (in-memory)
	userRepo := authRepo.NewUserRepo(nil)
	refreshTokenRepo := authRepo.NewRefreshTokenRepo(nil)
	pwdResetRepo := authRepo.NewPasswordResetOTPRepo(nil)
	verCodeRepo := authRepo.NewVerificationCodeRepo(nil)
	hashSvc := sharedService.NewHashService()
	jwtSvc := sharedService.NewJWTServiceWithSecret(config.AppConfig.SecretKey)
	emailSvc := sharedService.NewEmailService()
	authSvc := authService.NewAuthService(userRepo, refreshTokenRepo, pwdResetRepo, emailSvc, hashSvc, jwtSvc)
	verSvc := authService.NewVerificationService(verCodeRepo, userRepo, emailSvc, authSvc, 15*time.Minute)
	authH := authHandler.NewAuthHandler(authSvc, verSvc)
	verH := authHandler.NewVerificationHandler(verSvc, authSvc)

	// 2. Organization handlers (using mock repos from package tests)
	txRunner := &MockTxRunner{}
	orgRepo := NewMockOrgRepo()
	memberRepo := NewMockMemberRepo()
	auditWriter := NewMockAuditWriter()
	orgSvc := orgService.NewOrgServiceWithTx(txRunner, orgRepo, memberRepo, auditWriter)
	orgH := orghandler.NewOrgHandler(orgSvc)

	// 3. Request handlers (in-memory)
	reqRepo := reqRepoPkg.NewRequestRepo(nil)
	recRepo := reqRepoPkg.NewReceiptRepo(nil)
	comRepo := reqRepoPkg.NewCommentRepo(nil)
	userAdapter := &testUserProviderAdapter{repo: userRepo}
	reqSvc := reqServicePkg.NewRequestService(reqRepo, recRepo, comRepo, userAdapter)
	requestH := reqHandler.NewRequestHandler(reqSvc)

	// 4. Approval handlers (in-memory)
	apprSvc := &inMemoryApprovalService{reqRepo: reqRepo}
	approvalH := approvalHandler.NewApprovalHandler(apprSvc)

	allHandlers := &router.Handlers{
		Auth:         authH,
		Verification: verH,
		Org:          orgH,
		Request:      requestH,
		Approval:     approvalH,
	}

	engine := router.SetupRouter(allHandlers)
	return engine, userRepo, reqRepo
}

func TestIntegration_PayoutRequestFlow(t *testing.T) {
	engine, userRepo, _ := setupRequestTestApp(t)

	uniqueSuffix := fmt.Sprintf("%d", time.Now().UnixNano())
	email := fmt.Sprintf("req-user-%s@example.com", uniqueSuffix)
	password := "SecurePassword123!"
	name := "Integration Tester"

	// ---------------------------------------------------------
	// 1. Signup
	// ---------------------------------------------------------
	signupPayload, _ := json.Marshal(authDto.SignupRequest{
		Email:    email,
		Password: password,
		Name:     name,
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewBuffer(signupPayload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "signup must succeed")

	var signupResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &signupResp)
	require.NoError(t, err)
	userData := signupResp["user"].(map[string]interface{})
	userID := userData["id"].(string)
	require.NotEmpty(t, userID)

	// Mark user verified in memory repository
	userRepo.SetEmailVerifiedForTest(email, true)

	// ---------------------------------------------------------
	// 2. Login
	// ---------------------------------------------------------
	loginPayload, _ := json.Marshal(authDto.LoginRequest{
		Email:    email,
		Password: password,
	})
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginPayload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "login must succeed")

	var loginResp authDto.LoginResponse
	err = json.Unmarshal(w.Body.Bytes(), &loginResp)
	require.NoError(t, err)
	token := loginResp.Token
	require.NotEmpty(t, token)

	// ---------------------------------------------------------
	// 3. Create Organization
	// ---------------------------------------------------------
	createOrgPayload, _ := json.Marshal(orgDto.CreateOrgRequest{
		Name:     "Test Org " + uniqueSuffix,
		Currency: "USD",
	})
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewBuffer(createOrgPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "create org must succeed")

	var orgResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &orgResp)
	require.NoError(t, err)
	orgID := orgResp.Data.ID
	require.NotEmpty(t, orgID)

	// Register organization membership for user and issue token with org context
	userRepo.AddMemoryOrgMembership(authModel.OrgMembership{
		ID:         "mem-" + uniqueSuffix,
		OrgID:      orgID,
		OrgName:    "Test Org " + uniqueSuffix,
		OrgSlug:    "test-org-" + uniqueSuffix,
		UserID:     userID,
		Role:       "org_admin",
		Department: "Finance",
		Status:     "active",
	})

	// Generate JWT with org context
	orgToken := generateTestJWT(userID, orgID, "org_admin")
	require.NotEmpty(t, orgToken)

	// ---------------------------------------------------------
	// 4. Create Payout Request (reimbursement -> starts as draft)
	// ---------------------------------------------------------
	createReqPayload, _ := json.Marshal(reqDtoPkg.CreateRequestDTO{
		Type:    "reimbursement",
		Amount:  175.50,
		Purpose: "Business travel expenses for team offsite",
		Urgency: "routine",
	})
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/requests", bytes.NewBuffer(createReqPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+orgToken)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "create request must succeed: %s", w.Body.String())

	var reqCreateResp struct {
		Data struct {
			ID     string  `json:"id"`
			Status string  `json:"status"`
			Amount float64 `json:"amount"`
		} `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &reqCreateResp)
	require.NoError(t, err)
	requestID := reqCreateResp.Data.ID
	require.NotEmpty(t, requestID)
	assert.Equal(t, "draft", reqCreateResp.Data.Status)
	assert.Equal(t, 175.50, reqCreateResp.Data.Amount)

	// ---------------------------------------------------------
	// 5. Upload Receipt (multipart/form-data)
	// ---------------------------------------------------------
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="receipt"; filename="offsite_hotel.png"`)
	h.Set("Content-Type", "image/png")
	part, err := writer.CreatePart(h)
	require.NoError(t, err)
	_, err = part.Write([]byte("fake png image data for receipt"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/requests/%s/receipts", requestID), body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+orgToken)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "upload receipt must succeed: %s", w.Body.String())

	var receiptResp struct {
		Data struct {
			ID        string `json:"id"`
			FilePath  string `json:"file_path"`
			OCRStatus string `json:"ocr_status"`
		} `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &receiptResp)
	require.NoError(t, err)
	assert.NotEmpty(t, receiptResp.Data.ID)
	assert.NotEmpty(t, receiptResp.Data.FilePath)
	assert.Equal(t, "pending", receiptResp.Data.OCRStatus)

	// ---------------------------------------------------------
	// 6. Add Comment and Get Comments
	// ---------------------------------------------------------
	commentPayload, _ := json.Marshal(reqDtoPkg.AddCommentDTO{
		Content: "Attaching the itemized hotel bill for reimbursement.",
	})
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/requests/%s/comments", requestID), bytes.NewBuffer(commentPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+orgToken)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "add comment must succeed: %s", w.Body.String())

	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/requests/%s/comments", requestID), nil)
	req.Header.Set("Authorization", "Bearer "+orgToken)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "get comments must succeed: %s", w.Body.String())

	var commentsResp struct {
		Data []struct {
			ID      string `json:"id"`
			Content string `json:"content"`
		} `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &commentsResp)
	require.NoError(t, err)
	require.Len(t, commentsResp.Data, 1)
	assert.Equal(t, "Attaching the itemized hotel bill for reimbursement.", commentsResp.Data[0].Content)

	// ---------------------------------------------------------
	// 7. Submit Request (draft -> pending)
	// ---------------------------------------------------------
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/requests/%s/submit", requestID), nil)
	req.Header.Set("Authorization", "Bearer "+orgToken)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "submit request must succeed: %s", w.Body.String())

	var submitResp struct {
		Data struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &submitResp)
	require.NoError(t, err)
	assert.Equal(t, "pending", submitResp.Data.Status)

	// ---------------------------------------------------------
	// 8. Approve Request (pending -> approved)
	// ---------------------------------------------------------
	approvePayload, _ := json.Marshal(map[string]string{
		"note": "Approved by finance admin",
	})
	req, _ = http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/requests/%s/approve", requestID), bytes.NewBuffer(approvePayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+orgToken)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "approve request must succeed: %s", w.Body.String())

	// Verify request status is approved via detail endpoint
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/requests/%s", requestID), nil)
	req.Header.Set("Authorization", "Bearer "+orgToken)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var detailResp struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &detailResp)
	require.NoError(t, err)
	assert.Equal(t, "approved", detailResp.Data.Status)

	// ---------------------------------------------------------
	// 9. Mark Paid (approved -> paid)
	// ---------------------------------------------------------
	markPaidPayload, _ := json.Marshal(map[string]string{
		"payment_method": "bank_transfer",
	})
	req, _ = http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/requests/%s/mark-paid", requestID), bytes.NewBuffer(markPaidPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+orgToken)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "mark paid must succeed: %s", w.Body.String())

	// Verify final request status is paid
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/requests/%s", requestID), nil)
	req.Header.Set("Authorization", "Bearer "+orgToken)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &detailResp)
	require.NoError(t, err)
	assert.Equal(t, "paid", detailResp.Data.Status)

	// ---------------------------------------------------------
	// 10. List Requests
	// ---------------------------------------------------------
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/requests", nil)
	req.Header.Set("Authorization", "Bearer "+orgToken)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var listResp struct {
		Data []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &listResp)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, listResp.Meta.Total, 1)
	found := false
	for _, item := range listResp.Data {
		if item.ID == requestID && item.Status == "paid" {
			found = true
			break
		}
	}
	assert.True(t, found, "paid request should be in list")
}
