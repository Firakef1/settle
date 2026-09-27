package handler

import (
	"errors"
	"net/http"

	"github.com/Firakef1/settle/backend/internal/organaization/repository"
	"github.com/Firakef1/settle/backend/internal/organaization/service"
	"github.com/Firakef1/settle/backend/internal/shared/middleware"
	"github.com/gin-gonic/gin"
)

// DashboardHandler handles dashboard metrics, counts, and overview endpoints.
type DashboardHandler struct {
	dashboardService service.DashboardService
}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(dashboardService service.DashboardService) *DashboardHandler {
	return &DashboardHandler{
		dashboardService: dashboardService,
	}
}

// GetOrgStats godoc
// @Summary      Get organization metrics & statistics
// @Description  Returns aggregated organization statistics (member counts, request totals, spending, plan usage). Accessible by org_admin and finance roles.
// @Tags         dashboard
// @Produce      json
// @Param        id path string true "Organization ID"
// @Success      200 {object} map[string]dto.OrgStatsResponse
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /v1/organizations/{id}/stats [get]
func (h *DashboardHandler) GetOrgStats(c *gin.Context) {
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

	stats, err := h.dashboardService.GetOrgStats(c.Request.Context(), orgID, userID)
	if err != nil {
		if errors.Is(err, service.ErrNotOrgMember) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, repository.ErrOrgNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

// GetDashboardSummary godoc
// @Summary      Get dashboard summary overview
// @Description  Returns high-level organization overview: pending requests count, urgency breakdown, aging breakdown, and escalated items. Accessible by org_admin and finance roles.
// @Tags         dashboard
// @Produce      json
// @Param        org_id query string false "Organization ID (optional if present in JWT token)"
// @Success      200 {object} map[string]dto.DashboardSummaryResponse
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /v1/dashboard/summary [get]
func (h *DashboardHandler) GetDashboardSummary(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	orgID := middleware.GetOrgID(c)
	if orgID == "" {
		orgID = c.Query("org_id")
	}
	if orgID == "" {
		orgID = c.GetHeader("X-Organization-Id")
	}
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "organization context is required (in token or org_id parameter)"})
		return
	}

	summary, err := h.dashboardService.GetSummary(c.Request.Context(), orgID, userID)
	if err != nil {
		if errors.Is(err, service.ErrNotOrgMember) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": summary})
}
