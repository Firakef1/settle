package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Firakef1/settle/backend/internal/requests/dto"
	"github.com/Firakef1/settle/backend/internal/requests/repository"
	"github.com/Firakef1/settle/backend/internal/requests/service"
	"github.com/Firakef1/settle/backend/internal/requests/validator"
	"github.com/Firakef1/settle/backend/internal/shared/middleware"
)

type RequestHandler struct {
	service *service.RequestService
}

func NewRequestHandler(s *service.RequestService) *RequestHandler {
	return &RequestHandler{service: s}
}

func (h *RequestHandler) getOrgIDAndRole(c *gin.Context) (string, string) {
	// Fallback to checking gin context directly if middleware functions aren't available
	orgID := c.GetString("org_id")
	role := c.GetString("role")
	return orgID, role
}

// Create godoc
// @Summary      Create request
// @Description  Create a new payout request
// @Tags         requests
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateRequestDTO true "Create Request DTO"
// @Success      201  {object}  map[string]dto.RequestResponse
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /requests [post]
func (h *RequestHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	orgID, _ := h.getOrgIDAndRole(c)

	var req dto.CreateRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	resp, err := h.service.Create(c.Request.Context(), orgID, userID, req)
	if err != nil {
		if errors.Is(err, validator.ErrInvalidAmount) ||
			errors.Is(err, validator.ErrInvalidPurpose) ||
			errors.Is(err, validator.ErrInvalidType) ||
			errors.Is(err, validator.ErrInvalidUrgency) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.Printf("[ERROR] Create request failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": resp})
}

