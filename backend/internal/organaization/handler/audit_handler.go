package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Firakef1/settle/backend/internal/organaization/dto"
	"github.com/Firakef1/settle/backend/internal/organaization/repository"
	"github.com/Firakef1/settle/backend/internal/organaization/service"
	"github.com/Firakef1/settle/backend/internal/shared/middleware"
	"github.com/gin-gonic/gin"
)

// AuditHandler handles audit log queries and paginated requests.
type AuditHandler struct {
	auditService service.AuditService
}

// NewAuditHandler creates a new AuditHandler.
func NewAuditHandler(auditService service.AuditService) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
	}
}

// GetAuditLogs godoc
// @Summary      Query immutable audit logs
// @Description  Queries and paginates immutable audit logs for an organization. Accessible by org_admin and finance roles.
// @Tags         audit
// @Produce      json
// @Param        id path string true "Organization ID"
// @Param        action query string false "Filter by action type"
// @Param        actor_id query string false "Filter by actor user ID"
// @Param        start_date query string false "Filter by start timestamp (RFC3339)"
// @Param        end_date query string false "Filter by end timestamp (RFC3339)"
// @Param        page query int false "Page number (defaults to 1)"
// @Param        limit query int false "Items per page (defaults to 20, max 100)"
// @Success      200 {object} dto.AuditLogListResponse
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /v1/organizations/{id}/audit-log [get]
func (h *AuditHandler) GetAuditLogs(c *gin.Context) {
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

	var filters dto.AuditFilters
	filters.Action = c.Query("action")
	filters.ActorID = c.Query("actor_id")

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			filters.Page = p
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			filters.Limit = l
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			filters.Offset = o
		}
	}

	if startStr := c.Query("start_date"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			filters.StartDate = &t
		}
	}
	if endStr := c.Query("end_date"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			filters.EndDate = &t
		}
	}

	res, err := h.auditService.GetAuditLogs(c.Request.Context(), orgID, userID, filters)
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

	c.JSON(http.StatusOK, gin.H{
		"data":       res.Data,
		"pagination": res.Pagination,
	})
}
