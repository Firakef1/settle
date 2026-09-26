package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Firakef1/settle/backend/internal/organaization/model"
)

var (
	ErrInvitationNotFound = errors.New("invitation not found")
)

type InvitationRepository interface {
	Create(ctx context.Context, db DBTX, inv *model.Invitation) error
	GetByToken(ctx context.Context, db DBTX, token string) (*model.Invitation, error)
	MarkUsedTx(ctx context.Context, tx *sql.Tx, token string, usedAt time.Time) error
}

type pgInvitationRepo struct{}

func NewInvitationRepository() InvitationRepository {
	return &pgInvitationRepo{}
}

func (r *pgInvitationRepo) Create(ctx context.Context, db DBTX, inv *model.Invitation) error {
	query := `
		INSERT INTO invitations (id, org_id, email, role, token, expires_at, used_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := db.ExecContext(ctx, query, inv.ID, inv.OrgID, inv.Email, inv.Role, inv.Token, inv.ExpiresAt, inv.UsedAt, inv.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create invitation: %w", err)
	}
	return nil
}

func (r *pgInvitationRepo) GetByToken(ctx context.Context, db DBTX, token string) (*model.Invitation, error) {
	query := `
		SELECT id, org_id, email, role, token, expires_at, used_at, created_at
		FROM invitations
		WHERE token = $1
	`
	inv := &model.Invitation{}
	err := db.QueryRowContext(ctx, query, token).Scan(
		&inv.ID,
		&inv.OrgID,
		&inv.Email,
		&inv.Role,
		&inv.Token,
		&inv.ExpiresAt,
		&inv.UsedAt,
		&inv.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvitationNotFound
		}
		return nil, fmt.Errorf("failed to get invitation by token: %w", err)
	}
	return inv, nil
}

func (r *pgInvitationRepo) MarkUsedTx(ctx context.Context, tx *sql.Tx, token string, usedAt time.Time) error {
	query := `
		UPDATE invitations
		SET used_at = $1
		WHERE token = $2 AND used_at IS NULL
	`
	res, err := tx.ExecContext(ctx, query, usedAt, token)
	if err != nil {
		return fmt.Errorf("failed to mark invitation as used: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return errors.New("invitation already used or not found")
	}
	return nil
}
