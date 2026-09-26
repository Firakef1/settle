package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
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
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvitationExpired = errors.New("invitation token has expired")
	ErrInvitationUsed    = errors.New("invitation token has already been used")
	ErrAlreadyMember     = errors.New("user is already a member of this organization")
)

type InvitationService interface {
	CreateInvitation(ctx context.Context, orgID string, actorID string, req dto.InviteRequest) (*dto.InviteResponse, error)
	AcceptInvitation(ctx context.Context, token string, req dto.AcceptInviteRequest) (*dto.MemberResponse, error)
}

type invitationService struct {
	db          *sql.DB
	txRunner    TxRunner
	invRepo     repository.InvitationRepository
	memberRepo  repository.MemberRepository
	userRepo    repository.UserRepository
	auditWriter repository.AuditWriter
}

func NewInvitationService(
	db *sql.DB,
	invRepo repository.InvitationRepository,
	memberRepo repository.MemberRepository,
	userRepo repository.UserRepository,
	auditWriter repository.AuditWriter,
) InvitationService {
	var runner TxRunner
	if db != nil {
		runner = NewSQLTxRunner(db)
	} else {
		runner = &MockTxRunner{}
	}
	return &invitationService{
		db:          db,
		txRunner:    runner,
		invRepo:     invRepo,
		memberRepo:  memberRepo,
		userRepo:    userRepo,
		auditWriter: auditWriter,
	}
}

func NewInvitationServiceWithTx(
	txRunner TxRunner,
	invRepo repository.InvitationRepository,
	memberRepo repository.MemberRepository,
	userRepo repository.UserRepository,
	auditWriter repository.AuditWriter,
) InvitationService {
	return &invitationService{
		txRunner:    txRunner,
		invRepo:     invRepo,
		memberRepo:  memberRepo,
		userRepo:    userRepo,
		auditWriter: auditWriter,
	}
}

func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure random token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func (s *invitationService) getExecutor() repository.DBTX {
	if s.db != nil {
		return s.db
	}
	return nil
}

func (s *invitationService) CreateInvitation(ctx context.Context, orgID string, actorID string, req dto.InviteRequest) (*dto.InviteResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if err := validator.ValidateEmail(email); err != nil {
		return nil, err
	}

	role := strings.ToLower(strings.TrimSpace(req.Role))
	if err := validator.ValidateInviteRole(role); err != nil {
		return nil, err
	}

	token, err := generateSecureToken()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	expiresAt := now.Add(7 * 24 * time.Hour) // 7 days TTL

	inv := &model.Invitation{
		ID:        uuid.New().String(),
		OrgID:     orgID,
		Email:     email,
		Role:      role,
		Token:     token,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}

	exec := s.getExecutor()
	if err := s.invRepo.Create(ctx, exec, inv); err != nil {
		return nil, err
	}

	metaBytes, _ := json.Marshal(map[string]any{
		"email":      email,
		"role":       role,
		"expires_at": expiresAt,
	})
	_ = s.auditWriter.Write(ctx, exec, &model.AuditLog{
		ID:        uuid.New().String(),
		OrgID:     orgID,
		ActorID:   actorID,
		Action:    "invite_created",
		TargetID:  &inv.ID,
		Metadata:  string(metaBytes),
		CreatedAt: now,
	})

	return &dto.InviteResponse{
		ID:        inv.ID,
		OrgID:     inv.OrgID,
		Email:     inv.Email,
		Role:      inv.Role,
		Token:     inv.Token,
		ExpiresAt: inv.ExpiresAt,
		CreatedAt: inv.CreatedAt,
	}, nil
}

func (s *invitationService) AcceptInvitation(ctx context.Context, token string, req dto.AcceptInviteRequest) (*dto.MemberResponse, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("token cannot be empty")
	}

	exec := s.getExecutor()
	inv, err := s.invRepo.GetByToken(ctx, exec, token)
	if err != nil {
		return nil, err
	}

	if inv.UsedAt != nil {
		return nil, ErrInvitationUsed
	}

	now := time.Now().UTC()
	if now.After(inv.ExpiresAt) {
		return nil, ErrInvitationExpired
	}

	var memberResponse *dto.MemberResponse

	err = s.txRunner.WithTx(ctx, func(tx *sql.Tx) error {
		// 1. Check if user with invited email already exists
		var userID string
		var userName string
		user, err := s.userRepo.GetByEmail(ctx, tx, inv.Email)
		if err != nil {
			if errors.Is(err, repository.ErrUserNotFound) {
				// New user registration required
				if err := validator.ValidatePassword(req.Password); err != nil {
					return err
				}
				name := strings.TrimSpace(req.Name)
				if name == "" {
					parts := strings.Split(inv.Email, "@")
					name = parts[0]
				}

				hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
				if err != nil {
					return fmt.Errorf("failed to hash password: %w", err)
				}

				newUser := &model.User{
					ID:           uuid.New().String(),
					Name:         name,
					Email:        inv.Email,
					PasswordHash: string(hash),
					CreatedAt:    now,
					UpdatedAt:    now,
				}

				if err := s.userRepo.CreateTx(ctx, tx, newUser); err != nil {
					return fmt.Errorf("failed to create new user: %w", err)
				}
				userID = newUser.ID
				userName = newUser.Name
			} else {
				return fmt.Errorf("failed to lookup user: %w", err)
			}
		} else {
			userID = user.ID
			userName = user.Name
		}

		// 2. Check if already a member of this organization
		existingMember, err := s.memberRepo.GetMember(ctx, tx, inv.OrgID, userID)
		if err == nil && existingMember != nil {
			return ErrAlreadyMember
		}

		// 3. Add to org_members
		member := &model.OrgMember{
			OrgID:    inv.OrgID,
			UserID:   userID,
			Role:     inv.Role,
			JoinedAt: now,
		}
		if err := s.memberRepo.AddTx(ctx, tx, member); err != nil {
			return fmt.Errorf("failed to add member: %w", err)
		}

		// 4. Mark invitation as used
		if err := s.invRepo.MarkUsedTx(ctx, tx, token, now); err != nil {
			return fmt.Errorf("failed to mark invitation as used: %w", err)
		}

		// 5. Write audit log
		metaBytes, _ := json.Marshal(map[string]any{
			"email":   inv.Email,
			"user_id": userID,
			"role":    inv.Role,
		})
		auditLog := &model.AuditLog{
			ID:        uuid.New().String(),
			OrgID:     inv.OrgID,
			ActorID:   userID,
			Action:    "invite_accepted",
			TargetID:  &inv.ID,
			Metadata:  string(metaBytes),
			CreatedAt: now,
		}
		if err := s.auditWriter.WriteTx(ctx, tx, auditLog); err != nil {
			return fmt.Errorf("failed to write audit log: %w", err)
		}

		memberResponse = &dto.MemberResponse{
			UserID:   userID,
			Name:     userName,
			Email:    inv.Email,
			Role:     inv.Role,
			JoinedAt: now,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return memberResponse, nil
}
