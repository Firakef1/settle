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

type MemberHandler struct {
	memberService service.MemberService
}

func NewMemberHandler(memberService service.MemberService) *MemberHandler {
	return &MemberHandler{
		memberService: memberService,
	}
}

// ListMembers godoc
// @Summary      List organization members
// @Description  Returns all members of an organization. Accessible by org_admin and finance roles.
// @Tags         members
// @Produce      json
// @Param        id path string true "Organization ID"
// @Success      200 {object} map[string][]dto.MemberResponse
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /v1/organizations/{id}/members [get]
func (h *MemberHandler) ListMembers(c *gin.Context) {
	orgID := c.Param("id")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "organization id is required"})
		return
	}

	members, err := h.memberService.ListMembers(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if members == nil {
		members = []*dto.MemberResponse{}
	}

	c.JSON(http.StatusOK, gin.H{"data": members})
}

// RemoveMember godoc
// @Summary      Remove organization member
// @Description  Removes a member from an organization. Returns 403 Forbidden if the member is the last org_admin.
// @Tags         members
// @Produce      json
// @Param        id path string true "Organization ID"
// @Param        uid path string true "User ID to remove"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /v1/organizations/{id}/members/{uid} [delete]
func (h *MemberHandler) RemoveMember(c *gin.Context) {
	actorID := middleware.GetUserID(c)
	if actorID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	orgID := c.Param("id")
	uid := c.Param("uid")
	if orgID == "" || uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "organization id and user id are required"})
		return
	}

	err := h.memberService.RemoveMember(c.Request.Context(), orgID, uid, actorID)
	if err != nil {
		if errors.Is(err, service.ErrCannotRemoveLastAdmin) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, repository.ErrMemberNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "member not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "member removed successfully"})
}

// UpdateRole godoc
// @Summary      Update member role
// @Description  Switches role between staff and finance. Cannot demote the member if they are the last org_admin.
// @Tags         members
// @Accept       json
// @Produce      json
// @Param        id path string true "Organization ID"
// @Param        uid path string true "User ID whose role is being changed"
// @Param        body body dto.UpdateRoleRequest true "New role payload"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /v1/organizations/{id}/members/{uid}/role [put]
func (h *MemberHandler) UpdateRole(c *gin.Context) {
	actorID := middleware.GetUserID(c)
	if actorID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	orgID := c.Param("id")
	uid := c.Param("uid")
	if orgID == "" || uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "organization id and user id are required"})
		return
	}

	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	err := h.memberService.UpdateRole(c.Request.Context(), orgID, uid, req.Role, actorID)
	if err != nil {
		if errors.Is(err, validator.ErrInvalidRole) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrCannotDemoteLastAdmin) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, repository.ErrMemberNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "member not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "role updated successfully"})
}
