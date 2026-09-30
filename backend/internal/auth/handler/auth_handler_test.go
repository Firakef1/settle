package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Firakef1/settle/backend/internal/auth/dto"
	"github.com/Firakef1/settle/backend/internal/auth/repository"
	"github.com/Firakef1/settle/backend/internal/auth/service"
	sharedService "github.com/Firakef1/settle/backend/internal/shared/service"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// noopEmailSender is a stub EmailSender for handler tests that don't test email delivery.
type noopEmailSender struct{}

func (n *noopEmailSender) Send(_ context.Context, _ sharedService.Email) error { return nil }

func setupHandlerTest() (*AuthHandler, *repository.UserRepo, *repository.RefreshTokenRepo) {
	userRepo := repository.NewUserRepo(nil)
	refreshTokenRepo := repository.NewRefreshTokenRepo(nil)
	pwdResetRepo := repository.NewPasswordResetOTPRepo(nil)
	hashSvc := sharedService.NewHashService()
	jwtSvc := sharedService.NewJWTServiceWithSecret("handler_test_secret")
	emailSender := &noopEmailSender{}
	svc := service.NewAuthService(userRepo, refreshTokenRepo, pwdResetRepo, emailSender, hashSvc, jwtSvc)
	verCodeRepo := repository.NewVerificationCodeRepo(nil)
	verSvc := service.NewVerificationService(verCodeRepo, userRepo, emailSender, svc, 15*time.Minute)
	handler := NewAuthHandler(svc, verSvc)
	return handler, userRepo, refreshTokenRepo
}

func performRequest(handler gin.HandlerFunc, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	r.Handle(method, path, handler)

	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	c.Request = req
	r.ServeHTTP(w, req)
	return w
}

func TestHandler_Signup_Success(t *testing.T) {
	handler, _, _ := setupHandlerTest()

	req := dto.SignupRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}

	w := performRequest(handler.Signup, "POST", "/signup", req, nil)

	assert.Equal(t, http.StatusCreated, w.Code)

	var res map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &res)
	require.NoError(t, err)
	assert.Equal(t, "verification code sent to your email", res["message"])

	userData := res["user"].(map[string]interface{})
	assert.Equal(t, "test@example.com", userData["email"])
	assert.Equal(t, "Test User", userData["name"])
	assert.NotEmpty(t, userData["id"])
}

