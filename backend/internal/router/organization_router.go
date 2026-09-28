// Package router registers HTTP routes.
package router

import (
	orghandler "github.com/Firakef1/settle/backend/internal/organaization/handler"
	"github.com/Firakef1/settle/backend/internal/shared/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterOrgRoutes registers organization domain routes on an existing Gin router group.
func RegisterOrgRoutes(
	v1 *gin.RouterGroup,
	orgHandler *orghandler.OrgHandler,
	memberHandler *orghandler.MemberHandler,
	invitationHandler *orghandler.InvitationHandler,
	optionalHandlers ...any,
) {
	RegisterOrganizationRoutes(v1, orgHandler, memberHandler, invitationHandler, optionalHandlers...)
}

// RegisterAllOrgRoutes explicitly registers all organization domain routes including audit, stats, dashboard, and billing.
func RegisterAllOrgRoutes(
	v1 *gin.RouterGroup,
	orgHandler *orghandler.OrgHandler,
	memberHandler *orghandler.MemberHandler,
	invitationHandler *orghandler.InvitationHandler,
	auditHandler *orghandler.AuditHandler,
	dashboardHandler *orghandler.DashboardHandler,
	billingHandler *orghandler.BillingHandler,
) {
	RegisterOrganizationRoutes(v1, orgHandler, memberHandler, invitationHandler, auditHandler, dashboardHandler, billingHandler)
}

// RegisterOrganizationRoutes registers explicit organization domain routes on an existing Gin router group.
func RegisterOrganizationRoutes(
	v1 *gin.RouterGroup,
	orgHandler *orghandler.OrgHandler,
	memberHandler *orghandler.MemberHandler,
	invitationHandler *orghandler.InvitationHandler,
	optionalHandlers ...any,
) {
	var auditHandler *orghandler.AuditHandler
	var dashboardHandler *orghandler.DashboardHandler
	var billingHandler *orghandler.BillingHandler

	for _, opt := range optionalHandlers {
		switch h := opt.(type) {
		case *orghandler.AuditHandler:
			auditHandler = h
		case *orghandler.DashboardHandler:
			dashboardHandler = h
		case *orghandler.BillingHandler:
			billingHandler = h
		}
	}

	if orgHandler == nil && memberHandler == nil && invitationHandler == nil &&
		auditHandler == nil && dashboardHandler == nil && billingHandler == nil {
		return
	}

	// Authenticated routes
	authenticated := v1.Group("", middleware.AuthRequired())
	{
		// Organizations
		if orgHandler != nil {
			authenticated.POST("/organizations", orgHandler.CreateOrg)
			authenticated.GET("/organizations/:id", orgHandler.GetOrg)
			authenticated.PUT("/organizations/:id", middleware.RequireRole("org_admin"), orgHandler.UpdateOrg)
		}

		// Members
		if memberHandler != nil {
			authenticated.GET("/organizations/:id/members", middleware.RequireRole("org_admin", "finance"), memberHandler.ListMembers)
			authenticated.DELETE("/organizations/:id/members/:uid", middleware.RequireRole("org_admin"), memberHandler.RemoveMember)
			authenticated.PUT("/organizations/:id/members/:uid/role", middleware.RequireRole("org_admin"), memberHandler.UpdateRole)
		}

		// Invitations
		if invitationHandler != nil {
			authenticated.POST("/organizations/:id/invitations", middleware.RequireRole("org_admin"), invitationHandler.CreateInvitation)
		}

		// Audit Log
		if auditHandler != nil {
			authenticated.GET("/organizations/:id/audit-log", middleware.RequireRole("org_admin", "finance"), auditHandler.GetAuditLogs)
		}

		// Dashboard & Stats
		if dashboardHandler != nil {
			authenticated.GET("/organizations/:id/stats", middleware.RequireRole("org_admin", "finance"), dashboardHandler.GetOrgStats)
			authenticated.GET("/dashboard/summary", middleware.RequireRole("org_admin", "finance"), dashboardHandler.GetDashboardSummary)
		}

		// Plan Management (Org scope)
		if billingHandler != nil {
			authenticated.PUT("/organizations/:id/plan", middleware.RequireRole("org_admin"), billingHandler.UpdatePlan)
		}
	}

	// Public routes
	if invitationHandler != nil {
		v1.POST("/invitations/:token/accept", invitationHandler.AcceptInvitation)
	}
	if billingHandler != nil {
		v1.GET("/billing/plans", billingHandler.GetPlans)
	}
}
