package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Firakef1/settle/backend/internal/organaization/dto"
	"github.com/Firakef1/settle/backend/internal/organaization/model"
	"github.com/Firakef1/settle/backend/internal/organaization/repository"
	"github.com/Firakef1/settle/backend/internal/organaization/validator"
	"github.com/google/uuid"
)

var (
	ErrNotOrgMember = errors.New("forbidden: user is not a member of this organization")
)

type OrgService interface {
	CreateOrg(ctx context.Context, userID string, req dto.CreateOrgRequest) (*dto.OrgResponse, error)
	GetOrg(ctx context.Context, orgID string, callerUserID string) (*dto.OrgResponse, error)
	UpdateOrg(ctx context.Context, orgID string, actorID string, req dto.UpdateOrgRequest) (*dto.OrgResponse, error)
}

type orgService struct {
	db          *sql.DB
	txRunner    TxRunner
	orgRepo     repository.OrgRepository
	memberRepo  repository.MemberRepository
	auditWriter repository.AuditWriter
}

func NewOrgService(
	db *sql.DB,
	orgRepo repository.OrgRepository,
	memberRepo repository.MemberRepository,
	auditWriter repository.AuditWriter,
) OrgService {
	var runner TxRunner
	if db != nil {
		runner = NewSQLTxRunner(db)
	} else {
		runner = &MockTxRunner{}
	}
	return &orgService{
		db:          db,
		txRunner:    runner,
		orgRepo:     orgRepo,
		memberRepo:  memberRepo,
		auditWriter: auditWriter,
	}
}

func NewOrgServiceWithTx(
	txRunner TxRunner,
	orgRepo repository.OrgRepository,
	memberRepo repository.MemberRepository,
	auditWriter repository.AuditWriter,
) OrgService {
	return &orgService{
		txRunner:    txRunner,
		orgRepo:     orgRepo,
		memberRepo:  memberRepo,
		auditWriter: auditWriter,
	}
}

func generateSlug(name string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	re := regexp.MustCompile(`[^a-z0-9]+`)
	slug := re.ReplaceAllString(lower, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "org"
	}
	return fmt.Sprintf("%s-%s", slug, uuid.New().String()[:8])
}

func (s *orgService) getExecutor() repository.DBTX {
	if s.db != nil {
		return s.db
	}
	return nil
}

func (s *orgService) CreateOrg(ctx context.Context, userID string, req dto.CreateOrgRequest) (*dto.OrgResponse, error) {
	if err := validator.ValidateOrgName(req.Name); err != nil {
		return nil, err
	}
	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if err := validator.ValidateCurrency(currency); err != nil {
		return nil, err
	}

	orgID := uuid.New().String()
	now := time.Now().UTC()
	slug := generateSlug(req.Name)

	org := &model.Organization{
		ID:        orgID,
		Name:      strings.TrimSpace(req.Name),
		Slug:      slug,
		Currency:  currency,
		CreatedAt: now,
		UpdatedAt: now,
	}

	member := &model.OrgMember{
		OrgID:    orgID,
		UserID:   userID,
		Role:     "org_admin",
		JoinedAt: now,
	}

	metaBytes, _ := json.Marshal(map[string]any{
		"org_name": org.Name,
		"currency": org.Currency,
	})

	auditLog := &model.AuditLog{
		ID:        uuid.New().String(),
		OrgID:     orgID,
		ActorID:   userID,
		Action:    "org_created",
		TargetID:  &orgID,
		Metadata:  string(metaBytes),
		CreatedAt: now,
	}

	err := s.txRunner.WithTx(ctx, func(tx *sql.Tx) error {
		if err := s.orgRepo.CreateTx(ctx, tx, org); err != nil {
			return fmt.Errorf("failed to create organization: %w", err)
		}

		if err := s.memberRepo.AddTx(ctx, tx, member); err != nil {
			return fmt.Errorf("failed to add creator as org_admin: %w", err)
		}

		if err := s.auditWriter.WriteTx(ctx, tx, auditLog); err != nil {
			return fmt.Errorf("failed to write audit log: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &dto.OrgResponse{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		Currency:  org.Currency,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	}, nil
}

func (s *orgService) GetOrg(ctx context.Context, orgID string, callerUserID string) (*dto.OrgResponse, error) {
	exec := s.getExecutor()
	// Scoped to org members only
	_, err := s.memberRepo.GetMember(ctx, exec, orgID, callerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrMemberNotFound) {
			return nil, ErrNotOrgMember
		}
		return nil, err
	}

	org, err := s.orgRepo.GetByID(ctx, exec, orgID)
	if err != nil {
		return nil, err
	}

	return &dto.OrgResponse{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		Currency:  org.Currency,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	}, nil
}

func (s *orgService) UpdateOrg(ctx context.Context, orgID string, actorID string, req dto.UpdateOrgRequest) (*dto.OrgResponse, error) {
	exec := s.getExecutor()
	org, err := s.orgRepo.GetByID(ctx, exec, orgID)
	if err != nil {
		return nil, err
	}

	updated := false
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if err := validator.ValidateOrgName(name); err != nil {
			return nil, err
		}
		org.Name = name
		updated = true
	}

	if req.Currency != nil {
		curr := strings.ToUpper(strings.TrimSpace(*req.Currency))
		if err := validator.ValidateCurrency(curr); err != nil {
			return nil, err
		}
		org.Currency = curr
		updated = true
	}

	if updated {
		org.UpdatedAt = time.Now().UTC()
		if err := s.orgRepo.Update(ctx, exec, org); err != nil {
			return nil, err
		}

		metaBytes, _ := json.Marshal(map[string]any{
			"name":     org.Name,
			"currency": org.Currency,
		})
		_ = s.auditWriter.Write(ctx, exec, &model.AuditLog{
			ID:        uuid.New().String(),
			OrgID:     orgID,
			ActorID:   actorID,
			Action:    "org_updated",
			TargetID:  &orgID,
			Metadata:  string(metaBytes),
			CreatedAt: time.Now().UTC(),
		})
	}

	return &dto.OrgResponse{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		Currency:  org.Currency,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	}, nil
}
