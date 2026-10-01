package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Firakef1/settle/backend/internal/requests/dto"
	"github.com/Firakef1/settle/backend/internal/requests/model"
)

var (
	ErrRequestNotFound = errors.New("request not found")
)

type RequestRepository interface {
	WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error
	CreateTx(ctx context.Context, tx *sql.Tx, req *model.Request) error
	Create(ctx context.Context, req *model.Request) error
	GetByID(ctx context.Context, id string) (*model.Request, error)
	GetByIDTx(ctx context.Context, tx *sql.Tx, id string) (*model.Request, error)
	UpdateStatusTx(ctx context.Context, tx *sql.Tx, id, status string, submittedAt *time.Time) error
	UpdateStatus(ctx context.Context, id, status string) error
	List(ctx context.Context, orgID string, filters dto.ListFilters) ([]model.Request, int, error)
}

type RequestRepo struct {
	db             *sql.DB
	memoryRequests map[string]*model.Request // id -> request
	mu             sync.RWMutex
}

func NewRequestRepo(db *sql.DB) *RequestRepo {
	return &RequestRepo{
		db:             db,
		memoryRequests: make(map[string]*model.Request),
	}
}

func (r *RequestRepo) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	if r.db == nil {
		// Memory fallback: no actual transaction support, just run it
		return fn(nil)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *RequestRepo) Create(ctx context.Context, req *model.Request) error {
	return r.CreateTx(ctx, nil, req)
}

func (r *RequestRepo) CreateTx(ctx context.Context, tx *sql.Tx, req *model.Request) error {
	if r.db != nil {
		query := `
			INSERT INTO requests (id, org_id, requester_id, type, amount, purpose, urgency, status, resubmitted_as, submitted_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
		var err error
		if tx != nil {
			_, err = tx.ExecContext(ctx, query,
				req.ID, req.OrgID, req.RequesterID, req.Type, req.Amount, req.Purpose, req.Urgency, req.Status, req.ResubmittedAs, req.SubmittedAt, req.CreatedAt, req.UpdatedAt,
			)
		} else {
			_, err = r.db.ExecContext(ctx, query,
				req.ID, req.OrgID, req.RequesterID, req.Type, req.Amount, req.Purpose, req.Urgency, req.Status, req.ResubmittedAs, req.SubmittedAt, req.CreatedAt, req.UpdatedAt,
			)
		}
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	c := *req
	r.memoryRequests[req.ID] = &c
	return nil
}

func (r *RequestRepo) GetByID(ctx context.Context, id string) (*model.Request, error) {
	return r.GetByIDTx(ctx, nil, id)
}

func (r *RequestRepo) GetByIDTx(ctx context.Context, tx *sql.Tx, id string) (*model.Request, error) {
	if r.db != nil {
		query := `
			SELECT id, org_id, requester_id, type, amount, purpose, urgency, status, resubmitted_as, submitted_at, created_at, updated_at
			FROM requests
			WHERE id = $1`
		var row *sql.Row
		if tx != nil {
			row = tx.QueryRowContext(ctx, query, id)
		} else {
			row = r.db.QueryRowContext(ctx, query, id)
		}

		var req model.Request
		err := row.Scan(&req.ID, &req.OrgID, &req.RequesterID, &req.Type, &req.Amount, &req.Purpose, &req.Urgency, &req.Status, &req.ResubmittedAs, &req.SubmittedAt, &req.CreatedAt, &req.UpdatedAt)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrRequestNotFound
			}
			return nil, err
		}
		return &req, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	req, exists := r.memoryRequests[id]
	if !exists {
		return nil, ErrRequestNotFound
	}
	c := *req
	return &c, nil
}

func (r *RequestRepo) UpdateStatus(ctx context.Context, id, status string) error {
	return r.UpdateStatusTx(ctx, nil, id, status, nil)
}

func (r *RequestRepo) UpdateStatusTx(ctx context.Context, tx *sql.Tx, id, status string, submittedAt *time.Time) error {
	if r.db != nil {
		query := `
			UPDATE requests
			SET status = $1, submitted_at = COALESCE($2, submitted_at), updated_at = $3
			WHERE id = $4`
		var err error
		if tx != nil {
			_, err = tx.ExecContext(ctx, query, status, submittedAt, time.Now(), id)
		} else {
			_, err = r.db.ExecContext(ctx, query, status, submittedAt, time.Now(), id)
		}
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	req, exists := r.memoryRequests[id]
	if !exists {
		return ErrRequestNotFound
	}
	req.Status = status
	if submittedAt != nil {
		req.SubmittedAt = submittedAt
	}
	req.UpdatedAt = time.Now()
	return nil
}

func (r *RequestRepo) List(ctx context.Context, orgID string, filters dto.ListFilters) ([]model.Request, int, error) {
	if r.db != nil {
		whereClause := "WHERE org_id = $1"
		args := []any{orgID}
		argID := 2

		if filters.Status != "" {
			whereClause += fmt.Sprintf(" AND status = $%d", argID)
			args = append(args, filters.Status)
			argID++
		}
		if filters.Urgency != "" {
			whereClause += fmt.Sprintf(" AND urgency = $%d", argID)
			args = append(args, filters.Urgency)
			argID++
		}
		if filters.RequesterID != "" {
			whereClause += fmt.Sprintf(" AND requester_id = $%d", argID)
			args = append(args, filters.RequesterID)
			argID++
		}
		if filters.ExcludeDrafts {
			whereClause += " AND status != 'draft'"
		}

		countQuery := "SELECT COUNT(*) FROM requests " + whereClause
		var total int
		err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
		if err != nil {
			return nil, 0, err
		}

		orderClause := "ORDER BY created_at DESC"
		if filters.SortBy == "created_at" {
			orderClause = "ORDER BY created_at "
			if filters.SortOrder == "asc" {
				orderClause += "ASC"
			} else {
				orderClause += "DESC"
			}
		}

		limit := filters.Limit
		if limit <= 0 {
			limit = 20
		}
		offset := filters.Offset
		if offset < 0 {
			offset = 0
		}

		query := fmt.Sprintf(`
			SELECT id, org_id, requester_id, type, amount, purpose, urgency, status, resubmitted_as, submitted_at, created_at, updated_at
			FROM requests
			%s
			%s
			LIMIT $%d OFFSET $%d`, whereClause, orderClause, argID, argID+1)

		args = append(args, limit, offset)

		rows, err := r.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()

		var requests []model.Request
		for rows.Next() {
			var req model.Request
			if err := rows.Scan(&req.ID, &req.OrgID, &req.RequesterID, &req.Type, &req.Amount, &req.Purpose, &req.Urgency, &req.Status, &req.ResubmittedAs, &req.SubmittedAt, &req.CreatedAt, &req.UpdatedAt); err != nil {
				return nil, 0, err
			}
			requests = append(requests, req)
		}

		return requests, total, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var matches []model.Request
	for _, req := range r.memoryRequests {
		if req.OrgID != orgID {
			continue
		}
		if filters.Status != "" && req.Status != filters.Status {
			continue
		}
		if filters.Urgency != "" && req.Urgency != filters.Urgency {
			continue
		}
		if filters.RequesterID != "" && req.RequesterID != filters.RequesterID {
			continue
		}
		if filters.ExcludeDrafts && req.Status == "draft" {
			continue
		}
		matches = append(matches, *req)
	}

	total := len(matches)
	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := filters.Offset
	if offset < 0 {
		offset = 0
	}

	if offset >= total {
		return []model.Request{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return matches[offset:end], total, nil
}
