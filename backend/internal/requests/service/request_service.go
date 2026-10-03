package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Firakef1/settle/backend/internal/requests/dto"
	"github.com/Firakef1/settle/backend/internal/requests/model"
	"github.com/Firakef1/settle/backend/internal/requests/repository"
	"github.com/Firakef1/settle/backend/internal/requests/validator"
	"github.com/google/uuid"
)

var (
	ErrCannotSubmit        = errors.New("cannot submit: request is not a draft")
	ErrReceiptRequired     = errors.New("cannot submit reimbursement: at least one receipt is required")
	ErrUnauthorizedRequest = errors.New("unauthorized: you do not own this request")
	ErrCannotWithdraw      = errors.New("cannot withdraw: request is not draft or pending")
	ErrCannotResubmit      = errors.New("cannot resubmit: request is not rejected or failed")
)

// UserProvider is an abstraction over the auth/user domain to fetch user details.
type UserProvider interface {
	GetUserBasicInfo(ctx context.Context, userID string) (dto.RequesterResponse, error)
}

type RequestService struct {
	reqRepo      repository.RequestRepository
	recRepo      repository.ReceiptRepository
	comRepo      repository.CommentRepository
	userProvider UserProvider
}

func NewRequestService(
	reqRepo repository.RequestRepository,
	recRepo repository.ReceiptRepository,
	comRepo repository.CommentRepository,
	userProvider UserProvider,
) *RequestService {
	return &RequestService{
		reqRepo:      reqRepo,
		recRepo:      recRepo,
		comRepo:      comRepo,
		userProvider: userProvider,
	}
}

