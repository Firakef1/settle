package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Firakef1/settle/backend/internal/organaization/model"
)

var (
	ErrOrgNotFound = errors.New("organization not found")
)

type OrgRepository interface {
	CreateTx(ctx context.Context, tx *sql.Tx, org *model.Organization) error
	GetByID(ctx context.Context, db DBTX, id string) (*model.Organization, error)
	GetBySlug(ctx context.Context, db DBTX, slug string) (*model.Organization, error)
	Update(ctx context.Context, db DBTX, org *model.Organization) error
	UpdatePlan(ctx context.Context, db DBTX, orgID string, plan string) error
}

type pgOrgRepo struct{}

func NewOrgRepository() OrgRepository {
	return &pgOrgRepo{}
}

func (r *pgOrgRepo) CreateTx(ctx context.Context, tx *sql.Tx, org *model.Organization) error {
	query := `
		INSERT INTO organizations (id, name, slug, currency, plan, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	plan := org.Plan
	if plan == "" {
		plan = "free"
	}
	_, err := tx.ExecContext(ctx, query, org.ID, org.Name, org.Slug, org.Currency, plan, org.CreatedAt, org.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert organization: %w", err)
	}
	return nil
}

func (r *pgOrgRepo) GetByID(ctx context.Context, db DBTX, id string) (*model.Organization, error) {
	query := `
		SELECT id, name, slug, currency, COALESCE(plan, 'free'), created_at, updated_at
		FROM organizations
		WHERE id = $1
	`
	org := &model.Organization{}
	err := db.QueryRowContext(ctx, query, id).Scan(
		&org.ID,
		&org.Name,
		&org.Slug,
		&org.Currency,
		&org.Plan,
		&org.CreatedAt,
		&org.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrgNotFound
		}
		return nil, fmt.Errorf("failed to get organization by id: %w", err)
	}
	return org, nil
}

func (r *pgOrgRepo) GetBySlug(ctx context.Context, db DBTX, slug string) (*model.Organization, error) {
	query := `
		SELECT id, name, slug, currency, COALESCE(plan, 'free'), created_at, updated_at
		FROM organizations
		WHERE slug = $1
	`
	org := &model.Organization{}
	err := db.QueryRowContext(ctx, query, slug).Scan(
		&org.ID,
		&org.Name,
		&org.Slug,
		&org.Currency,
		&org.Plan,
		&org.CreatedAt,
		&org.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrgNotFound
		}
		return nil, fmt.Errorf("failed to get organization by slug: %w", err)
	}
	return org, nil
}

func (r *pgOrgRepo) Update(ctx context.Context, db DBTX, org *model.Organization) error {
	query := `
		UPDATE organizations
		SET name = $1, currency = $2, plan = $3, updated_at = $4
		WHERE id = $5
	`
	plan := org.Plan
	if plan == "" {
		plan = "free"
	}
	res, err := db.ExecContext(ctx, query, org.Name, org.Currency, plan, org.UpdatedAt, org.ID)
	if err != nil {
		return fmt.Errorf("failed to update organization: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrOrgNotFound
	}
	return nil
}

func (r *pgOrgRepo) UpdatePlan(ctx context.Context, db DBTX, orgID string, plan string) error {
	query := `
		UPDATE organizations
		SET plan = $1, updated_at = NOW()
		WHERE id = $2
	`
	res, err := db.ExecContext(ctx, query, plan, orgID)
	if err != nil {
		return fmt.Errorf("failed to update organization plan: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrOrgNotFound
	}
	return nil
}
