package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Firakef1/settle/backend/internal/auth/dto"
	"github.com/Firakef1/settle/backend/internal/auth/model"
	"github.com/Firakef1/settle/backend/internal/auth/repository"
	"github.com/Firakef1/settle/backend/internal/auth/validator"
	"github.com/Firakef1/settle/backend/internal/shared/config"
	sharedService "github.com/Firakef1/settle/backend/internal/shared/service"
)

var (
	ErrInvalidCredentials     = errors.New("invalid email or password")
	ErrEmailAlreadyRegistered = errors.New("email is already registered")
	ErrInvalidRefreshToken    = errors.New("invalid or expired refresh token")
	ErrRefreshTokenRevoked    = errors.New("refresh token has been revoked")
	ErrEmailNotVerified       = errors.New("email address has not been verified")
	ErrInvalidOTP             = errors.New("invalid or expired verification code")
	ErrMaxAttemptsExceeded    = errors.New("maximum verification attempts exceeded")
)

// AuthService handles authentication logic.
type AuthService struct {
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	pwdResetOTPRepo  repository.PasswordResetOTPRepository
	emailSender      sharedService.EmailSender
	hashService      *sharedService.HashService
	jwtService       *sharedService.JWTService
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	pwdResetOTPRepo repository.PasswordResetOTPRepository,
	emailSender sharedService.EmailSender,
	hashService *sharedService.HashService,
	jwtService *sharedService.JWTService,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		pwdResetOTPRepo:  pwdResetOTPRepo,
		emailSender:      emailSender,
		hashService:      hashService,
		jwtService:       jwtService,
	}
}

func hashOTP(otp, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(otp))
	return hex.EncodeToString(h.Sum(nil))
}

func compareOTPHash(inputOTP, secret, storedHash string) bool {
	expected := hashOTP(inputOTP, secret)
	return hmac.Equal([]byte(expected), []byte(storedHash))
}

// generateRefreshToken creates a cryptographically random refresh token string.
func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// hashRefreshToken creates a deterministic SHA256 hash of a refresh token for storage/lookup.
func hashRefreshToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// Signup handles user registration.
func (s *AuthService) Signup(ctx context.Context, req *dto.SignupRequest) (*dto.UserResponseDTO, error) {
	if err := validator.ValidateSignupRequest(req); err != nil {
		return nil, err
	}

	// Normalize email before any lookup or storage.
	req.Email = normalizeEmail(req.Email)

	existingUser, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		if existingUser.EmailVerified {
			// Verified email: reject as a duplicate.
			return nil, ErrEmailAlreadyRegistered
		}
		// Unverified email: overwrite name and password so the original registrant
		// cannot keep a squatted account with a known password.
		hashedPassword, err := s.hashService.Hash(req.Password)
		if err != nil {
			return nil, err
		}
		if err := s.userRepo.UpdateUnverifiedCredentials(ctx, existingUser.ID, req.Name, hashedPassword); err != nil {
			return nil, err
		}
		existingUser.Name = req.Name
		existingUser.PasswordHash = hashedPassword
		userDTO := dto.MapUserToDTO(existingUser)
		return &userDTO, nil
	}

	hashedPassword, err := s.hashService.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &model.User{
		ID:            uuid.New().String(),
		Email:         req.Email,
		Name:          req.Name,
		PasswordHash:  hashedPassword,
		Status:        "active",
		EmailVerified: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			return nil, ErrEmailAlreadyRegistered
		}
		return nil, err
	}

	userDTO := dto.MapUserToDTO(user)
	return &userDTO, nil
}

// createAndStoreRefreshToken generates a refresh token, hashes it with SHA256, and stores it.
func (s *AuthService) createAndStoreRefreshToken(ctx context.Context, userID string) (string, error) {
	refreshTokenStr, err := generateRefreshToken()
	if err != nil {
		return "", err
	}

	tokenHash := hashRefreshToken(refreshTokenStr)

	ttl := config.AppConfig.RefreshTokenTTL
	if ttl == 0 {
		ttl = 30 * 24 * time.Hour
	}

	now := time.Now()
	refreshToken := &model.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(ttl),
		Revoked:   false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.refreshTokenRepo.StoreRefreshToken(ctx, refreshToken); err != nil {
		return "", err
	}

	return refreshTokenStr, nil
}

