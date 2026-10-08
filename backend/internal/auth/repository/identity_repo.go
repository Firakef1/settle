package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"

	"github.com/lib/pq"

	"github.com/Firakef1/settle/backend/internal/auth/model"
)

var (
	ErrIdentityNotFound      = errors.New("identity not found")
	ErrIdentityAlreadyExists = errors.New("identity already linked")
)

// IdentityRepository stores links between users and OAuth provider accounts.
type IdentityRepository interface {
	FindByProviderSubject(ctx context.Context, provider, subject string) (*model.UserIdentity, error)
	Create(ctx context.Context, identity *model.UserIdentity) error
}

// IdentityRepo implements IdentityRepository with database/sql or a memory fallback.
type IdentityRepo struct {
	db     *sql.DB
	memory map[string]*model.UserIdentity // provider|subject -> identity
	mu     sync.RWMutex
}

// NewIdentityRepo returns a repo backed by db, or by memory when db is nil.
func NewIdentityRepo(db *sql.DB) *IdentityRepo {
	return &IdentityRepo{db: db, memory: map[string]*model.UserIdentity{}}
}

func identityKey(provider, subject string) string { return strings.ToLower(provider) + "|" + subject }

func (r *IdentityRepo) FindByProviderSubject(ctx context.Context, provider, subject string) (*model.UserIdentity, error) {
	if r.db != nil {
		row := r.db.QueryRowContext(ctx, `
			SELECT id, user_id, provider, subject, email, created_at
			FROM user_identities
			WHERE provider = $1 AND subject = $2`, provider, subject)
		var i model.UserIdentity
		if err := row.Scan(&i.ID, &i.UserID, &i.Provider, &i.Subject, &i.Email, &i.CreatedAt); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrIdentityNotFound
			}
			return nil, err
		}
		return &i, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	i, ok := r.memory[identityKey(provider, subject)]
	if !ok {
		return nil, ErrIdentityNotFound
	}
	c := *i
	return &c, nil
}

func (r *IdentityRepo) Create(ctx context.Context, identity *model.UserIdentity) error {
	if r.db != nil {
		_, err := r.db.ExecContext(ctx, `
			INSERT INTO user_identities (id, user_id, provider, subject, email, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			identity.ID, identity.UserID, identity.Provider, identity.Subject, identity.Email, identity.CreatedAt)
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return ErrIdentityAlreadyExists
		}
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	key := identityKey(identity.Provider, identity.Subject)
	if _, exists := r.memory[key]; exists {
		return ErrIdentityAlreadyExists
	}
	c := *identity
	r.memory[key] = &c
	return nil
}
