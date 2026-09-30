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

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

// UserRepository defines interface for user data operations.
type UserRepository interface {
	CreateUser(ctx context.Context, user *model.User) error
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, userID string) (*model.User, error)
	GetOrgMemberships(ctx context.Context, userID string) ([]model.OrgMembership, error)
	UpdateUnverifiedCredentials(ctx context.Context, userID, name, passwordHash string) error
	MarkEmailAsVerified(ctx context.Context, email string) error
	UpdatePassword(ctx context.Context, userID, newPasswordHash string) error
	SoftDeleteUser(ctx context.Context, userID string) error
}

// UserRepo implements UserRepository using database/sql or memory fallback.
type UserRepo struct {
	db          *sql.DB
	memoryUsers map[string]*model.User           // email lower -> user
	memoryOrgs  map[string][]model.OrgMembership // userID -> orgs
	mu          sync.RWMutex
}

// NewUserRepo creates a new UserRepo instance.
func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{
		db:          db,
		memoryUsers: make(map[string]*model.User),
		memoryOrgs:  make(map[string][]model.OrgMembership),
	}
}

// CreateUser saves a new user to database or memory store.
func (r *UserRepo) CreateUser(ctx context.Context, user *model.User) error {
	if r.db != nil {
		query := `
			INSERT INTO users (id, email, name, password_hash, status, email_verified, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
		_, err := r.db.ExecContext(ctx, query,
			user.ID,
			user.Email,
			user.Name,
			user.PasswordHash,
			user.Status,
			user.EmailVerified,
			user.CreatedAt,
			user.UpdatedAt,
		)
		if err != nil {
			if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
				return ErrUserAlreadyExists
			}
			return err
		}
		return nil
	}

	// Memory fallback
	r.mu.Lock()
	defer r.mu.Unlock()

	emailLower := strings.ToLower(user.Email)
	if _, exists := r.memoryUsers[emailLower]; exists {
		return ErrUserAlreadyExists
	}

	uCopy := *user
	r.memoryUsers[emailLower] = &uCopy
	return nil
}

// FindByEmail searches active user by email.
func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	emailLower := strings.ToLower(email)

	if r.db != nil {
		query := `
			SELECT id, email, name, password_hash, status, email_verified, created_at, updated_at
			FROM users
			WHERE LOWER(email) = $1 AND status = 'active'`
		row := r.db.QueryRowContext(ctx, query, emailLower)

		var u model.User
		err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.Status, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrUserNotFound
			}
			return nil, err
		}
		return &u, nil
	}

	// Memory fallback
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, exists := r.memoryUsers[emailLower]
	if !exists || u.Status != "active" {
		return nil, ErrUserNotFound
	}
	uCopy := *u
	return &uCopy, nil
}

// FindByID searches user by ID.
func (r *UserRepo) FindByID(ctx context.Context, userID string) (*model.User, error) {
	if r.db != nil {
		query := `
			SELECT id, email, name, password_hash, status, email_verified, created_at, updated_at
			FROM users
			WHERE id = $1 AND status = 'active'`
		row := r.db.QueryRowContext(ctx, query, userID)

		var u model.User
		err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.Status, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrUserNotFound
			}
			return nil, err
		}
		return &u, nil
	}

	// Memory fallback
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, u := range r.memoryUsers {
		if u.ID == userID && u.Status == "active" {
			uCopy := *u
			return &uCopy, nil
		}
	}
	return nil, ErrUserNotFound
}

// GetOrgMemberships retrieves organization memberships for user.
func (r *UserRepo) GetOrgMemberships(ctx context.Context, userID string) ([]model.OrgMembership, error) {
	if r.db != nil {
		query := `
			SELECT om.id, om.org_id, COALESCE(o.name, ''), COALESCE(o.slug, ''), om.user_id, om.role, COALESCE(om.department, ''), om.status
			FROM org_members om
			LEFT JOIN organizations o ON o.id = om.org_id
			WHERE om.user_id = $1 AND om.status = 'active'`
		rows, err := r.db.QueryContext(ctx, query, userID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var memberships []model.OrgMembership
		for rows.Next() {
			var m model.OrgMembership
			err := rows.Scan(&m.ID, &m.OrgID, &m.OrgName, &m.OrgSlug, &m.UserID, &m.Role, &m.Department, &m.Status)
			if err != nil {
				return nil, err
			}
			memberships = append(memberships, m)
		}
		return memberships, nil
	}

	// Memory fallback
	r.mu.RLock()
	defer r.mu.RUnlock()

	orgs, exists := r.memoryOrgs[userID]
	if !exists {
		return []model.OrgMembership{}, nil
	}
	return orgs, nil
}

// UpdateUnverifiedCredentials updates name and password_hash for a user whose email is not yet verified.
// Returns ErrUserNotFound if 0 rows are affected (already verified or user gone).
func (r *UserRepo) UpdateUnverifiedCredentials(ctx context.Context, userID, name, passwordHash string) error {
	if r.db != nil {
		res, err := r.db.ExecContext(ctx,
			`UPDATE users SET name = $2, password_hash = $3, updated_at = NOW()
			 WHERE id = $1 AND email_verified = FALSE`,
			userID, name, passwordHash,
		)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrUserNotFound
		}
		return nil
	}

	// Memory fallback.
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.memoryUsers {
		if u.ID == userID && !u.EmailVerified {
			u.Name = name
			u.PasswordHash = passwordHash
			return nil
		}
	}
	return ErrUserNotFound
}

// AddMemoryOrgMembership helper for testing memory memberships.
func (r *UserRepo) AddMemoryOrgMembership(membership model.OrgMembership) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.memoryOrgs[membership.UserID] = append(r.memoryOrgs[membership.UserID], membership)
}

// MarkEmailAsVerified sets email_verified = TRUE for the user with the given email.
func (r *UserRepo) MarkEmailAsVerified(ctx context.Context, email string) error {
	emailLower := strings.ToLower(email)
	if r.db != nil {
		_, err := r.db.ExecContext(ctx,
			`UPDATE users SET email_verified = TRUE, updated_at = NOW() WHERE LOWER(email) = $1`,
			emailLower,
		)
		return err
	}

	// Memory fallback.
	r.mu.Lock()
	defer r.mu.Unlock()
	u, exists := r.memoryUsers[emailLower]
	if !exists {
		return ErrUserNotFound
	}
	u.EmailVerified = true
	return nil
}

// SetEmailVerifiedForTest is a test helper that marks the user with the given (lowercased) email as verified.
func (r *UserRepo) SetEmailVerifiedForTest(email string, verified bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	emailLower := strings.ToLower(email)
	if u, ok := r.memoryUsers[emailLower]; ok {
		u.EmailVerified = verified
	}
}

// UpdatePassword updates a user's password_hash.
func (r *UserRepo) UpdatePassword(ctx context.Context, userID, newPasswordHash string) error {
	if r.db != nil {
		res, err := r.db.ExecContext(ctx,
			`UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1 AND status = 'active'`,
			userID, newPasswordHash,
		)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrUserNotFound
		}
		return nil
	}

	// Memory fallback
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.memoryUsers {
		if u.ID == userID && u.Status == "active" {
			u.PasswordHash = newPasswordHash
			u.UpdatedAt = time.Now()
			return nil
		}
	}
	return ErrUserNotFound
}

// SoftDeleteUser sets user status = 'deleted' and deleted_at = NOW().
func (r *UserRepo) SoftDeleteUser(ctx context.Context, userID string) error {
	if r.db != nil {
		res, err := r.db.ExecContext(ctx,
			`UPDATE users SET status = 'deleted', deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND status = 'active'`,
			userID,
		)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrUserNotFound
		}
		return nil
	}

	// Memory fallback
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.memoryUsers {
		if u.ID == userID && u.Status == "active" {
			u.Status = "deleted"
			now := time.Now()
			u.DeletedAt = &now
			u.UpdatedAt = now
			return nil
		}
	}
	return ErrUserNotFound
}
