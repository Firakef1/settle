package router

import (
	"github.com/gin-gonic/gin"

	requestsHandler "github.com/Firakef1/settle/backend/internal/requests/handler"
	"github.com/Firakef1/settle/backend/internal/shared/middleware"
)

// RegisterRequestRoutes attaches requests endpoints to the given router group.
func RegisterRequestRoutes(rg *gin.RouterGroup, reqH *requestsHandler.RequestHandler) {
	if reqH == nil {
		return
	}

	reqs := rg.Group("/requests")
	reqs.Use(middleware.AuthRequired())
	{
		reqs.POST("", reqH.Create)
		reqs.GET("", reqH.List)
		reqs.GET("/:id", reqH.GetDetail)

		reqs.GET("/:id/previous", reqH.GetPrevious)

		reqs.POST("/:id/submit", reqH.Submit)
		reqs.PUT("/:id/withdraw", reqH.Withdraw)
		reqs.POST("/:id/resubmit", reqH.Resubmit)

		reqs.POST("/:id/receipts", reqH.UploadReceipt)

		reqs.GET("/:id/comments", reqH.GetComments)
		reqs.POST("/:id/comments", reqH.AddComment)
	}

	receipts := rg.Group("/receipts")
	receipts.Use(middleware.AuthRequired())
	{
		receipts.GET("/:id", reqH.GetReceipt)
		receipts.POST("/upload", reqH.UploadReceipt) // User requested POST /receipts/upload
	}
}