// Submit godoc
// @Summary      Submit request
// @Description  Submit a draft payout request
// @Tags         requests
// @Produce      json
// @Param        id   path      string  true  "Request ID"
// @Success      200  {object}  map[string]dto.RequestResponse
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /requests/{id}/submit [post]
func (h *RequestHandler) Submit(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	orgID, _ := h.getOrgIDAndRole(c)
	reqID := c.Param("id")

	resp, err := h.service.Submit(c.Request.Context(), orgID, userID, reqID)
	if err != nil {
		if errors.Is(err, service.ErrCannotSubmit) || errors.Is(err, service.ErrReceiptRequired) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrUnauthorizedRequest) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, repository.ErrRequestNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
			return
		}
		log.Printf("[ERROR] Submit request failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// Withdraw godoc
// @Summary      Withdraw request
// @Description  Withdraw a pending or draft payout request
// @Tags         requests
// @Produce      json
// @Param        id   path      string  true  "Request ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /requests/{id}/withdraw [put]
func (h *RequestHandler) Withdraw(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	orgID, _ := h.getOrgIDAndRole(c)
	reqID := c.Param("id")

	err := h.service.Withdraw(c.Request.Context(), orgID, userID, reqID)
	if err != nil {
		if errors.Is(err, service.ErrCannotWithdraw) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrUnauthorizedRequest) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, repository.ErrRequestNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
			return
		}
		log.Printf("[ERROR] Withdraw request failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to withdraw request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "request withdrawn"})
}

// Resubmit godoc
// @Summary      Resubmit request
// @Description  Resubmit a rejected or failed payout request
// @Tags         requests
// @Accept       json
// @Produce      json
// @Param        id      path      string                  true  "Request ID"
// @Param        request body      dto.ResubmitRequestDTO  true  "Resubmit Request DTO"
// @Success      201     {object}  map[string]dto.RequestResponse
// @Failure      400     {object}  map[string]string
// @Failure      401     {object}  map[string]string
// @Failure      403     {object}  map[string]string
// @Failure      404     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Security     BearerAuth
// @Router       /requests/{id}/resubmit [post]
func (h *RequestHandler) Resubmit(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	orgID, _ := h.getOrgIDAndRole(c)
	reqID := c.Param("id")

	var req dto.ResubmitRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	resp, err := h.service.Resubmit(c.Request.Context(), orgID, userID, reqID, req)
	if err != nil {
		if errors.Is(err, service.ErrCannotResubmit) ||
			errors.Is(err, validator.ErrInvalidAmount) ||
			errors.Is(err, validator.ErrInvalidPurpose) ||
			errors.Is(err, validator.ErrInvalidUrgency) ||
			errors.Is(err, validator.ErrInvalidMode) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrUnauthorizedRequest) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, repository.ErrRequestNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
			return
		}
		log.Printf("[ERROR] Resubmit request failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resubmit request"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": resp})
}

// List godoc
// @Summary      List requests
// @Description  List requests with filtering and pagination
// @Tags         requests
// @Produce      json
// @Param        status       query     string  false  "Filter by status"
// @Param        urgency      query     string  false  "Filter by urgency"
// @Param        requester_id query     string  false  "Filter by requester"
// @Param        sort_by      query     string  false  "Sort by field (e.g., created_at)"
// @Param        sort_order   query     string  false  "Sort order (asc, desc)"
// @Param        limit        query     int     false  "Limit"
// @Param        offset       query     int     false  "Offset"
// @Success      200          {object}  dto.RequestListResponse
// @Failure      401          {object}  map[string]string
// @Failure      500          {object}  map[string]string
// @Security     BearerAuth
// @Router       /requests [get]
func (h *RequestHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	orgID, role := h.getOrgIDAndRole(c)

	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	filters := dto.ListFilters{
		Status:      c.Query("status"),
		Urgency:     c.Query("urgency"),
		RequesterID: c.Query("requester_id"),
		SortBy:      c.Query("sort_by"),
		SortOrder:   c.Query("sort_order"),
		Limit:       limit,
		Offset:      offset,
	}

	resp, err := h.service.List(c.Request.Context(), orgID, userID, role, filters)
	if err != nil {
		log.Printf("[ERROR] List requests failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list requests"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetDetail godoc
// @Summary      Get request details
// @Description  Get full request details including receipts and comments
// @Tags         requests
// @Produce      json
// @Param        id   path      string  true  "Request ID"
// @Success      200  {object}  map[string]dto.RequestResponse
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /requests/{id} [get]
func (h *RequestHandler) GetDetail(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	orgID, role := h.getOrgIDAndRole(c)
	reqID := c.Param("id")

	isStaff := (role != "admin" && role != "finance")
	resp, err := h.service.GetDetail(c.Request.Context(), orgID, userID, reqID, isStaff)
	if err != nil {
		if errors.Is(err, repository.ErrRequestNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
			return
		}
		log.Printf("[ERROR] Get detail request failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// UploadReceipt godoc
// @Summary      Upload receipt
// @Description  Upload a receipt to a draft request
// @Tags         requests
// @Accept       multipart/form-data
// @Produce      json
// @Param        id      path      string  true  "Request ID"
// @Param        receipt formData  file    true  "Receipt file"
// @Success      201     {object}  map[string]dto.ReceiptResponse
// @Failure      400     {object}  map[string]string
// @Failure      401     {object}  map[string]string
// @Failure      403     {object}  map[string]string
// @Failure      404     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Security     BearerAuth
// @Router       /requests/{id}/receipts [post]
func (h *RequestHandler) UploadReceipt(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	orgID, _ := h.getOrgIDAndRole(c)
	reqID := c.Param("id")

	// Dummy file upload parsing for now. A real implementation would extract
	// the file from multipart form and save to cloud storage/file system.
	file, err := c.FormFile("receipt")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "receipt file is required"})
		return
	}

	filePath := fmt.Sprintf("/uploads/receipts/%s", file.Filename)

	resp, err := h.service.UploadReceipt(c.Request.Context(), orgID, userID, reqID, filePath)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorizedRequest) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, repository.ErrRequestNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
			return
		}
		if err.Error() == "receipts can only be attached to drafts" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.Printf("[ERROR] Upload receipt failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload receipt"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": resp})
}

// AddComment godoc
// @Summary      Add comment
// @Description  Add a comment to a request
// @Tags         requests
// @Accept       json
// @Produce      json
// @Param        id      path      string            true  "Request ID"
// @Param        request body      dto.AddCommentDTO true  "Add Comment DTO"
// @Success      201     {object}  map[string]dto.CommentResponse
// @Failure      400     {object}  map[string]string
// @Failure      401     {object}  map[string]string
// @Failure      404     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Security     BearerAuth
// @Router       /requests/{id}/comments [post]
func (h *RequestHandler) AddComment(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	orgID, _ := h.getOrgIDAndRole(c)
	reqID := c.Param("id")

	var req dto.AddCommentDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	resp, err := h.service.AddComment(c.Request.Context(), orgID, userID, reqID, req)
	if err != nil {
		if errors.Is(err, validator.ErrEmptyContent) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, repository.ErrRequestNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
			return
		}
		log.Printf("[ERROR] Add comment failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add comment"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": resp})
}
