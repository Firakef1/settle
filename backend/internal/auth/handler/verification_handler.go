package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Firakef1/settle/backend/internal/auth/dto"
	"github.com/Firakef1/settle/backend/internal/auth/service"
)

// VerificationHandler handles email verification endpoints.
type VerificationHandler struct {
	verSvc  *service.VerificationService
	authSvc *service.AuthService
}

// NewVerificationHandler creates a new VerificationHandler.
func NewVerificationHandler(verSvc *service.VerificationService, authSvc *service.AuthService) *VerificationHandler {
	return &VerificationHandler{
		verSvc:  verSvc,
		authSvc: authSvc,
	}
}

// VerifyEmail godoc
// @Summary      Verify email address
// @Description  Verifies a user's email with a 6-digit code. On success, returns auth tokens (user is signed in).
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.VerifyEmailRequest  true  "Email and code"
// @Success      200      {object}  dto.VerifyEmailResponse
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /auth/verify-email [post]
func (h *VerificationHandler) VerifyEmail(c *gin.Context) {
	var req dto.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	resp, err := h.verSvc.VerifyEmail(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidOrExpiredCode) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired verification code"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "verification failed"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ResendCode godoc
// @Summary      Resend verification code
// @Description  Sends a new verification code to the given email. Always returns 200 to prevent enumeration.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.ResendCodeRequest  true  "Email address"
// @Success      200      {object}  map[string]string
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /auth/resend-verification [post]
func (h *VerificationHandler) ResendCode(c *gin.Context) {
	var req dto.ResendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.verSvc.SendCode(c.Request.Context(), req.Email); err != nil {
		log.Printf("[ERROR] ResendCode failed for email %s: %v", req.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("could not send verification email: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "if this account exists and is unverified, a verification code has been sent",
	})
}
