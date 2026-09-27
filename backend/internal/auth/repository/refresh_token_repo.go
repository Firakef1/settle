package repository

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/Firakef1/settle/backend/internal/auth/model"
)

var (
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	ErrRefreshTokenRevoked  = errors.New("refresh token has been revoked")
	ErrRefreshTokenExpired  = errors.New("refresh token has expired")
)

// RefreshTokenRepository defines interface for refresh token data operations.
type RefreshTokenRepository interface {
	StoreRefreshToken(ctx context.Context, token *model.RefreshToken) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	RevokeByTokenHash(ctx context.Context, tokenHash string) error
	RevokeAllForUser(ctx context.Context, userID string) error
}

// RefreshTokenRepo implements RefreshTokenRepository using database/sql or memory fallback.
type RefreshTokenRepo struct {
	db           *sql.DB
	memoryTokens map[string]*model.RefreshToken // tokenHash -> token
	mu           sync.RWMutex
}

// NewRefreshTokenRepo creates a new RefreshTokenRepo instance.
func NewRefreshTokenRepo(db *sql.DB) *RefreshTokenRepo {
	return &RefreshTokenRepo{
		db:           db,
		memoryTokens: make(map[string]*model.RefreshToken),
	}
}

// StoreRefreshToken saves a new refresh token.
func (r *RefreshTokenRepo) StoreRefreshToken(ctx context.Context, token *model.RefreshToken) error {
	if r.db != nil {
		query := `
			INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`
		_, err := r.db.ExecContext(ctx, query,
			token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.Revoked, token.CreatedAt, token.UpdatedAt,
		)
		return err
	}

	// Memory fallback
	r.mu.Lock()
	defer r.mu.Unlock()
	tCopy := *token
	r.memoryTokens[token.TokenHash] = &tCopy
	return nil
}

// FindByTokenHash retrieves a refresh token by its hash.
func (r *RefreshTokenRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	if r.db != nil {
		query := `
			SELECT id, user_id, token_hash, expires_at, revoked, created_at, updated_at
			FROM refresh_tokens
			WHERE token_hash = $1`
		row := r.db.QueryRowContext(ctx, query, tokenHash)

		var t model.RefreshToken
		err := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.Revoked, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrRefreshTokenNotFound
			}
			return nil, err
		}
		return &t, nil
	}

	// Memory fallback
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, exists := r.memoryTokens[tokenHash]
	if !exists {
		return nil, ErrRefreshTokenNotFound
	}
	tCopy := *t
	return &tCopy, nil
}

// RevokeByTokenHash marks a refresh token as revoked.
func (r *RefreshTokenRepo) RevokeByTokenHash(ctx context.Context, tokenHash string) error {
	if r.db != nil {
		query := `UPDATE refresh_tokens SET revoked = TRUE, updated_at = $1 WHERE token_hash = $2`
		_, err := r.db.ExecContext(ctx, query, time.Now(), tokenHash)
		return err
	}

	// Memory fallback
	r.mu.Lock()
	defer r.mu.Unlock()

	t, exists := r.memoryTokens[tokenHash]
	if !exists {
		return ErrRefreshTokenNotFound
	}
	t.Revoked = true
	t.UpdatedAt = time.Now()
	return nil
}

// RevokeAllForUser marks all refresh tokens for a user as revoked.
func (r *RefreshTokenRepo) RevokeAllForUser(ctx context.Context, userID string) error {
	if r.db != nil {
		query := `UPDATE refresh_tokens SET revoked = TRUE, updated_at = $1 WHERE user_id = $2 AND revoked = FALSE`
		_, err := r.db.ExecContext(ctx, query, time.Now(), userID)
		return err
	}

	// Memory fallback
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for _, t := range r.memoryTokens {
		if t.UserID == userID && !t.Revoked {
			t.Revoked = true
			t.UpdatedAt = now
		}
	}
	return nil
}
