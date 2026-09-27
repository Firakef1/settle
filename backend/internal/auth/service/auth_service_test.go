package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Firakef1/settle/backend/internal/auth/dto"
	"github.com/Firakef1/settle/backend/internal/auth/repository"
	sharedService "github.com/Firakef1/settle/backend/internal/shared/service"
)

func setupTestService() (*AuthService, *repository.UserRepo, *repository.RefreshTokenRepo, *sharedService.JWTService) {
	userRepo := repository.NewUserRepo(nil)
	refreshTokenRepo := repository.NewRefreshTokenRepo(nil)
	hashSvc := sharedService.NewHashService()
	jwtSvc := sharedService.NewJWTServiceWithSecret("test_secret_key_1234567890")
	svc := NewAuthService(userRepo, refreshTokenRepo, hashSvc, jwtSvc)
	return svc, userRepo, refreshTokenRepo, jwtSvc
}

func TestSignup_Success(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	req := &dto.SignupRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}

	userDTO, err := svc.Signup(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", userDTO.Email)
	assert.Equal(t, "Test User", userDTO.Name)
	assert.NotEmpty(t, userDTO.ID)
}

func TestSignup_DuplicateEmail(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	req := &dto.SignupRequest{
		Email:    "duplicate@example.com",
		Password: "password123",
		Name:     "User One",
	}

	_, err := svc.Signup(ctx, req)
	require.NoError(t, err)

	_, err = svc.Signup(ctx, req)
	assert.ErrorIs(t, err, ErrEmailAlreadyRegistered)
}