func TestHandler_Signup_InvalidJSON(t *testing.T) {
	handler, _, _ := setupHandlerTest()

	w := performRequest(handler.Signup, "POST", "/signup", "invalid json", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Signup_ValidationError(t *testing.T) {
	handler, _, _ := setupHandlerTest()

	req := dto.SignupRequest{
		Password: "password123",
		Name:     "Test User",
	}

	w := performRequest(handler.Signup, "POST", "/signup", req, nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Signup_DuplicateEmail(t *testing.T) {
	handler, userRepo, _ := setupHandlerTest()

	req := dto.SignupRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}

	// First signup: success (unverified).
	performRequest(handler.Signup, "POST", "/signup", req, nil)
	// Mark user as verified so the second signup hits the conflict path.
	userRepo.SetEmailVerifiedForTest("test@example.com", true)

	// Second signup with same email (now verified): should 409.
	w := performRequest(handler.Signup, "POST", "/signup", req, nil)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_Login_Success(t *testing.T) {
	handler, userRepo, _ := setupHandlerTest()

	signupReq := dto.SignupRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	performRequest(handler.Signup, "POST", "/signup", signupReq, nil)
	userRepo.SetEmailVerifiedForTest("test@example.com", true)

	loginReq := dto.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	w := performRequest(handler.Login, "POST", "/login", loginReq, nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var res dto.LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &res)
	require.NoError(t, err)
	assert.NotEmpty(t, res.Token)
	assert.NotEmpty(t, res.RefreshToken)
	assert.Equal(t, "test@example.com", res.User.Email)
}

func TestHandler_Login_InvalidJSON(t *testing.T) {
	handler, _, _ := setupHandlerTest()

	w := performRequest(handler.Login, "POST", "/login", "invalid json", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Login_InvalidCredentials(t *testing.T) {
	handler, _, _ := setupHandlerTest()

	signupReq := dto.SignupRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	performRequest(handler.Signup, "POST", "/signup", signupReq, nil)

	loginReq := dto.LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}

	w := performRequest(handler.Login, "POST", "/login", loginReq, nil)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Refresh_Success(t *testing.T) {
	handler, userRepo, _ := setupHandlerTest()

	signupReq := dto.SignupRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	performRequest(handler.Signup, "POST", "/signup", signupReq, nil)
	userRepo.SetEmailVerifiedForTest("test@example.com", true)

	loginReq := dto.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	wLogin := performRequest(handler.Login, "POST", "/login", loginReq, nil)
	var loginRes dto.LoginResponse
	json.Unmarshal(wLogin.Body.Bytes(), &loginRes)

	refreshReq := dto.RefreshRequest{
		RefreshToken: loginRes.RefreshToken,
	}

	w := performRequest(handler.Refresh, "POST", "/refresh", refreshReq, nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var res dto.LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &res)
	require.NoError(t, err)
	assert.NotEmpty(t, res.Token)
	assert.NotEmpty(t, res.RefreshToken)
}

func TestHandler_Refresh_InvalidJSON(t *testing.T) {
	handler, _, _ := setupHandlerTest()

	w := performRequest(handler.Refresh, "POST", "/refresh", "invalid json", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Refresh_InvalidToken(t *testing.T) {
	handler, _, _ := setupHandlerTest()

	refreshReq := dto.RefreshRequest{
		RefreshToken: "bogustoken",
	}

	w := performRequest(handler.Refresh, "POST", "/refresh", refreshReq, nil)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Logout_Success(t *testing.T) {
	handler, userRepo, _ := setupHandlerTest()

	signupReq := dto.SignupRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	performRequest(handler.Signup, "POST", "/signup", signupReq, nil)
	userRepo.SetEmailVerifiedForTest("test@example.com", true)

	loginReq := dto.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	wLogin := performRequest(handler.Login, "POST", "/login", loginReq, nil)
	var loginRes dto.LoginResponse
	json.Unmarshal(wLogin.Body.Bytes(), &loginRes)

	logoutReq := dto.LogoutRequest{
		RefreshToken: loginRes.RefreshToken,
	}

	headers := map[string]string{
		"Authorization": "Bearer " + loginRes.Token,
	}

	w := performRequest(handler.Logout, "POST", "/logout", logoutReq, headers)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Logout_NoBody(t *testing.T) {
	handler, userRepo, _ := setupHandlerTest()

	signupReq := dto.SignupRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	performRequest(handler.Signup, "POST", "/signup", signupReq, nil)
	userRepo.SetEmailVerifiedForTest("test@example.com", true)

	loginReq := dto.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	wLogin := performRequest(handler.Login, "POST", "/login", loginReq, nil)
	var loginRes dto.LoginResponse
	json.Unmarshal(wLogin.Body.Bytes(), &loginRes)

	headers := map[string]string{
		"Authorization": "Bearer " + loginRes.Token,
	}

	w := performRequest(handler.Logout, "POST", "/logout", nil, headers)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ForgotPassword(t *testing.T) {
	handler, _, _ := setupHandlerTest()

	// Generic success for any valid email
	req := dto.ForgotPasswordRequest{Email: "user@example.com"}
	w := performRequest(handler.ForgotPassword, "POST", "/forgot-password", req, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	// Invalid email format 400
	badReq := dto.ForgotPasswordRequest{Email: "invalid-email"}
	wBad := performRequest(handler.ForgotPassword, "POST", "/forgot-password", badReq, nil)
	assert.Equal(t, http.StatusBadRequest, wBad.Code)
}

func TestHandler_ResetPassword_ValidationFailure(t *testing.T) {
	handler, _, _ := setupHandlerTest()

	// Short password
	req := dto.ResetPasswordRequest{
		Email:       "user@example.com",
		OTP:         "123456",
		NewPassword: "short",
	}
	w := performRequest(handler.ResetPassword, "POST", "/reset-password", req, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Invalid OTP format
	reqOTP := dto.ResetPasswordRequest{
		Email:       "user@example.com",
		OTP:         "abc",
		NewPassword: "newpassword123",
	}
	wOTP := performRequest(handler.ResetPassword, "POST", "/reset-password", reqOTP, nil)
	assert.Equal(t, http.StatusBadRequest, wOTP.Code)
}
