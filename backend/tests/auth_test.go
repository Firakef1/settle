package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Firakef1/settle/backend/internal/auth/dto"
	authHandler "github.com/Firakef1/settle/backend/internal/auth/handler"
	"github.com/Firakef1/settle/backend/internal/auth/model"
	"github.com/Firakef1/settle/backend/internal/auth/repository"
	authService "github.com/Firakef1/settle/backend/internal/auth/service"
	"github.com/Firakef1/settle/backend/internal/router"
	sharedService "github.com/Firakef1/settle/backend/internal/shared/service"
)

func setupTestApp() (*authService.AuthService, *repository.UserRepo, *repository.RefreshTokenRepo, *sharedService.JWTService, http.Handler) {
	repo := repository.NewUserRepo(nil)
	refreshTokenRepo := repository.NewRefreshTokenRepo(nil)
	hashSvc := sharedService.NewHashService()
	jwtSvc := sharedService.NewJWTServiceWithSecret("test_secret_key_1234567890")
	svc := authService.NewAuthService(repo, refreshTokenRepo, hashSvc, jwtSvc)
	handler := authHandler.NewAuthHandler(svc)
	allHandlers := &router.Handlers{Auth: handler}
	r := router.SetupRouter(allHandlers)
	return svc, repo, refreshTokenRepo, jwtSvc, r
}

func TestIntegration_SignupLoginRefreshLogout(t *testing.T) {
	_, repo, _, jwtSvc, app := setupTestApp()

	// Step 1: Signup
	signupPayload := dto.SignupRequest{
		Email:    "integration@example.com",
		Password: "password123",
		Name:     "Integration User",
	}
	signupBody, _ := json.Marshal(signupPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewBuffer(signupBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var signupResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &signupResp)
	require.NoError(t, err)
	assert.Equal(t, "user created successfully", signupResp["message"])

	userData := signupResp["user"].(map[string]interface{})
	userID := userData["id"].(string)
	assert.NotEmpty(t, userID)

	// Add mock org membership
	repo.AddMemoryOrgMembership(model.OrgMembership{
		ID:         "mem-1",
		OrgID:      "org-100",
		OrgName:    "Acme Corp",
		OrgSlug:    "acme",
		UserID:     userID,
		Role:       "org_admin",
		Department: "Finance",
		Status:     "active",
	})

	// Step 2: Login – should return access token AND refresh token
	loginBody, _ := json.Marshal(dto.LoginRequest{
		Email:    "integration@example.com",
		Password: "password123",
	})
	loginReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	wLogin := httptest.NewRecorder()
	app.ServeHTTP(wLogin, loginReq)

	assert.Equal(t, http.StatusOK, wLogin.Code)

	var loginResp dto.LoginResponse
	err = json.Unmarshal(wLogin.Body.Bytes(), &loginResp)
	require.NoError(t, err)

	assert.NotEmpty(t, loginResp.Token, "access token should be present")
	assert.NotEmpty(t, loginResp.RefreshToken, "refresh token should be present")
	assert.Equal(t, "integration@example.com", loginResp.User.Email)
	assert.Len(t, loginResp.Orgs, 1)
	assert.Equal(t, "org-100", loginResp.Orgs[0].OrgID)
	assert.Equal(t, "Acme Corp", loginResp.Orgs[0].OrgName)

	// Verify access token
	claims, err := jwtSvc.Verify(loginResp.Token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.NotEmpty(t, claims.ID, "JWT should have a JTI")

	// Step 3: Refresh token
	refreshBody, _ := json.Marshal(dto.RefreshRequest{
		RefreshToken: loginResp.RefreshToken,
	})
	refreshReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(refreshBody))
	refreshReq.Header.Set("Content-Type", "application/json")
	wRefresh := httptest.NewRecorder()
	app.ServeHTTP(wRefresh, refreshReq)

	assert.Equal(t, http.StatusOK, wRefresh.Code)

	var refreshResp dto.LoginResponse
	err = json.Unmarshal(wRefresh.Body.Bytes(), &refreshResp)
	require.NoError(t, err)

	assert.NotEmpty(t, refreshResp.Token)
	assert.NotEmpty(t, refreshResp.RefreshToken)
	assert.NotEqual(t, loginResp.Token, refreshResp.Token, "new access token should differ")
	assert.NotEqual(t, loginResp.RefreshToken, refreshResp.RefreshToken, "refresh token should be rotated")

	// Step 4: Old refresh token should be rejected (rotation)
	oldRefreshBody, _ := json.Marshal(dto.RefreshRequest{
		RefreshToken: loginResp.RefreshToken,
	})
	oldRefreshReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(oldRefreshBody))
	oldRefreshReq.Header.Set("Content-Type", "application/json")
	wOldRefresh := httptest.NewRecorder()
	app.ServeHTTP(wOldRefresh, oldRefreshReq)

	assert.Equal(t, http.StatusUnauthorized, wOldRefresh.Code)

	// Step 5: Logout with the new refresh token
	logoutBody, _ := json.Marshal(dto.LogoutRequest{
		RefreshToken: refreshResp.RefreshToken,
	})
	logoutReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewBuffer(logoutBody))
	logoutReq.Header.Set("Content-Type", "application/json")
	logoutReq.Header.Set("Authorization", "Bearer "+refreshResp.Token)
	wLogout := httptest.NewRecorder()
	app.ServeHTTP(wLogout, logoutReq)

	assert.Equal(t, http.StatusOK, wLogout.Code)

	var logoutResp map[string]interface{}
	err = json.Unmarshal(wLogout.Body.Bytes(), &logoutResp)
	require.NoError(t, err)
	assert.Equal(t, "logged out successfully", logoutResp["message"])

	// Step 6: After logout, refresh token should not work
	postLogoutRefreshBody, _ := json.Marshal(dto.RefreshRequest{
		RefreshToken: refreshResp.RefreshToken,
	})
	postLogoutReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(postLogoutRefreshBody))
	postLogoutReq.Header.Set("Content-Type", "application/json")
	wPostLogout := httptest.NewRecorder()
	app.ServeHTTP(wPostLogout, postLogoutReq)

	assert.Equal(t, http.StatusUnauthorized, wPostLogout.Code)
}

func TestIntegration_Signup_DuplicateEmail(t *testing.T) {
	_, _, _, _, app := setupTestApp()

	payload := dto.SignupRequest{
		Email:    "duplicate@example.com",
		Password: "password123",
		Name:     "First User",
	}
	body, _ := json.Marshal(payload)

	// First request succeeds
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewBuffer(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	app.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusCreated, w1.Code)

	// Second request fails with 409 Conflict
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewBuffer(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	app.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusConflict, w2.Code)
}

func TestIntegration_Login_InvalidCredentials(t *testing.T) {
	_, _, _, _, app := setupTestApp()

	// Non-existent user
	loginBody, _ := json.Marshal(dto.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIntegration_Refresh_InvalidToken(t *testing.T) {
	_, _, _, _, app := setupTestApp()

	refreshBody, _ := json.Marshal(dto.RefreshRequest{
		RefreshToken: "totally_bogus_token",
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(refreshBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIntegration_APIv1_Routes(t *testing.T) {
	_, _, _, _, app := setupTestApp()

	// Verify API v1 routes also work
	signupPayload := dto.SignupRequest{
		Email:    "apiv1@example.com",
		Password: "password123",
		Name:     "API V1 User",
	}
	body, _ := json.Marshal(signupPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}
