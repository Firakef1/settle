package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Firakef1/settle/backend/internal/auth/dto"
	"github.com/Firakef1/settle/backend/internal/auth/repository"
	"github.com/Firakef1/settle/backend/internal/auth/service"
	"github.com/Firakef1/settle/backend/internal/auth/validator"
	"github.com/Firakef1/settle/backend/internal/shared/middleware"
)

// AuthHandler exposes HTTP handlers for authentication.
type AuthHandler struct {
	authService *service.AuthService
	verService  *service.VerificationService
}

// NewAuthHandler creates a new AuthHandler instance.
func NewAuthHandler(authService *service.AuthService, verService *service.VerificationService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		verService:  verService,
	}
}

// Signup godoc
// @Summary      Register a new user
// @Description  Creates a new user account and sends a verification code to the email.
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
		log.Printf("[ERROR] Signup user creation failed for email %s: %v", req.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create user account: %v", err)})
		return
	}

	// Send verification code.
	if err := h.verService.SendCode(c.Request.Context(), userDTO.Email); err != nil {
		log.Printf("[ERROR] Signup SendCode failed for email %s: %v", userDTO.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("could not send verification email: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "verification code sent to your email",
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
// @Failure      403      {object}  map[string]string
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
		if errors.Is(err, service.ErrEmailNotVerified) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "email address has not been verified",
				"code":  "email_not_verified",
			})
			return
		}
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
// @Failure      403      {object}  map[string]string
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
		if errors.Is(err, service.ErrEmailNotVerified) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "email address has not been verified",
				"code":  "email_not_verified",
			})
			return
		}
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

// ForgotPassword godoc
// @Summary      Request password reset
// @Description  Sends a password reset OTP to the user's email address if registered.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.ForgotPasswordRequest  true  "Forgot password request"
// @Success      200      {object}  map[string]string
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /auth/forgot-password [post]
// ForgotPassword handles POST /auth/forgot-password
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body format"})
		return
	}

	if err := h.authService.ForgotPassword(c.Request.Context(), &req); err != nil {
		if errors.Is(err, validator.ErrInvalidEmail) || errors.Is(err, validator.ErrEmailRequired) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.Printf("[ERROR] ForgotPassword failed for email %s: %v", req.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process forgot password request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "If that email is registered, a password reset OTP has been sent.",
	})
}

// ResetPassword godoc
// @Summary      Reset password with OTP
// @Description  Verifies password reset OTP and sets a new password.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.ResetPasswordRequest  true  "Reset password details"
// @Success      200      {object}  map[string]string
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /auth/reset-password [post]
// ResetPassword handles POST /auth/reset-password
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body format"})
		return
	}

	if err := h.authService.ResetPassword(c.Request.Context(), &req); err != nil {
		if errors.Is(err, service.ErrInvalidOTP) ||
			errors.Is(err, service.ErrMaxAttemptsExceeded) ||
			errors.Is(err, validator.ErrInvalidEmail) ||
			errors.Is(err, validator.ErrEmailRequired) ||
			errors.Is(err, validator.ErrOTPRequired) ||
			errors.Is(err, validator.ErrInvalidOTP) ||
			errors.Is(err, validator.ErrPasswordTooShort) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.Printf("[ERROR] ResetPassword failed for email %s: %v", req.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "password reset failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "password reset successfully",
	})
}

// DeleteAccount godoc
// @Summary      Delete user account
// @Description  Soft deletes the authenticated user's account and revokes active tokens.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.DeleteAccountRequest  true  "Current password confirmation"
// @Success      200      {object}  map[string]string
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Failure      403      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /auth/me [delete]
// DeleteAccount handles DELETE /auth/me
func (h *AuthHandler) DeleteAccount(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req dto.DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "current_password is required"})
		return
	}

	if err := h.authService.DeleteAccount(c.Request.Context(), userID, req.CurrentPassword); err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid current password"})
			return
		}
		if errors.Is(err, repository.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		if errors.Is(err, validator.ErrCurrentPasswordRequired) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.Printf("[ERROR] DeleteAccount failed for userID %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete account"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "account deleted successfully",
	})
}
