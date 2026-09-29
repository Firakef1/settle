// Package approvals composes the approval domain for later router wiring.
package approvals

import (
	"database/sql"

	"github.com/Firakef1/settle/backend/internal/approvals/handler"
	"github.com/Firakef1/settle/backend/internal/approvals/repository"
	"github.com/Firakef1/settle/backend/internal/approvals/service"
)

// SetupHandler builds the approval handler with repository and service injection.
// It is intentionally not called from cmd/settle/main.go.
//
// Wire it later with:
//
//	approvalHandler := approvals.SetupHandler(database.DB)
//	router.RegisterApprovalRoutes(apiV1, approvalHandler)
func SetupHandler(db *sql.DB) *handler.ApprovalHandler {
	approvalRepo := repository.NewApprovalRepository()
	requestReader := repository.NewRequestReader()
	memberReader := repository.NewMemberReader()
	auditWriter := repository.NewAuditWriter()
	approvalSvc := service.NewApprovalService(db, approvalRepo, requestReader, memberReader, auditWriter)
	return handler.NewApprovalHandler(approvalSvc)
}