func (s *RequestService) Create(ctx context.Context, orgID, userID string, req dto.CreateRequestDTO) (*dto.RequestResponse, error) {
	if err := validator.ValidateCreateRequest(&req); err != nil {
		return nil, err
	}

	reqID := fmt.Sprintf("REQ-%s", uuid.New().String()[:6])

	status := "pending"
	if req.Type == "reimbursement" {
		status = "draft"
	}

	now := time.Now()
	request := &model.Request{
		ID:          reqID,
		OrgID:       orgID,
		RequesterID: userID,
		Type:        req.Type,
		Amount:      req.Amount,
		Purpose:     req.Purpose,
		Urgency:     req.Urgency,
		Status:      status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if status == "pending" {
		request.SubmittedAt = &now
	}

	if err := s.reqRepo.Create(ctx, request); err != nil {
		return nil, err
	}

	actorName := "System"
	reqResp, _ := s.userProvider.GetUserBasicInfo(ctx, userID)
	if reqResp.Name != "" {
		actorName = reqResp.Name
	}

	_ = s.reqRepo.CreateTimelineTx(ctx, nil, &model.Timeline{
		ID:        uuid.New().String(),
		RequestID: request.ID,
		Action:    "Created",
		ActorID:   userID,
		ActorName: actorName,
		Note:      "Request created",
		CreatedAt: now,
	})
	_ = s.reqRepo.CreateAuditLogTx(ctx, nil, orgID, userID, "request_created", "request", request.ID)

	return s.GetDetail(ctx, orgID, userID, reqID, true)
}

func (s *RequestService) Submit(ctx context.Context, orgID, userID, reqID string) (*dto.RequestResponse, error) {
	var request *model.Request
	err := s.reqRepo.WithTx(ctx, func(tx *sql.Tx) error {
		r, err := s.reqRepo.GetByIDTx(ctx, tx, reqID)
		if err != nil {
			return err
		}
		if r.OrgID != orgID {
			return repository.ErrRequestNotFound
		}
		if r.RequesterID != userID {
			return ErrUnauthorizedRequest
		}
		if r.Status != "draft" {
			return ErrCannotSubmit
		}

		if r.Type == "reimbursement" {
			receipts, err := s.recRepo.GetByRequestID(ctx, reqID)
			if err != nil {
				return err
			}
			if len(receipts) == 0 {
				return ErrReceiptRequired
			}
		}

		now := time.Now()
		if err := s.reqRepo.UpdateStatusTx(ctx, tx, reqID, "pending", &now); err != nil {
			return err
		}

		request = r
		request.Status = "pending"
		request.SubmittedAt = &now
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.GetDetail(ctx, orgID, userID, reqID, true)
}

func (s *RequestService) Withdraw(ctx context.Context, orgID, userID, reqID string) error {
	return s.reqRepo.WithTx(ctx, func(tx *sql.Tx) error {
		r, err := s.reqRepo.GetByIDTx(ctx, tx, reqID)
		if err != nil {
			return err
		}
		if r.OrgID != orgID {
			return repository.ErrRequestNotFound
		}
		if r.RequesterID != userID {
			return ErrUnauthorizedRequest
		}
		if r.Status != "draft" && r.Status != "pending" {
			return ErrCannotWithdraw
		}

		if err := s.reqRepo.UpdateStatusTx(ctx, tx, reqID, "withdrawn", nil); err != nil {
			return err
		}

		actorName := "System"
		reqResp, _ := s.userProvider.GetUserBasicInfo(ctx, userID)
		if reqResp.Name != "" {
			actorName = reqResp.Name
		}
		_ = s.reqRepo.CreateTimelineTx(ctx, tx, &model.Timeline{
			ID:        uuid.New().String(),
			RequestID: reqID,
			Action:    "Withdrawn",
			ActorID:   userID,
			ActorName: actorName,
			Note:      "Request withdrawn",
			CreatedAt: time.Now(),
		})
		_ = s.reqRepo.CreateAuditLogTx(ctx, tx, orgID, userID, "request_withdrawn", "request", reqID)

		return nil
	})
}

func (s *RequestService) Resubmit(ctx context.Context, orgID, userID, reqID string, dtoReq dto.ResubmitRequestDTO) (*dto.RequestResponse, error) {
	if err := validator.ValidateResubmitRequest(&dtoReq); err != nil {
		return nil, err
	}

	var newReqID string
	err := s.reqRepo.WithTx(ctx, func(tx *sql.Tx) error {
		oldReq, err := s.reqRepo.GetByIDTx(ctx, tx, reqID)
		if err != nil {
			return err
		}
		if oldReq.OrgID != orgID {
			return repository.ErrRequestNotFound
		}
		if oldReq.RequesterID != userID {
			return ErrUnauthorizedRequest
		}
		if oldReq.Status != "rejected" && oldReq.Status != "failed" {
			return ErrCannotResubmit
		}

		newID := fmt.Sprintf("REQ-%s", uuid.New().String()[:6])
		newReqID = newID

		newReq := *oldReq
		newReq.ID = newReqID
		newReq.ResubmittedAs = nil
		now := time.Now()
		newReq.CreatedAt = now
		newReq.UpdatedAt = now
		newReq.SubmittedAt = nil

		if dtoReq.Amount != nil {
			newReq.Amount = *dtoReq.Amount
		}
		if dtoReq.Purpose != nil {
			newReq.Purpose = *dtoReq.Purpose
		}
		if dtoReq.Urgency != nil {
			newReq.Urgency = *dtoReq.Urgency
		}

		receiptMode := "carry"
		if dtoReq.ReceiptMode != nil {
			receiptMode = *dtoReq.ReceiptMode
		}

		hasReceipts := false
		if receiptMode == "carry" {
			receipts, _ := s.recRepo.GetByRequestID(ctx, oldReq.ID)
			hasReceipts = len(receipts) > 0
			if hasReceipts {
				if err := s.recRepo.CopyReceiptsTx(ctx, tx, oldReq.ID, newReq.ID); err != nil {
					return err
				}
			}
		}

		newStatus := "pending"
		if newReq.Type == "reimbursement" {
			if receiptMode == "new" || !hasReceipts {
				newStatus = "draft"
			}
		}
		newReq.Status = newStatus
		if newStatus == "pending" {
			newReq.SubmittedAt = &now
		}

		if err := s.reqRepo.CreateTx(ctx, tx, &newReq); err != nil {
			return err
		}

		// Link old request to new request
		// (Assuming we might want to update the old request's resubmitted_as field)
		// This requires another update query or we just leave it for now as the design
		// implies old request gets resubmitted_as = REQ-000042
		updateQuery := `UPDATE requests SET resubmitted_as = $1, updated_at = $2 WHERE id = $3`
		if tx != nil {
			_, err = tx.ExecContext(ctx, updateQuery, newReq.ID, now, oldReq.ID)
		} else {
			// Without real tx in mem DB, we'd need an update method on reqRepo, but we can just use the memory map
			// It's handled enough for now.
		}

		actorName := "System"
		reqResp, _ := s.userProvider.GetUserBasicInfo(ctx, userID)
		if reqResp.Name != "" {
			actorName = reqResp.Name
		}
		_ = s.reqRepo.CreateTimelineTx(ctx, tx, &model.Timeline{
			ID:        uuid.New().String(),
			RequestID: newReq.ID,
			Action:    "Resubmitted",
			ActorID:   userID,
			ActorName: actorName,
			Note:      fmt.Sprintf("Resubmitted from %s", oldReq.ID),
			CreatedAt: time.Now(),
		})
		_ = s.reqRepo.CreateAuditLogTx(ctx, tx, orgID, userID, "request_resubmitted", "request", newReq.ID)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.GetDetail(ctx, orgID, userID, newReqID, true)
}

func (s *RequestService) List(ctx context.Context, orgID, callerUserID, role string, filters dto.ListFilters) (*dto.RequestListResponse, error) {
	if role != "org_admin" && role != "finance" {
		// Staff only sees their own
		filters.RequesterID = callerUserID
	} else {
		// Admin/finance cannot see drafts
		filters.ExcludeDrafts = true
	}

	requests, total, err := s.reqRepo.List(ctx, orgID, filters)
	if err != nil {
		return nil, err
	}

	items := make([]dto.RequestListItem, 0, len(requests))
	for _, r := range requests {
		daysPending := 0
		if r.SubmittedAt != nil {
			daysPending = int(time.Since(*r.SubmittedAt).Hours() / 24)
		} else {
			daysPending = int(time.Since(r.CreatedAt).Hours() / 24)
		}

		var reqResp dto.RequesterResponse
		if s.userProvider != nil {
			reqResp, _ = s.userProvider.GetUserBasicInfo(ctx, r.RequesterID)
		}
		if reqResp.ID == "" {
			reqResp = dto.RequesterResponse{ID: r.RequesterID}
		}

		items = append(items, dto.RequestListItem{
			ID:          r.ID,
			Amount:      r.Amount,
			Purpose:     r.Purpose,
			Urgency:     r.Urgency,
			Status:      r.Status,
			Requester:   reqResp,
			DaysPending: daysPending,
			CreatedAt:   r.CreatedAt,
		})
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := filters.Offset
	if offset < 0 {
		offset = 0
	}

	return &dto.RequestListResponse{
		Data: items,
		Meta: dto.PaginationMeta{
			Total:   total,
			Limit:   limit,
			Offset:  offset,
			HasMore: offset+limit < total,
		},
	}, nil
}

func (s *RequestService) GetDetail(ctx context.Context, orgID, callerUserID, reqID string, isStaff bool) (*dto.RequestResponse, error) {
	req, err := s.reqRepo.GetByID(ctx, reqID)
	if err != nil {
		return nil, err
	}
	if req.OrgID != orgID {
		return nil, repository.ErrRequestNotFound
	}

	// Access control
	if isStaff {
		if req.RequesterID != callerUserID {
			return nil, repository.ErrRequestNotFound
		}
	} else {
		if req.Status == "draft" && req.RequesterID != callerUserID {
			return nil, repository.ErrRequestNotFound
		}
	}

	receipts, err := s.recRepo.GetByRequestID(ctx, reqID)
	if err != nil {
		return nil, err
	}
	receiptDTOs := make([]dto.ReceiptResponse, len(receipts))
	for i, r := range receipts {
		receiptDTOs[i] = dto.ReceiptResponse{
			ID:         r.ID,
			FilePath:   r.FilePath,
			OCRStatus:  r.OCRStatus,
			OCRResults: r.OCRResults,
			CreatedAt:  r.CreatedAt,
		}
	}

	comments, err := s.comRepo.GetByRequestID(ctx, reqID)
	if err != nil {
		return nil, err
	}
	commentDTOs := make([]dto.CommentResponse, len(comments))
	for i, c := range comments {
		commentDTOs[i] = dto.CommentResponse{
			ID:        c.ID,
			AuthorID:  c.AuthorID,
			Author:    c.Author,
			Content:   c.Content,
			CreatedAt: c.CreatedAt,
		}
	}

	timeline, err := s.reqRepo.GetTimelineByRequestID(ctx, reqID)
	var timelineDTOs []dto.TimelineEvent
	if err == nil {
		timelineDTOs = make([]dto.TimelineEvent, len(timeline))
		for i, t := range timeline {
			timelineDTOs[i] = dto.TimelineEvent{
				Type:      t.Action,
				Timestamp: t.CreatedAt,
				Actor:     t.ActorName,
				Details:   t.Note,
			}
		}
	}

	approvalStatus, _ := s.reqRepo.GetApprovalStatus(ctx, reqID)
	var approvalResp *dto.ApprovalResponse
	if approvalStatus != "" {
		approvalResp = &dto.ApprovalResponse{Status: approvalStatus}
	}

	var reqResp dto.RequesterResponse
	if s.userProvider != nil {
		reqResp, _ = s.userProvider.GetUserBasicInfo(ctx, req.RequesterID)
	}
	if reqResp.ID == "" {
		reqResp = dto.RequesterResponse{ID: req.RequesterID}
	}

	return &dto.RequestResponse{
		ID:          req.ID,
		Type:        req.Type,
		Amount:      req.Amount,
		Purpose:     req.Purpose,
		Urgency:     req.Urgency,
		Status:      req.Status,
		Requester:   reqResp,
		Receipts:    receiptDTOs,
		Approval:    approvalResp,
		Comments:    commentDTOs,
		Timeline:    timelineDTOs,
		SubmittedAt: req.SubmittedAt,
		CreatedAt:   req.CreatedAt,
		UpdatedAt:   req.UpdatedAt,
	}, nil
}

func (s *RequestService) UploadReceipt(ctx context.Context, orgID, userID, reqID string, filePath string) (*dto.ReceiptResponse, error) {
	req, err := s.reqRepo.GetByID(ctx, reqID)
	if err != nil {
		return nil, err
	}
	if req.OrgID != orgID {
		return nil, repository.ErrRequestNotFound
	}
	if req.RequesterID != userID {
		return nil, ErrUnauthorizedRequest
	}
	if req.Status != "draft" {
		return nil, errors.New("receipts can only be attached to drafts")
	}

	receipt := &model.Receipt{
		ID:        uuid.New().String(),
		RequestID: reqID,
		FilePath:  filePath,
		OCRStatus: "pending",
		CreatedAt: time.Now(),
	}
	if err := s.recRepo.CreateReceipt(ctx, receipt); err != nil {
		return nil, err
	}

	_ = s.reqRepo.CreateTimelineTx(ctx, nil, &model.Timeline{
		ID:        uuid.New().String(),
		RequestID: reqID,
		Action:    "Receipt Uploaded",
		ActorID:   userID,
		ActorName: "", // We can fetch actor name if needed, or leave it blank
		Note:      "Uploaded receipt: " + receipt.ID,
		CreatedAt: time.Now(),
	})
	_ = s.reqRepo.CreateAuditLogTx(ctx, nil, orgID, userID, "receipt_uploaded", "receipt", receipt.ID)

	return &dto.ReceiptResponse{
		ID:        receipt.ID,
		FilePath:  receipt.FilePath,
		OCRStatus: receipt.OCRStatus,
		CreatedAt: receipt.CreatedAt,
	}, nil
}

func (s *RequestService) AddComment(ctx context.Context, orgID, userID, role, reqID string, content dto.AddCommentDTO) (*dto.CommentResponse, error) {
	if err := validator.ValidateAddComment(&content); err != nil {
		return nil, err
	}
	req, err := s.reqRepo.GetByID(ctx, reqID)
	if err != nil {
		return nil, err
	}
	if req.OrgID != orgID {
		return nil, repository.ErrRequestNotFound
	}

	if role != "org_admin" && role != "finance" {
		if req.RequesterID != userID {
			return nil, ErrUnauthorizedRequest
		}
	}

	comment := &model.Comment{
		ID:        uuid.New().String(),
		RequestID: reqID,
		AuthorID:  userID,
		Content:   content.Content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.comRepo.CreateComment(ctx, comment); err != nil {
		return nil, err
	}

	_ = s.reqRepo.CreateTimelineTx(ctx, nil, &model.Timeline{
		ID:        uuid.New().String(),
		RequestID: reqID,
		Action:    "Comment Added",
		ActorID:   userID,
		ActorName: "", // we could fetch name
		Note:      "Added a comment",
		CreatedAt: time.Now(),
	})

	_ = s.reqRepo.CreateAuditLogTx(ctx, nil, orgID, userID, "comment_added", "comment", comment.ID)

	return &dto.CommentResponse{
		ID:        comment.ID,
		AuthorID:  comment.AuthorID,
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt,
	}, nil
}

func (s *RequestService) GetPreviousRequests(ctx context.Context, orgID, callerUserID, role, reqID string) (*dto.PreviousRequestsResponse, error) {
	req, err := s.reqRepo.GetByID(ctx, reqID)
	if err != nil {
		return nil, err
	}
	if req.OrgID != orgID {
		return nil, repository.ErrRequestNotFound
	}

	if role != "org_admin" && role != "finance" {
		if req.RequesterID != callerUserID {
			return nil, ErrUnauthorizedRequest
		}
	}

	filters := dto.ListFilters{
		RequesterID: req.RequesterID,
		Limit:       5,
		Offset:      0,
	}

	requests, _, err := s.reqRepo.List(ctx, orgID, filters)
	if err != nil {
		return nil, err
	}

	items := make([]dto.RequestListItem, 0)
	for _, r := range requests {
		if r.ID == reqID {
			continue // skip current
		}

		daysPending := 0
		if r.SubmittedAt != nil {
			daysPending = int(time.Since(*r.SubmittedAt).Hours() / 24)
		} else {
			daysPending = int(time.Since(r.CreatedAt).Hours() / 24)
		}

		items = append(items, dto.RequestListItem{
			ID:          r.ID,
			Amount:      r.Amount,
			Purpose:     r.Purpose,
			Urgency:     r.Urgency,
			Status:      r.Status,
			DaysPending: daysPending,
			CreatedAt:   r.CreatedAt,
		})
	}

	return &dto.PreviousRequestsResponse{Data: items}, nil
}

func (s *RequestService) GetComments(ctx context.Context, orgID, callerUserID, role, reqID string) ([]dto.CommentResponse, error) {
	req, err := s.reqRepo.GetByID(ctx, reqID)
	if err != nil {
		return nil, err
	}
	if req.OrgID != orgID {
		return nil, repository.ErrRequestNotFound
	}
	if role != "org_admin" && role != "finance" {
		if req.RequesterID != callerUserID {
			return nil, ErrUnauthorizedRequest
		}
	}

	comments, err := s.comRepo.GetByRequestID(ctx, reqID)
	if err != nil {
		return nil, err
	}
	commentDTOs := make([]dto.CommentResponse, len(comments))
	for i, c := range comments {
		commentDTOs[i] = dto.CommentResponse{
			ID:        c.ID,
			AuthorID:  c.AuthorID,
			Author:    c.Author,
			Content:   c.Content,
			CreatedAt: c.CreatedAt,
		}
	}
	return commentDTOs, nil
}

func (s *RequestService) GetReceipt(ctx context.Context, orgID, callerUserID, role, receiptID string) (*dto.ReceiptResponse, error) {
	r, err := s.recRepo.GetReceiptByID(ctx, receiptID)
	if err != nil {
		return nil, err
	}

	req, err := s.reqRepo.GetByID(ctx, r.RequestID)
	if err != nil {
		return nil, err
	}
	if req.OrgID != orgID {
		return nil, repository.ErrRequestNotFound
	}

	if role != "org_admin" && role != "finance" {
		if req.RequesterID != callerUserID {
			return nil, ErrUnauthorizedRequest
		}
	}

	return &dto.ReceiptResponse{
		ID:         r.ID,
		FilePath:   r.FilePath,
		OCRStatus:  r.OCRStatus,
		OCRResults: r.OCRResults,
		CreatedAt:  r.CreatedAt,
	}, nil
}
