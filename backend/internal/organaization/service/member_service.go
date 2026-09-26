package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Firakef1/settle/backend/internal/organaization/dto"
	"github.com/Firakef1/settle/backend/internal/organaization/model"
	"github.com/Firakef1/settle/backend/internal/organaization/repository"
	"github.com/Firakef1/settle/backend/internal/organaization/validator"
	"github.com/google/uuid"
)

var (
	ErrCannotRemoveLastAdmin = errors.New("cannot remove the last organization admin")
	ErrCannotDemoteLastAdmin = errors.New("cannot demote the last organization admin")
)

type MemberService interface {
	ListMembers(ctx context.Context, orgID string) ([]*dto.MemberResponse, error)
	RemoveMember(ctx context.Context, orgID string, targetUID string, actorID string) error
	UpdateRole(ctx context.Context, orgID string, targetUID string, newRole string, actorID string) error
}

type memberService struct {
	db          *sql.DB
	memberRepo  repository.MemberRepository
	auditWriter repository.AuditWriter
}

func NewMemberService(
	db *sql.DB,
	memberRepo repository.MemberRepository,
	auditWriter repository.AuditWriter,
) MemberService {
	return &memberService{
		db:          db,
		memberRepo:  memberRepo,
		auditWriter: auditWriter,
	}
}

func (s *memberService) getExecutor() repository.DBTX {
	if s.db != nil {
		return s.db
	}
	return nil
}

func (s *memberService) ListMembers(ctx context.Context, orgID string) ([]*dto.MemberResponse, error) {
	exec := s.getExecutor()
	return s.memberRepo.ListByOrgID(ctx, exec, orgID)
}

func (s *memberService) RemoveMember(ctx context.Context, orgID string, targetUID string, actorID string) error {
	exec := s.getExecutor()

	member, err := s.memberRepo.GetMember(ctx, exec, orgID, targetUID)
	if err != nil {
		return err
	}

	if member.Role == "org_admin" {
		adminCount, err := s.memberRepo.CountAdmins(ctx, exec, orgID)
		if err != nil {
			return fmt.Errorf("failed to check admin count: %w", err)
		}
		if adminCount <= 1 {
			return ErrCannotRemoveLastAdmin
		}
	}

	if err := s.memberRepo.Delete(ctx, exec, orgID, targetUID); err != nil {
		return err
	}

	metaBytes, _ := json.Marshal(map[string]any{
		"target_user_id": targetUID,
		"removed_role":   member.Role,
	})
	_ = s.auditWriter.Write(ctx, exec, &model.AuditLog{
		ID:        uuid.New().String(),
		OrgID:     orgID,
		ActorID:   actorID,
		Action:    "member_removed",
		TargetID:  &targetUID,
		Metadata:  string(metaBytes),
		CreatedAt: time.Now().UTC(),
	})

	return nil
}

func (s *memberService) UpdateRole(ctx context.Context, orgID string, targetUID string, newRole string, actorID string) error {
	exec := s.getExecutor()

	role := strings.ToLower(strings.TrimSpace(newRole))
	if err := validator.ValidateInviteRole(role); err != nil {
		return err
	}

	member, err := s.memberRepo.GetMember(ctx, exec, orgID, targetUID)
	if err != nil {
		return err
	}

	if member.Role == "org_admin" {
		adminCount, err := s.memberRepo.CountAdmins(ctx, exec, orgID)
		if err != nil {
			return fmt.Errorf("failed to check admin count: %w", err)
		}
		if adminCount <= 1 {
			return ErrCannotDemoteLastAdmin
		}
	}

	if err := s.memberRepo.UpdateRole(ctx, exec, orgID, targetUID, role); err != nil {
		return err
	}

	metaBytes, _ := json.Marshal(map[string]any{
		"target_user_id": targetUID,
		"old_role":       member.Role,
		"new_role":       role,
	})
	_ = s.auditWriter.Write(ctx, exec, &model.AuditLog{
		ID:        uuid.New().String(),
		OrgID:     orgID,
		ActorID:   actorID,
		Action:    "role_updated",
		TargetID:  &targetUID,
		Metadata:  string(metaBytes),
		CreatedAt: time.Now().UTC(),
	})

	return nil
}
