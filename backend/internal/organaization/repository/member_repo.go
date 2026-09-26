package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Firakef1/settle/backend/internal/organaization/dto"
	"github.com/Firakef1/settle/backend/internal/organaization/model"
)

var (
	ErrMemberNotFound = errors.New("member not found")
)

type MemberRepository interface {
	AddTx(ctx context.Context, tx *sql.Tx, member *model.OrgMember) error
	Add(ctx context.Context, db DBTX, member *model.OrgMember) error
	GetMember(ctx context.Context, db DBTX, orgID, userID string) (*model.OrgMember, error)
	ListByOrgID(ctx context.Context, db DBTX, orgID string) ([]*dto.MemberResponse, error)
	CountAdmins(ctx context.Context, db DBTX, orgID string) (int, error)
	Delete(ctx context.Context, db DBTX, orgID, userID string) error
	UpdateRole(ctx context.Context, db DBTX, orgID, userID, newRole string) error
}

type pgMemberRepo struct{}

func NewMemberRepository() MemberRepository {
	return &pgMemberRepo{}
}

func (r *pgMemberRepo) AddTx(ctx context.Context, tx *sql.Tx, member *model.OrgMember) error {
	query := `
		INSERT INTO org_members (org_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := tx.ExecContext(ctx, query, member.OrgID, member.UserID, member.Role, member.JoinedAt)
	if err != nil {
		return fmt.Errorf("failed to add member in tx: %w", err)
	}
	return nil
}

func (r *pgMemberRepo) Add(ctx context.Context, db DBTX, member *model.OrgMember) error {
	query := `
		INSERT INTO org_members (org_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := db.ExecContext(ctx, query, member.OrgID, member.UserID, member.Role, member.JoinedAt)
	if err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}
	return nil
}

func (r *pgMemberRepo) GetMember(ctx context.Context, db DBTX, orgID, userID string) (*model.OrgMember, error) {
	query := `
		SELECT org_id, user_id, role, joined_at
		FROM org_members
		WHERE org_id = $1 AND user_id = $2
	`
	member := &model.OrgMember{}
	err := db.QueryRowContext(ctx, query, orgID, userID).Scan(
		&member.OrgID,
		&member.UserID,
		&member.Role,
		&member.JoinedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMemberNotFound
		}
		return nil, fmt.Errorf("failed to get member: %w", err)
	}
	return member, nil
}

func (r *pgMemberRepo) ListByOrgID(ctx context.Context, db DBTX, orgID string) ([]*dto.MemberResponse, error) {
	query := `
		SELECT m.user_id, COALESCE(u.name, ''), COALESCE(u.email, ''), m.role, m.joined_at
		FROM org_members m
		LEFT JOIN users u ON m.user_id = u.id
		WHERE m.org_id = $1
		ORDER BY m.joined_at ASC
	`
	rows, err := db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list members: %w", err)
	}
	defer rows.Close()

	var members []*dto.MemberResponse
	for rows.Next() {
		m := &dto.MemberResponse{}
		if err := rows.Scan(&m.UserID, &m.Name, &m.Email, &m.Role, &m.JoinedAt); err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return members, nil
}

func (r *pgMemberRepo) CountAdmins(ctx context.Context, db DBTX, orgID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM org_members
		WHERE org_id = $1 AND role = 'org_admin'
	`
	var count int
	err := db.QueryRowContext(ctx, query, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count org admins: %w", err)
	}
	return count, nil
}

func (r *pgMemberRepo) Delete(ctx context.Context, db DBTX, orgID, userID string) error {
	query := `
		DELETE FROM org_members
		WHERE org_id = $1 AND user_id = $2
	`
	res, err := db.ExecContext(ctx, query, orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete member: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrMemberNotFound
	}
	return nil
}

func (r *pgMemberRepo) UpdateRole(ctx context.Context, db DBTX, orgID, userID, newRole string) error {
	query := `
		UPDATE org_members
		SET role = $1
		WHERE org_id = $2 AND user_id = $3
	`
	res, err := db.ExecContext(ctx, query, newRole, orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrMemberNotFound
	}
	return nil
}
