package router

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	approvalhandler "github.com/Firakef1/settle/backend/internal/approvals/handler"
	authHandler "github.com/Firakef1/settle/backend/internal/auth/handler"
	orghandler "github.com/Firakef1/settle/backend/internal/organaization/handler"
	requestshandler "github.com/Firakef1/settle/backend/internal/requests/handler"
)

// Handlers holds all domain handlers to be registered in the router.
type Handlers struct {
	Auth         *authHandler.AuthHandler
	Verification *authHandler.VerificationHandler
	OAuth        *authHandler.OAuthHandler
	Org          *orghandler.OrgHandler
	Member       *orghandler.MemberHandler
	Invitation   *orghandler.InvitationHandler
	Audit        *orghandler.AuditHandler
	Dashboard    *orghandler.DashboardHandler
	Billing      *orghandler.BillingHandler
	Approval     *approvalhandler.ApprovalHandler
	Request      *requestshandler.RequestHandler
}

// SetupRouter initializes the Gin router and registers all domain routes.
func SetupRouter(h *Handlers) *gin.Engine {
	route := gin.Default()

	// Swagger documentation route
	route.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	if h == nil {
		return route
	}

	// API v1 group - all domain routes mounted under /api/v1
	apiV1 := route.Group("/api/v1")
	{
		RegisterAuthRoutes(apiV1, h.Auth, h.Verification, h.OAuth)
		RegisterOrganizationRoutes(apiV1, h.Org, h.Member, h.Invitation, h.Audit, h.Dashboard, h.Billing)
		RegisterApprovalRoutes(apiV1, h.Approval)
		RegisterRequestRoutes(apiV1, h.Request)
	}

	// v1 group - backwards compatibility
	v1 := route.Group("/v1")
	{
		RegisterAuthRoutes(v1, h.Auth, h.Verification, h.OAuth)
		RegisterOrganizationRoutes(v1, h.Org, h.Member, h.Invitation, h.Audit, h.Dashboard, h.Billing)
		RegisterApprovalRoutes(v1, h.Approval)
		RegisterRequestRoutes(v1, h.Request)
	}

	return route
}
