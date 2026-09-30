package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Firakef1/settle/backend/internal/auth/dto"
	"github.com/Firakef1/settle/backend/internal/auth/model"
	"github.com/Firakef1/settle/backend/internal/auth/repository"
	sharedService "github.com/Firakef1/settle/backend/internal/shared/service"
)

var (
	ErrInvalidOrExpiredCode = errors.New("invalid or expired verification code")
	ErrInvalidCode          = ErrInvalidOrExpiredCode
)

type VerificationCodeReader interface {
	FindByEmailAndCode(ctx context.Context, email, code string) (*model.VerificationCode, error)
	DeleteByEmail(ctx context.Context, email string) error
}

type UserVerifier interface {
	MarkEmailAsVerified(ctx context.Context, email string) error
}

type VerificationService struct {
	verCodeRepo repository.VerificationCodeRepository
	userRepo    repository.UserRepository
	emailSender sharedService.EmailSender
	authSvc     *AuthService
	codeTTL     time.Duration
}

func NewVerificationService(
	verCodeRepo repository.VerificationCodeRepository,
	userRepo repository.UserRepository,
	emailSender sharedService.EmailSender,
	authSvc *AuthService,
	codeTTL time.Duration,
) *VerificationService {
	if codeTTL <= 0 {
		codeTTL = 15 * time.Minute
	}
	return &VerificationService{
		verCodeRepo: verCodeRepo,
		userRepo:    userRepo,
		emailSender: emailSender,
		authSvc:     authSvc,
		codeTTL:     codeTTL,
	}
}

func generateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func (s *VerificationService) SendCode(ctx context.Context, email string) error {
	email = normalizeEmail(email)

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil || user == nil {
		return nil
	}
	if user.EmailVerified {
		return nil
	}

	codeStr, err := generateCode()
	if err != nil {
		return fmt.Errorf("generate verification code: %w", err)
	}

	now := time.Now()
	verCode := &model.VerificationCode{
		ID:        uuid.New().String(),
		Email:     email,
		Code:      codeStr,
		ExpiresAt: now.Add(s.codeTTL),
		CreatedAt: now,
	}

	if err := s.verCodeRepo.Save(ctx, verCode); err != nil {
		return fmt.Errorf("save verification code: %w", err)
	}

	body := fmt.Sprintf("Hi %s,\n\nUse the verification code below to confirm your email address:\n\n%s\n\nIt expires in %d minutes.", user.Name, codeStr, int(s.codeTTL.Minutes()))
	if err := s.emailSender.Send(ctx, sharedService.Email{
		To:      email,
		Subject: "Your Settle verification code",
		Body:    body,
	}); err != nil {
		return err
	}

	return nil
}

func (s *VerificationService) VerifyEmail(ctx context.Context, req dto.VerifyEmailRequest) (*dto.VerifyEmailResponse, error) {
	email := normalizeEmail(req.Email)
	code := strings.TrimSpace(req.GetCode())

	record, err := s.verCodeRepo.FindByEmailAndCode(ctx, email, code)
	if err != nil || record == nil || time.Now().After(record.ExpiresAt) {
		return nil, ErrInvalidOrExpiredCode
	}

	if err := s.userRepo.MarkEmailAsVerified(ctx, email); err != nil {
		return nil, err
	}

	_ = s.verCodeRepo.DeleteByEmail(ctx, email)

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil || user == nil {
		return &dto.VerifyEmailResponse{Message: "Email verified successfully"}, nil
	}

	var sessionToken, refreshToken string
	var userDTO dto.UserResponseDTO
	var orgDTOs []dto.OrgMembershipDTO

	if s.authSvc != nil {
		loginResp, err := s.authSvc.IssueSession(ctx, user)
		if err == nil && loginResp != nil {
			sessionToken = loginResp.Token
			refreshToken = loginResp.RefreshToken
			userDTO = loginResp.User
			orgDTOs = loginResp.Orgs
		}
	}

	return &dto.VerifyEmailResponse{
		Message:      "Email verified successfully",
		Token:        sessionToken,
		RefreshToken: refreshToken,
		User:         &userDTO,
		Orgs:         orgDTOs,
	}, nil
}

func (s *VerificationService) VerifyCode(ctx context.Context, email, code string) (*model.User, error) {
	resp, err := s.VerifyEmail(ctx, dto.VerifyEmailRequest{Email: email, VerificationCode: code})
	if err != nil {
		return nil, err
	}
	_ = resp
	return s.userRepo.FindByEmail(ctx, normalizeEmail(email))
}

func (s *VerificationService) RunCleanup(ctx context.Context, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = s.verCodeRepo.DeleteExpired(ctx)
		}
	}
}
