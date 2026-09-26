// Package router registers HTTP routes and coordinates domain routers.
package router

import (
	orghandler "github.com/Firakef1/settle/backend/internal/organaization/handler"
	"github.com/gin-gonic/gin"
)

// SetupRouter initializes the Gin engine and registers routes for all domain routers.
func SetupRouter(
	orgHandler *orghandler.OrgHandler,
	memberHandler *orghandler.MemberHandler,
	invitationHandler *orghandler.InvitationHandler,
) *gin.Engine {
	r := gin.Default()

	v1 := r.Group("/v1")

	// Register organization routes from explicit domain router
	RegisterOrganizationRoutes(v1, orgHandler, memberHandler, invitationHandler)

	return r
}
