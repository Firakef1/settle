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

type InvitationHandler struct {
	invitationService service.InvitationService
}

func NewInvitationHandler(invitationService service.InvitationService) *InvitationHandler {
	return &InvitationHandler{
		invitationService: invitationService,
	}
}

// CreateInvitation godoc
// @Summary      Create organization invitation
// @Description  Generates a cryptographically secure token (via crypto/rand), stores it with an expiry (7 days TTL), and assigns a role (staff or finance).
// @Tags         invitations
// @Accept       json
// @Produce      json
// @Param        id path string true "Organization ID"
// @Param        body body dto.InviteRequest true "Invitation creation payload"
// @Success      201 {object} map[string]dto.InviteResponse
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /v1/organizations/{id}/invitations [post]
func (h *InvitationHandler) CreateInvitation(c *gin.Context) {
	actorID := middleware.GetUserID(c)
	if actorID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	orgID := c.Param("id")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "organization id is required"})
		return
	}

	var req dto.InviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	invite, err := h.invitationService.CreateInvitation(c.Request.Context(), orgID, actorID, req)
	if err != nil {
		if errors.Is(err, validator.ErrInvalidEmail) || errors.Is(err, validator.ErrInvalidRole) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": invite})
}

// AcceptInvitation godoc
// @Summary      Accept organization invitation
// @Description  Public endpoint. Validates token existence and expiry. Either registers a new user or links an existing account, then inserts them into org_members with the pre-assigned role within a single transaction.
// @Tags         invitations
// @Accept       json
// @Produce      json
// @Param        token path string true "Invitation Token"
// @Param        body body dto.AcceptInviteRequest false "Accept invitation payload"
// @Success      201 {object} map[string]dto.MemberResponse
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /v1/invitations/{token}/accept [post]
func (h *InvitationHandler) AcceptInvitation(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invitation token is required"})
		return
	}

	var req dto.AcceptInviteRequest
	_ = c.ShouldBindJSON(&req)

	member, err := h.invitationService.AcceptInvitation(c.Request.Context(), token, req)
	if err != nil {
		if errors.Is(err, repository.ErrInvitationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "invitation not found"})
			return
		}
		if errors.Is(err, service.ErrInvitationExpired) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invitation token has expired"})
			return
		}
		if errors.Is(err, service.ErrInvitationUsed) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invitation token has already been used"})
			return
		}
		if errors.Is(err, service.ErrAlreadyMember) {
			c.JSON(http.StatusConflict, gin.H{"error": "user is already a member of this organization"})
			return
		}
		if errors.Is(err, validator.ErrPasswordTooShort) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": member})
}
