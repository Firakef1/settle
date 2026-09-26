// Package router registers HTTP routes.
package router

import (
	orghandler "github.com/Firakef1/settle/backend/internal/organaization/handler"
	"github.com/Firakef1/settle/backend/internal/shared/middleware"
	"github.com/gin-gonic/gin"
)

// SetupRouter initializes Gin engine and registers routes for the organization domain.
func SetupRouter(
	orgHandler *orghandler.OrgHandler,
	memberHandler *orghandler.MemberHandler,
	invitationHandler *orghandler.InvitationHandler,
) *gin.Engine {
	r := gin.Default()

	v1 := r.Group("/v1")
	RegisterOrgRoutes(v1, orgHandler, memberHandler, invitationHandler)

	return r
}

// RegisterOrgRoutes registers organization domain routes on an existing Gin router group.
func RegisterOrgRoutes(
	v1 *gin.RouterGroup,
	orgHandler *orghandler.OrgHandler,
	memberHandler *orghandler.MemberHandler,
	invitationHandler *orghandler.InvitationHandler,
) {
	// Authenticated routes
	authenticated := v1.Group("", middleware.AuthRequired())
	{
		// Organizations
		authenticated.POST("/organizations", orgHandler.CreateOrg)
		authenticated.GET("/organizations/:id", orgHandler.GetOrg)
		authenticated.PUT("/organizations/:id", middleware.RequireRole("org_admin"), orgHandler.UpdateOrg)

		// Members
		authenticated.GET("/organizations/:id/members", middleware.RequireRole("org_admin", "finance"), memberHandler.ListMembers)
		authenticated.DELETE("/organizations/:id/members/:uid", middleware.RequireRole("org_admin"), memberHandler.RemoveMember)
		authenticated.PUT("/organizations/:id/members/:uid/role", middleware.RequireRole("org_admin"), memberHandler.UpdateRole)

		// Invitations
		authenticated.POST("/organizations/:id/invitations", middleware.RequireRole("org_admin"), invitationHandler.CreateInvitation)
	}

	// Public routes
	v1.POST("/invitations/:token/accept", invitationHandler.AcceptInvitation)
}