// Login validates user credentials and returns access + refresh tokens.
func (s *AuthService) Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error) {
	if err := validator.ValidateLoginRequest(req); err != nil {
		return nil, err
	}

	req.Email = normalizeEmail(req.Email)

	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return nil, ErrInvalidCredentials
	}

	// Check password before revealing verification state.
	if !s.hashService.Compare(req.Password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// Gate: reject unverified users after a correct password.
	if !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	return s.IssueSession(ctx, user)
}

// IssueSession creates and returns access + refresh tokens for an already-authenticated user.
// It is called by Login after credential validation and by VerificationHandler after successful verify.
func (s *AuthService) IssueSession(ctx context.Context, user *model.User) (*dto.LoginResponse, error) {
	memberships, err := s.userRepo.GetOrgMemberships(ctx, user.ID)
	if err != nil {
		memberships = []model.OrgMembership{}
	}

	primaryOrgID := ""
	primaryRole := ""
	if len(memberships) > 0 {
		primaryOrgID = memberships[0].OrgID
		primaryRole = memberships[0].Role
	}

	// Generate access token
	accessToken, err := s.jwtService.Generate(user.ID, user.Email, primaryOrgID, primaryRole)
	if err != nil {
		return nil, err
	}

	// Generate and store refresh token
	refreshTokenStr, err := s.createAndStoreRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	userDTO := dto.MapUserToDTO(user)
	orgDTOs := dto.MapMembershipsToDTO(memberships)

	return &dto.LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshTokenStr,
		User:         userDTO,
		Orgs:         orgDTOs,
	}, nil
}

// Refresh validates a refresh token and issues new access + refresh tokens (rotation).
func (s *AuthService) Refresh(ctx context.Context, req *dto.RefreshRequest) (*dto.LoginResponse, error) {
	if req.RefreshToken == "" {
		return nil, ErrInvalidRefreshToken
	}

	tokenHash := hashRefreshToken(req.RefreshToken)
	storedToken, err := s.refreshTokenRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	if storedToken.Revoked {
		// Token reuse detected - revoke all tokens for this user as a security measure
		_ = s.refreshTokenRepo.RevokeAllForUser(ctx, storedToken.UserID)
		return nil, ErrRefreshTokenRevoked
	}

	if time.Now().After(storedToken.ExpiresAt) {
		return nil, ErrInvalidRefreshToken
	}

	// Revoke old refresh token (rotation)
	_ = s.refreshTokenRepo.RevokeByTokenHash(ctx, tokenHash)

	// Find user by ID
	user, err := s.userRepo.FindByID(ctx, storedToken.UserID)
	if err != nil || user == nil {
		return nil, ErrInvalidRefreshToken
	}

	// Gate: stop pre-existing sessions from bypassing the email-verified requirement.
	if !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	return s.IssueSession(ctx, user)
}

// Logout revokes the user's refresh tokens.
func (s *AuthService) Logout(ctx context.Context, accessTokenStr string, req *dto.LogoutRequest) error {
	// Revoke the specific refresh token if provided
	if req != nil && req.RefreshToken != "" {
		tokenHash := hashRefreshToken(req.RefreshToken)
		_ = s.refreshTokenRepo.RevokeByTokenHash(ctx, tokenHash)
	}

	// If we have an access token, parse it to get user ID and revoke all refresh tokens
	if accessTokenStr != "" {
		claims, err := s.jwtService.Verify(accessTokenStr)
		if err == nil {
			_ = s.refreshTokenRepo.RevokeAllForUser(ctx, claims.UserID)
		}
	}

	return nil
}

