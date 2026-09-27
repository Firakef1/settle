package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Firakef1/settle/backend/internal/auth/dto"
	"github.com/Firakef1/settle/backend/internal/auth/service"
	"github.com/Firakef1/settle/backend/internal/auth/validator"
)

// AuthHandler exposes HTTP handlers for authentication.
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new AuthHandler instance.
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Signup godoc
// @Summary      Register a new user
// @Description  Creates a new user account and returns the user object.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.SignupRequest  true  "Signup details"
// @Success      201      {object}  map[string]interface{}
// @Failure      400      {object}  map[string]string
// @Failure      409      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /auth/signup [post]
// Signup handles POST /auth/signup
func (h *AuthHandler) Signup(c *gin.Context) {
	var req dto.SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body format"})
		return
	}

	userDTO, err := h.authService.Signup(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrEmailAlreadyRegistered) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, validator.ErrInvalidEmail) ||
			errors.Is(err, validator.ErrPasswordTooShort) ||
			errors.Is(err, validator.ErrNameRequired) ||
			errors.Is(err, validator.ErrEmailRequired) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user account"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "user created successfully",
		"user":    userDTO,
	})
}

// Login godoc
// @Summary      User login
// @Description  Authenticates a user and returns access and refresh tokens.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.LoginRequest  true  "Login credentials"
// @Success      200      {object}  dto.LoginResponse
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /auth/login [post]
// Login handles POST /auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body format"})
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, validator.ErrInvalidEmail) ||
			errors.Is(err, validator.ErrEmailRequired) ||
			errors.Is(err, validator.ErrPasswordRequired) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Refresh godoc
// @Summary      Refresh access token
// @Description  Uses a valid refresh token to issue a new access token and a rotated refresh token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.RefreshRequest  true  "Refresh token"
// @Success      200      {object}  dto.LoginResponse
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /auth/refresh [post]
// Refresh handles POST /auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
		return
	}

	resp, err := h.authService.Refresh(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrRefreshTokenRevoked) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token has been revoked"})
			return
		}
		if errors.Is(err, service.ErrInvalidRefreshToken) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token refresh failed"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Logout godoc
// @Summary      User logout
// @Description  Revokes the user's tokens. Optionally invalidates the refresh token if provided.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.LogoutRequest  false  "Refresh token to revoke"
// @Success      200      {object}  map[string]string
// @Router       /auth/logout [post]
// Logout handles POST /auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// Extract access token from Authorization header
	authHeader := c.GetHeader("Authorization")
	accessToken := ""
	if strings.HasPrefix(authHeader, "Bearer ") {
		accessToken = strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Parse optional JSON body for refresh token
	var req dto.LogoutRequest
	_ = c.ShouldBindJSON(&req)

	_ = h.authService.Logout(c.Request.Context(), accessToken, &req)

	c.JSON(http.StatusOK, gin.H{
		"message": "logged out successfully",
	})
}
