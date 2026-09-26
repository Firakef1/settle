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

type OrgHandler struct {
	orgService service.OrgService
}

func NewOrgHandler(orgService service.OrgService) *OrgHandler {
	return &OrgHandler{
		orgService: orgService,
	}
}

// CreateOrg godoc
// @Summary      Create an organization
// @Description  Creates a new organization row with an auto-generated slug, inserts the creator into org_members with role org_admin, and writes an audit log in a single transaction.
// @Tags         organizations
// @Accept       json
// @Produce      json
// @Param        body body dto.CreateOrgRequest true "Organization creation payload"
// @Success      201 {object} map[string]dto.OrgResponse
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /v1/organizations [post]
func (h *OrgHandler) CreateOrg(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req dto.CreateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	org, err := h.orgService.CreateOrg(c.Request.Context(), userID, req)
	if err != nil {
		if errors.Is(err, validator.ErrEmptyOrgName) ||
			errors.Is(err, validator.ErrOrgNameTooLong) ||
			errors.Is(err, validator.ErrInvalidCurrency) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": org})
}

// GetOrg godoc
// @Summary      Get organization profile
// @Description  Returns organization profile. Scoped to members of that organization only.
// @Tags         organizations
// @Produce      json
// @Param        id path string true "Organization ID"
// @Success      200 {object} map[string]dto.OrgResponse
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /v1/organizations/{id} [get]
func (h *OrgHandler) GetOrg(c *gin.Context) {
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

	org, err := h.orgService.GetOrg(c.Request.Context(), orgID, userID)
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

	c.JSON(http.StatusOK, gin.H{"data": org})
}

// UpdateOrg godoc
// @Summary      Update organization
// @Description  Updates organization name and/or currency. Allowed for org_admin only. Currency must be USD, EUR, or GBP.
// @Tags         organizations
// @Accept       json
// @Produce      json
// @Param        id path string true "Organization ID"
// @Param        body body dto.UpdateOrgRequest true "Organization update payload"
// @Success      200 {object} map[string]dto.OrgResponse
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /v1/organizations/{id} [put]
func (h *OrgHandler) UpdateOrg(c *gin.Context) {
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

	var req dto.UpdateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	org, err := h.orgService.UpdateOrg(c.Request.Context(), orgID, userID, req)
	if err != nil {
		if errors.Is(err, validator.ErrInvalidCurrency) ||
			errors.Is(err, validator.ErrEmptyOrgName) ||
			errors.Is(err, validator.ErrOrgNameTooLong) {
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

	c.JSON(http.StatusOK, gin.H{"data": org})
}
