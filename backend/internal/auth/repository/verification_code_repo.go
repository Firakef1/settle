package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/Firakef1/settle/backend/internal/auth/model"
)

var ErrVerificationCodeNotFound = errors.New("verification code not found")

type VerificationCodeReader interface {
	FindByEmailAndCode(ctx context.Context, email, code string) (*model.VerificationCode, error)
	DeleteByEmail(ctx context.Context, email string) error
}

type VerificationCodeRepository interface {
	VerificationCodeReader
	Save(ctx context.Context, code *model.VerificationCode) error
	DeleteExpired(ctx context.Context) (int64, error)
}

type VerificationCodeRepo struct {
	db       *sql.DB
	mu       sync.RWMutex
	memCodes map[string]*model.VerificationCode // email (lower) -> code
}

func NewVerificationCodeRepo(db *sql.DB) *VerificationCodeRepo {
	return &VerificationCodeRepo{
		db:       db,
		memCodes: make(map[string]*model.VerificationCode),
	}
}

func (r *VerificationCodeRepo) Save(ctx context.Context, code *model.VerificationCode) error {
	emailLower := strings.ToLower(code.Email)
	if r.db != nil {
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback() //nolint:errcheck

		_, _ = tx.ExecContext(ctx, `DELETE FROM verification_codes WHERE LOWER(email) = $1`, emailLower)

		query := `
			INSERT INTO verification_codes (id, email, code, expires_at, created_at)
			VALUES ($1, $2, $3, $4, $5)`
		if _, err := tx.ExecContext(ctx, query, code.ID, emailLower, code.Code, code.ExpiresAt, code.CreatedAt); err != nil {
			return err
		}
		return tx.Commit()
	}

	// Memory fallback
	r.mu.Lock()
	defer r.mu.Unlock()
	c := *code
	c.Email = emailLower
	r.memCodes[emailLower] = &c
	return nil
}

func (r *VerificationCodeRepo) FindByEmailAndCode(ctx context.Context, email, code string) (*model.VerificationCode, error) {
	emailLower := strings.ToLower(email)
	if r.db != nil {
		query := `
			SELECT id, email, code, expires_at, created_at
			FROM verification_codes
			WHERE LOWER(email) = $1 AND code = $2`
		row := r.db.QueryRowContext(ctx, query, emailLower, code)

		var c model.VerificationCode
		err := row.Scan(&c.ID, &c.Email, &c.Code, &c.ExpiresAt, &c.CreatedAt)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrVerificationCodeNotFound
			}
			return nil, err
		}
		return &c, nil
	}

	// Memory fallback
	r.mu.RLock()
	defer r.mu.RUnlock()

	c, exists := r.memCodes[emailLower]
	if !exists || c.Code != code {
		return nil, ErrVerificationCodeNotFound
	}
	cCopy := *c
	return &cCopy, nil
}

func (r *VerificationCodeRepo) DeleteByEmail(ctx context.Context, email string) error {
	emailLower := strings.ToLower(email)
	if r.db != nil {
		_, err := r.db.ExecContext(ctx, `DELETE FROM verification_codes WHERE LOWER(email) = $1`, emailLower)
		return err
	}

	// Memory fallback
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.memCodes, emailLower)
	return nil
}

func (r *VerificationCodeRepo) DeleteExpired(ctx context.Context) (int64, error) {
	if r.db != nil {
		res, err := r.db.ExecContext(ctx, `DELETE FROM verification_codes WHERE expires_at <= NOW()`)
		if err != nil {
			return 0, err
		}
		return res.RowsAffected()
	}

	// Memory fallback
	r.mu.Lock()
	defer r.mu.Unlock()
	var count int64
	now := time.Now()
	for email, c := range r.memCodes {
		if now.After(c.ExpiresAt) {
			delete(r.memCodes, email)
			count++
		}
	}
	return count, nil
}
