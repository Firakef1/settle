package handler

import (
	"context"
	"net/http"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Firakef1/settle/backend/internal/auth/dto"
	"github.com/Firakef1/settle/backend/internal/auth/repository"
	"github.com/Firakef1/settle/backend/internal/auth/service"
	sharedService "github.com/Firakef1/settle/backend/internal/shared/service"
)

type capturerEmailSender struct {
	mu        sync.Mutex
	lastEmail sharedService.Email
	calls     int
}

func (c *capturerEmailSender) Send(_ context.Context, email sharedService.Email) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	c.lastEmail = email
	return nil
}

var codeRe = regexp.MustCompile(`\n(\d{6})\n`)

func (c *capturerEmailSender) extractCode() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	match := codeRe.FindStringSubmatch(c.lastEmail.Body)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

func setupVerificationHandlerTest() (*VerificationHandler, *AuthHandler, *capturerEmailSender, *service.VerificationService, *service.AuthService, *repository.UserRepo) {
	userRepo := repository.NewUserRepo(nil)
	refreshTokenRepo := repository.NewRefreshTokenRepo(nil)
	pwdResetRepo := repository.NewPasswordResetOTPRepo(nil)
	hashSvc := sharedService.NewHashService()
	jwtSvc := sharedService.NewJWTServiceWithSecret("verification_handler_test_secret")
	sender := &capturerEmailSender{}
	authSvc := service.NewAuthService(userRepo, refreshTokenRepo, pwdResetRepo, sender, hashSvc, jwtSvc)

	verCodeRepo := repository.NewVerificationCodeRepo(nil)
	verSvc := service.NewVerificationService(verCodeRepo, userRepo, sender, authSvc, 15*time.Minute)

	verHandler := NewVerificationHandler(verSvc, authSvc)
	authHandler := NewAuthHandler(authSvc, verSvc)

	return verHandler, authHandler, sender, verSvc, authSvc, userRepo
}

func TestVerificationHandler_VerifyEmail_Success(t *testing.T) {
	verH, authH, sender, _, _, _ := setupVerificationHandlerTest()

	signupReq := dto.SignupRequest{
		Email:    "testverify@example.com",
		Password: "password123",
		Name:     "Test Verify",
	}
	wSignup := performRequest(authH.Signup, "POST", "/signup", signupReq, nil)
	require.Equal(t, http.StatusCreated, wSignup.Code)

	code := sender.extractCode()
	require.NotEmpty(t, code)

	verifyReq := dto.VerifyEmailRequest{
		Email:            "testverify@example.com",
		VerificationCode: code,
	}
	wVerify := performRequest(verH.VerifyEmail, "POST", "/verify-email", verifyReq, nil)
	assert.Equal(t, http.StatusOK, wVerify.Code)
}

func TestVerificationHandler_VerifyEmail_InvalidCode(t *testing.T) {
	verH, authH, _, _, _, _ := setupVerificationHandlerTest()

	signupReq := dto.SignupRequest{
		Email:    "wrongcode@example.com",
		Password: "password123",
		Name:     "Wrong Code",
	}
	wSignup := performRequest(authH.Signup, "POST", "/signup", signupReq, nil)
	require.Equal(t, http.StatusCreated, wSignup.Code)

	verifyReq := dto.VerifyEmailRequest{
		Email:            "wrongcode@example.com",
		VerificationCode: "999999",
	}
	wVerify := performRequest(verH.VerifyEmail, "POST", "/verify-email", verifyReq, nil)
	assert.Equal(t, http.StatusBadRequest, wVerify.Code)
}

func TestVerificationHandler_VerifyEmail_InvalidJSON(t *testing.T) {
	verH, _, _, _, _, _ := setupVerificationHandlerTest()

	w := performRequest(verH.VerifyEmail, "POST", "/verify-email", "invalid json", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVerificationHandler_ResendCode_Success(t *testing.T) {
	verH, authH, sender, _, _, _ := setupVerificationHandlerTest()

	signupReq := dto.SignupRequest{
		Email:    "resend@example.com",
		Password: "password123",
		Name:     "Resend User",
	}
	wSignup := performRequest(authH.Signup, "POST", "/signup", signupReq, nil)
	require.Equal(t, http.StatusCreated, wSignup.Code)

	resendReq := dto.ResendCodeRequest{
		Email: "resend@example.com",
	}
	wResend := performRequest(verH.ResendCode, "POST", "/resend-verification", resendReq, nil)
	assert.Equal(t, http.StatusOK, wResend.Code)
	assert.Equal(t, 2, sender.calls)
}

func TestVerificationHandler_ResendCode_UnknownEmail(t *testing.T) {
	verH, _, sender, _, _, _ := setupVerificationHandlerTest()

	resendReq := dto.ResendCodeRequest{
		Email: "nonexistent@example.com",
	}
	wResend := performRequest(verH.ResendCode, "POST", "/resend-verification", resendReq, nil)
	assert.Equal(t, http.StatusOK, wResend.Code)
	assert.Equal(t, 0, sender.calls)
}

func TestVerificationHandler_ResendCode_InvalidJSON(t *testing.T) {
	verH, _, _, _, _, _ := setupVerificationHandlerTest()

	w := performRequest(verH.ResendCode, "POST", "/resend-verification", "invalid json", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
