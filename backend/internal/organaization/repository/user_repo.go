package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Firakef1/settle/backend/internal/organaization/model"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

type UserRepository interface {
	GetByEmail(ctx context.Context, db DBTX, email string) (*model.User, error)
	CreateTx(ctx context.Context, tx *sql.Tx, user *model.User) error
}

type pgUserRepo struct{}

func NewUserRepository() UserRepository {
	return &pgUserRepo{}
}

func (r *pgUserRepo) GetByEmail(ctx context.Context, db DBTX, email string) (*model.User, error) {
	query := `
		SELECT id, name, email, password_hash, created_at, updated_at
		FROM users
		WHERE LOWER(email) = LOWER($1)
	`
	user := &model.User{}
	err := db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return user, nil
}

func (r *pgUserRepo) CreateTx(ctx context.Context, tx *sql.Tx, user *model.User) error {
	query := `
		INSERT INTO users (id, name, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := tx.ExecContext(ctx, query, user.ID, user.Name, user.Email, user.PasswordHash, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create user in tx: %w", err)
	}
	return nil
}
