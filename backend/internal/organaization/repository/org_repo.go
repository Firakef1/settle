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
}

type pgOrgRepo struct{}

func NewOrgRepository() OrgRepository {
	return &pgOrgRepo{}
}

func (r *pgOrgRepo) CreateTx(ctx context.Context, tx *sql.Tx, org *model.Organization) error {
	query := `
		INSERT INTO organizations (id, name, slug, currency, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := tx.ExecContext(ctx, query, org.ID, org.Name, org.Slug, org.Currency, org.CreatedAt, org.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert organization: %w", err)
	}
	return nil
}

func (r *pgOrgRepo) GetByID(ctx context.Context, db DBTX, id string) (*model.Organization, error) {
	query := `
		SELECT id, name, slug, currency, created_at, updated_at
		FROM organizations
		WHERE id = $1
	`
	org := &model.Organization{}
	err := db.QueryRowContext(ctx, query, id).Scan(
		&org.ID,
		&org.Name,
		&org.Slug,
		&org.Currency,
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
		SELECT id, name, slug, currency, created_at, updated_at
		FROM organizations
		WHERE slug = $1
	`
	org := &model.Organization{}
	err := db.QueryRowContext(ctx, query, slug).Scan(
		&org.ID,
		&org.Name,
		&org.Slug,
		&org.Currency,
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
		SET name = $1, currency = $2, updated_at = $3
		WHERE id = $4
	`
	res, err := db.ExecContext(ctx, query, org.Name, org.Currency, org.UpdatedAt, org.ID)
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
