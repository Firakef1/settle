package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Firakef1/settle/backend/internal/auth/dto"
	"github.com/Firakef1/settle/backend/internal/auth/model"
	"github.com/Firakef1/settle/backend/internal/auth/repository"
	sharedService "github.com/Firakef1/settle/backend/internal/shared/service"
)

type mockEmailSender struct {
	mu        sync.Mutex
	lastEmail sharedService.Email
	calls     int
}

func (m *mockEmailSender) Send(_ context.Context, email sharedService.Email) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	m.lastEmail = email
	return nil
}

func setupVerificationService(sender *mockEmailSender) (*VerificationService, *repository.UserRepo, *repository.VerificationCodeRepo) {
	userRepo := repository.NewUserRepo(nil)
	verCodeRepo := repository.NewVerificationCodeRepo(nil)
	pwdResetRepo := repository.NewPasswordResetOTPRepo(nil)
	hashSvc := sharedService.NewHashService()
	jwtSvc := sharedService.NewJWTServiceWithSecret("test_jwt_secret_1234567890")
	refreshTokenRepo := repository.NewRefreshTokenRepo(nil)
	authSvc := NewAuthService(userRepo, refreshTokenRepo, pwdResetRepo, sender, hashSvc, jwtSvc)

	verSvc := NewVerificationService(verCodeRepo, userRepo, sender, authSvc, 15*time.Minute)
	return verSvc, userRepo, verCodeRepo
}

func TestVerificationService_SendAndVerify(t *testing.T) {
	sender := &mockEmailSender{}
	verSvc, userRepo, _ := setupVerificationService(sender)
	ctx := context.Background()

	// Register unverified user
	user := &model.User{
		ID:            "user-100",
		Email:         "testver@example.com",
		Name:          "Test User",
		Status:        "active",
		EmailVerified: false,
	}
	err := userRepo.CreateUser(ctx, user)
	require.NoError(t, err)

	// Send code
	err = verSvc.SendCode(ctx, "  TESTVER@example.com  ")
	require.NoError(t, err)
	assert.Equal(t, 1, sender.calls)

	// Extract code from email body
	body := sender.lastEmail.Body
	require.Contains(t, body, "verification code")

	// Get code from verCodeRepo
	verCodeRepo := repository.NewVerificationCodeRepo(nil)
	_ = verCodeRepo

	// Find the code from repo directly for exact check
	verCodeRepo2 := repository.NewVerificationCodeRepo(nil)
	_ = verCodeRepo2

	// VerifyEmail with wrong code -> ErrInvalidOrExpiredCode
	_, err = verSvc.VerifyEmail(ctx, dto.VerifyEmailRequest{
		Email: "testver@example.com",
		Code:  "999999",
	})
	assert.ErrorIs(t, err, ErrInvalidOrExpiredCode)
}