func TestSignup_ValidationErrors(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	tests := []struct {
		name    string
		req     *dto.SignupRequest
		wantErr bool
	}{
		{
			name: "short password",
			req: &dto.SignupRequest{
				Email: "user@example.com", Password: "short", Name: "Test",
			},
			wantErr: true,
		},
		{
			name: "invalid email",
			req: &dto.SignupRequest{
				Email: "not-an-email", Password: "validpassword123", Name: "Test",
			},
			wantErr: true,
		},
		{
			name: "missing name",
			req: &dto.SignupRequest{
				Email: "valid@example.com", Password: "validpassword123", Name: "",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Signup(ctx, tc.req)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLogin_Success_ReturnsTokens(t *testing.T) {
	svc, _, _, jwtSvc := setupTestService()
	ctx := context.Background()

	// Sign up first
	signupReq := &dto.SignupRequest{
		Email:    "login@example.com",
		Password: "password123",
		Name:     "Login User",
	}
	_, err := svc.Signup(ctx, signupReq)
	require.NoError(t, err)

	// Login
	loginReq := &dto.LoginRequest{
		Email:    "login@example.com",
		Password: "password123",
	}
	resp, err := svc.Login(ctx, loginReq)
	require.NoError(t, err)

	assert.NotEmpty(t, resp.Token, "access token should not be empty")
	assert.NotEmpty(t, resp.RefreshToken, "refresh token should not be empty")
	assert.Equal(t, "login@example.com", resp.User.Email)

	// Verify access token is valid JWT
	claims, err := jwtSvc.Verify(resp.Token)
	require.NoError(t, err)
	assert.Equal(t, "login@example.com", claims.Email)
	assert.NotEmpty(t, claims.ID, "JWT should have a JTI")
}

func TestLogin_InvalidCredentials(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	// Non-existent user
	loginReq := &dto.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	}
	_, err := svc.Login(ctx, loginReq)
	assert.ErrorIs(t, err, ErrInvalidCredentials)

	// Sign up and try wrong password
	signupReq := &dto.SignupRequest{
		Email:    "wrongpass@example.com",
		Password: "password123",
		Name:     "Wrong Pass",
	}
	_, err = svc.Signup(ctx, signupReq)
	require.NoError(t, err)

	loginReq = &dto.LoginRequest{
		Email:    "wrongpass@example.com",
		Password: "wrongpassword",
	}
	_, err = svc.Login(ctx, loginReq)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestRefresh_RotatesTokens(t *testing.T) {
	svc, _, _, jwtSvc := setupTestService()
	ctx := context.Background()

	// Sign up and login
	_, err := svc.Signup(ctx, &dto.SignupRequest{
		Email: "refresh@example.com", Password: "password123", Name: "Refresh User",
	})
	require.NoError(t, err)

	loginResp, err := svc.Login(ctx, &dto.LoginRequest{
		Email: "refresh@example.com", Password: "password123",
	})
	require.NoError(t, err)
	originalRefreshToken := loginResp.RefreshToken
	originalAccessToken := loginResp.Token

	// Refresh
	refreshResp, err := svc.Refresh(ctx, &dto.RefreshRequest{
		RefreshToken: originalRefreshToken,
	})
	require.NoError(t, err)

	// New tokens should be different from the original
	assert.NotEqual(t, originalAccessToken, refreshResp.Token, "new access token should differ")
	assert.NotEqual(t, originalRefreshToken, refreshResp.RefreshToken, "new refresh token should differ (rotation)")
	assert.NotEmpty(t, refreshResp.Token)
	assert.NotEmpty(t, refreshResp.RefreshToken)
	assert.Equal(t, "refresh@example.com", refreshResp.User.Email)

	// New access token should be valid
	claims, err := jwtSvc.Verify(refreshResp.Token)
	require.NoError(t, err)
	assert.Equal(t, "refresh@example.com", claims.Email)
}

func TestRefresh_OldTokenRevoked(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	// Sign up and login
	_, err := svc.Signup(ctx, &dto.SignupRequest{
		Email: "revoke@example.com", Password: "password123", Name: "Revoke User",
	})
	require.NoError(t, err)

	loginResp, err := svc.Login(ctx, &dto.LoginRequest{
		Email: "revoke@example.com", Password: "password123",
	})
	require.NoError(t, err)
	originalRefreshToken := loginResp.RefreshToken

	// Refresh once (old token gets revoked)
	_, err = svc.Refresh(ctx, &dto.RefreshRequest{
		RefreshToken: originalRefreshToken,
	})
	require.NoError(t, err)

	// Try to reuse old refresh token (should fail - token reuse detection)
	_, err = svc.Refresh(ctx, &dto.RefreshRequest{
		RefreshToken: originalRefreshToken,
	})
	assert.ErrorIs(t, err, ErrRefreshTokenRevoked)
}

func TestRefresh_InvalidToken(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	// Try with bogus token
	_, err := svc.Refresh(ctx, &dto.RefreshRequest{
		RefreshToken: "bogus_token_string",
	})
	assert.ErrorIs(t, err, ErrInvalidRefreshToken)

	// Try with empty token
	_, err = svc.Refresh(ctx, &dto.RefreshRequest{
		RefreshToken: "",
	})
	assert.ErrorIs(t, err, ErrInvalidRefreshToken)
}

func TestLogout_RevokesTokens(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	// Sign up and login
	_, err := svc.Signup(ctx, &dto.SignupRequest{
		Email: "logout@example.com", Password: "password123", Name: "Logout User",
	})
	require.NoError(t, err)

	loginResp, err := svc.Login(ctx, &dto.LoginRequest{
		Email: "logout@example.com", Password: "password123",
	})
	require.NoError(t, err)

	// Logout with both access token and refresh token
	err = svc.Logout(ctx, loginResp.Token, &dto.LogoutRequest{
		RefreshToken: loginResp.RefreshToken,
	})
	require.NoError(t, err)

	// Try to refresh with the revoked token
	_, err = svc.Refresh(ctx, &dto.RefreshRequest{
		RefreshToken: loginResp.RefreshToken,
	})
	assert.Error(t, err, "refresh should fail after logout")
}

func TestLogout_WithoutRefreshToken(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	// Logout with empty should not error
	err := svc.Logout(ctx, "", nil)
	require.NoError(t, err)

	err = svc.Logout(ctx, "", &dto.LogoutRequest{})
	require.NoError(t, err)
}
