package router

import (
	"github.com/gin-gonic/gin"

	authHandler "github.com/Firakef1/settle/backend/internal/auth/handler"
	"github.com/Firakef1/settle/backend/internal/shared/middleware"
)

// RegisterAuthRoutes attaches auth endpoints to the given router group.
func RegisterAuthRoutes(rg *gin.RouterGroup, authH *authHandler.AuthHandler, verH *authHandler.VerificationHandler) {
	if authH == nil {
		return
	}
	auth := rg.Group("/auth")
	{
		auth.POST("/signup", authH.Signup)
		auth.POST("/login", authH.Login)
		auth.POST("/logout", authH.Logout)
		auth.POST("/refresh", authH.Refresh)
		auth.POST("/forgot-password", authH.ForgotPassword)
		auth.POST("/reset-password", authH.ResetPassword)
		if verH != nil {
			auth.POST("/verify-email", verH.VerifyEmail)
			auth.POST("/resend-verification", verH.ResendCode)
		}

		protected := auth.Group("")
		protected.Use(middleware.AuthRequired())
		{
			protected.DELETE("/me", authH.DeleteAccount)
		}
	}
}
