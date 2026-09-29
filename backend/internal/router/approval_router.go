package router

import (
	approvalhandler "github.com/Firakef1/settle/backend/internal/approvals/handler"
	"github.com/Firakef1/settle/backend/internal/shared/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterApprovalRoutes mounts finance decision routes on an existing router group.
// SetupRouter does not call this yet, so the shared router file stays free of this change.
func RegisterApprovalRoutes(v1 *gin.RouterGroup, approvalHandler *approvalhandler.ApprovalHandler) {
	if v1 == nil || approvalHandler == nil {
		return
	}

	authenticated := v1.Group("", middleware.AuthRequired())
	authenticated.PUT("/requests/:id/approve", middleware.RequireRole("finance", "org_admin"), approvalHandler.Approve)
	authenticated.PUT("/requests/:id/reject", middleware.RequireRole("finance", "org_admin"), approvalHandler.Reject)
	authenticated.PUT("/requests/:id/mark-paid", middleware.RequireRole("finance", "org_admin"), approvalHandler.MarkPaid)
	authenticated.PUT("/requests/:id/payment-failed", middleware.RequireRole("finance", "org_admin"), approvalHandler.MarkFailed)
}
