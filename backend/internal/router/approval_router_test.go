package router

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	approvalhandler "github.com/Firakef1/settle/backend/internal/approvals/handler"
)

func TestRegisterApprovalRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	v1 := engine.Group("/v1")
	RegisterApprovalRoutes(v1, approvalhandler.NewApprovalHandler(nil))

	wanted := map[string]bool{
		"PUT /v1/requests/:id/approve":        false,
		"PUT /v1/requests/:id/reject":         false,
		"PUT /v1/requests/:id/mark-paid":      false,
		"PUT /v1/requests/:id/payment-failed": false,
	}
	for _, route := range engine.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := wanted[key]; ok {
			wanted[key] = true
		}
	}
	for key, found := range wanted {
		require.Truef(t, found, "missing route %s", key)
	}
}
