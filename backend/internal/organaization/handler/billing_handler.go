package handler

import (
	"errors"
	"net/http"

	"github.com/Firakef1/settle/backend/internal/organaization/dto"
	"github.com/Firakef1/settle/backend/internal/organaization/repository"
	"github.com/Firakef1/settle/backend/internal/organaization/service"
	"github.com/Firakef1/settle/backend/internal/organaization/validator"
	"github.com/Firakef1/settle/backend/internal/shared/middleware"
	"github.com/gin-gonic/gin"
)

// BillingHandler handles subscription plans and billing management endpoints.
type BillingHandler struct {
	billingService service.BillingService
}

// NewBillingHandler creates a new BillingHandler.
func NewBillingHandler(billingService service.BillingService) *BillingHandler {
	return &BillingHandler{
		billingService: billingService,
	}
}

// GetPlans godoc
// @Summary      Get available subscription plans
// @Description  Returns all available subscription plans with pricing, feature sets, and request/user limits. Public endpoint.
// @Tags         billing
// @Produce      json
// @Success      200 {object} map[string][]dto.Plan
// @Router       /v1/billing/plans [get]
func (h *BillingHandler) GetPlans(c *gin.Context) {
	plans := h.billingService.GetPlans()
	c.JSON(http.StatusOK, gin.H{"data": plans})
}

// UpdatePlan godoc
// @Summary      Update organization subscription plan
// @Description  Updates an organization's subscription plan (free, starter, pro). Allowed for org_admin only.
// @Tags         billing
// @Accept       json
// @Produce      json
// @Param        id path string true "Organization ID"
// @Param        body body dto.UpdatePlanRequest true "Plan update payload"
// @Success      200 {object} map[string]dto.UpdatePlanResponse
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /v1/organizations/{id}/plan [put]
func (h *BillingHandler) UpdatePlan(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	orgID := c.Param("id")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "organization id is required"})
		return
	}

	var req dto.UpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	resp, err := h.billingService.UpdatePlan(c.Request.Context(), orgID, userID, req.Plan)
	if err != nil {
		if errors.Is(err, validator.ErrInvalidPlan) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, repository.ErrOrgNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}
