package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
)

// AuthService handles authentication logic.
type AuthService struct {
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	hashService      *sharedService.HashService
	jwtService       *sharedService.JWTService
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	hashService *sharedService.HashService,
	jwtService *sharedService.JWTService,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		hashService:      hashService,
		jwtService:       jwtService,
	}
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

	existingUser, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, ErrEmailAlreadyRegistered
	}

	hashedPassword, err := s.hashService.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &model.User{
		ID:           uuid.New().String(),
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: hashedPassword,
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
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

	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return nil, ErrInvalidCredentials
	}

	if !s.hashService.Compare(req.Password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

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

	// Get org memberships
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

	// Generate new access token
	accessToken, err := s.jwtService.Generate(user.ID, user.Email, primaryOrgID, primaryRole)
	if err != nil {
		return nil, err
	}

	// Generate new refresh token (rotation)
	newRefreshTokenStr, err := s.createAndStoreRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	userDTO := dto.MapUserToDTO(user)
	orgDTOs := dto.MapMembershipsToDTO(memberships)

	return &dto.LoginResponse{
		Token:        accessToken,
		RefreshToken: newRefreshTokenStr,
		User:         userDTO,
		Orgs:         orgDTOs,
	}, nil
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
