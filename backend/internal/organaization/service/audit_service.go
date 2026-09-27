package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Firakef1/settle/backend/internal/organaization/dto"
	"github.com/Firakef1/settle/backend/internal/organaization/repository"
)

// AuditService handles querying and retrieving immutable audit logs.
type AuditService interface {
	GetAuditLogs(ctx context.Context, orgID string, callerUserID string, filters dto.AuditFilters) (*dto.AuditLogListResponse, error)
}

type auditService struct {
	db         *sql.DB
	auditRepo  repository.AuditRepository
	memberRepo repository.MemberRepository
}

// NewAuditService constructs a new AuditService instance.
func NewAuditService(
	db *sql.DB,
	auditRepo repository.AuditRepository,
	memberRepo repository.MemberRepository,
) AuditService {
	return &auditService{
		db:         db,
		auditRepo:  auditRepo,
		memberRepo: memberRepo,
	}
}

func (s *auditService) getExecutor() repository.DBTX {
	if s.db != nil {
		return s.db
	}
	return nil
}

func (s *auditService) GetAuditLogs(ctx context.Context, orgID string, callerUserID string, filters dto.AuditFilters) (*dto.AuditLogListResponse, error) {
	exec := s.getExecutor()

	// Ensure caller is member of the organization
	if callerUserID != "" && s.memberRepo != nil {
		_, err := s.memberRepo.GetMember(ctx, exec, orgID, callerUserID)
		if err != nil {
			if errors.Is(err, repository.ErrMemberNotFound) {
				return nil, ErrNotOrgMember
			}
			return nil, err
		}
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	page := filters.Page
	if page <= 0 {
		page = 1
	}

	offset := filters.Offset
	if offset <= 0 && page > 1 {
		offset = (page - 1) * limit
	}

	filters.Limit = limit
	filters.Offset = offset
	filters.Page = page

	logs, total, err := s.auditRepo.Query(ctx, exec, orgID, filters)
	if err != nil {
		return nil, err
	}

	hasMore := total > (offset + len(logs))

	return &dto.AuditLogListResponse{
		Data: logs,
		Pagination: dto.AuditPagination{
			Total:   total,
			Page:    page,
			Limit:   limit,
			Offset:  offset,
			HasMore: hasMore,
		},
	}, nil
}
