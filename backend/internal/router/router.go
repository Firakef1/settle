package router

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	authHandler "github.com/Firakef1/settle/backend/internal/auth/handler"
)

// Handlers holds all domain handlers to be registered in the router.
type Handlers struct {
	Auth *authHandler.AuthHandler
	// TODO: Add other domain handlers here (Requests, Receipts, Billing, etc.)
}

// SetupRouter initializes the Gin router and registers all domain routes.
func SetupRouter(h *Handlers) *gin.Engine {
	route := gin.Default()

	// Swagger documentation route
	route.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 group - all domain routes mounted under /api/v1
	public := route.Group("/api/v1")
	{
		RegisterAuthRoutes(public, h.Auth)
		// TODO: Register other domain routes here (requests, receipts, etc.)
	}

	return route
}