// ForgotPassword handles initiating password reset request.
func (s *AuthService) ForgotPassword(ctx context.Context, req *dto.ForgotPasswordRequest) error {
	if err := validator.ValidateForgotPasswordRequest(req); err != nil {
		return err
	}

	email := normalizeEmail(req.Email)
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil || user == nil {
		// Generic response to prevent user enumeration
		return nil
	}

	otpStr, err := generateCode()
	if err != nil {
		return fmt.Errorf("generate otp: %w", err)
	}

	secret := config.AppConfig.PasswordResetSecret
	ttl := config.AppConfig.PasswordResetOTPTTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}

	otpHash := hashOTP(otpStr, secret)
	now := time.Now()
	otpRecord := &model.PasswordResetOTP{
		ID:           uuid.New().String(),
		UserID:       user.ID,
		OTPHash:      otpHash,
		ExpiresAt:    now.Add(ttl),
		AttemptCount: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if s.pwdResetOTPRepo != nil {
		if err := s.pwdResetOTPRepo.UpsertOTP(ctx, otpRecord); err != nil {
			return fmt.Errorf("save password reset otp: %w", err)
		}
	}

	if s.emailSender != nil {
		body := fmt.Sprintf("Hi %s,\n\nYou requested to reset your password. Use the verification code below to complete your password reset:\n\n%s\n\nIt expires in %d minutes.\nIf you did not request a password reset, please ignore this email.", user.Name, otpStr, int(ttl.Minutes()))
		_ = s.emailSender.Send(ctx, sharedService.Email{
			To:      email,
			Subject: "Reset your password",
			Body:    body,
		})
	}

	return nil
}

// ResetPassword handles verifying OTP and updating password.
func (s *AuthService) ResetPassword(ctx context.Context, req *dto.ResetPasswordRequest) error {
	if err := validator.ValidateResetPasswordRequest(req); err != nil {
		return err
	}

	email := normalizeEmail(req.Email)
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil || user == nil {
		return ErrInvalidOTP
	}

	if s.pwdResetOTPRepo == nil {
		return ErrInvalidOTP
	}

	otpRecord, err := s.pwdResetOTPRepo.FindByUserID(ctx, user.ID)
	if err != nil || otpRecord == nil {
		return ErrInvalidOTP
	}

	maxAttempts := config.AppConfig.PasswordResetMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 5
	}

	if time.Now().After(otpRecord.ExpiresAt) {
		_ = s.pwdResetOTPRepo.DeleteByUserID(ctx, user.ID)
		return ErrInvalidOTP
	}

	if otpRecord.AttemptCount >= maxAttempts {
		_ = s.pwdResetOTPRepo.DeleteByUserID(ctx, user.ID)
		return ErrMaxAttemptsExceeded
	}

	secret := config.AppConfig.PasswordResetSecret
	if !compareOTPHash(req.OTP, secret, otpRecord.OTPHash) {
		newCount, _ := s.pwdResetOTPRepo.IncrementAttemptCount(ctx, user.ID)
		if newCount >= maxAttempts {
			_ = s.pwdResetOTPRepo.DeleteByUserID(ctx, user.ID)
			return ErrMaxAttemptsExceeded
		}
		return ErrInvalidOTP
	}

	// Update password
	hashedPassword, err := s.hashService.Hash(req.NewPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := s.userRepo.UpdatePassword(ctx, user.ID, hashedPassword); err != nil {
		return err
	}

	// Single use OTP consume
	_ = s.pwdResetOTPRepo.DeleteByUserID(ctx, user.ID)

	// Revoke all refresh tokens for this user
	if s.refreshTokenRepo != nil {
		_ = s.refreshTokenRepo.RevokeAllForUser(ctx, user.ID)
	}

	return nil
}

// DeleteAccount handles authenticated user self-service account deletion.
func (s *AuthService) DeleteAccount(ctx context.Context, userID, currentPassword string) error {
	if err := validator.ValidateDeleteAccountRequest(&dto.DeleteAccountRequest{CurrentPassword: currentPassword}); err != nil {
		return err
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return repository.ErrUserNotFound
	}

	if !s.hashService.Compare(currentPassword, user.PasswordHash) {
		return ErrInvalidCredentials
	}

	if err := s.userRepo.SoftDeleteUser(ctx, userID); err != nil {
		return err
	}

	if s.refreshTokenRepo != nil {
		_ = s.refreshTokenRepo.RevokeAllForUser(ctx, userID)
	}
	if s.pwdResetOTPRepo != nil {
		_ = s.pwdResetOTPRepo.DeleteByUserID(ctx, userID)
	}

	return nil
}
