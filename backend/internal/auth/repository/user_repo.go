package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"

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
			INSERT INTO users (id, email, name, password_hash, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`
		_, err := r.db.ExecContext(ctx, query,
			user.ID,
			user.Email,
			user.Name,
			user.PasswordHash,
			user.Status,
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
			SELECT id, email, name, password_hash, status, created_at, updated_at
			FROM users
			WHERE LOWER(email) = $1 AND status = 'active'`
		row := r.db.QueryRowContext(ctx, query, emailLower)

		var u model.User
		err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.Status, &u.CreatedAt, &u.UpdatedAt)
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
			SELECT id, email, name, password_hash, status, created_at, updated_at
			FROM users
			WHERE id = $1 AND status = 'active'`
		row := r.db.QueryRowContext(ctx, query, userID)

		var u model.User
		err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.Status, &u.CreatedAt, &u.UpdatedAt)
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

// AddMemoryOrgMembership helper for testing memory memberships.
func (r *UserRepo) AddMemoryOrgMembership(membership model.OrgMembership) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.memoryOrgs[membership.UserID] = append(r.memoryOrgs[membership.UserID], membership)
}
