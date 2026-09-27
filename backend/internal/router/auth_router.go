package router

import (
	"github.com/gin-gonic/gin"

	authHandler "github.com/Firakef1/settle/backend/internal/auth/handler"
)

// RegisterAuthRoutes attaches auth endpoints to the given router group.
func RegisterAuthRoutes(rg *gin.RouterGroup, authH *authHandler.AuthHandler) {
	if authH == nil {
		return
	}
	auth := rg.Group("/auth")
	{
		auth.POST("/signup", authH.Signup)
		auth.POST("/login", authH.Login)
		auth.POST("/logout", authH.Logout)
		auth.POST("/refresh", authH.Refresh)
	}
}
