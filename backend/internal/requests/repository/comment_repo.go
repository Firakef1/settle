package repository

import (
	"context"
	"database/sql"
	"sync"

	"github.com/Firakef1/settle/backend/internal/requests/model"
)

type CommentRepository interface {
	CreateComment(ctx context.Context, comment *model.Comment) error
	GetByRequestID(ctx context.Context, requestID string) ([]model.Comment, error)
}

type CommentRepo struct {
	db             *sql.DB
	memoryComments map[string]*model.Comment // id -> comment
	mu             sync.RWMutex
}

func NewCommentRepo(db *sql.DB) *CommentRepo {
	return &CommentRepo{
		db:             db,
		memoryComments: make(map[string]*model.Comment),
	}
}

func (r *CommentRepo) CreateComment(ctx context.Context, comment *model.Comment) error {
	if r.db != nil {
		query := `
			INSERT INTO comments (id, request_id, author_id, content, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)`
		_, err := r.db.ExecContext(ctx, query,
			comment.ID,
			comment.RequestID,
			comment.AuthorID,
			comment.Content,
			comment.CreatedAt,
			comment.UpdatedAt,
		)
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	c := *comment
	r.memoryComments[comment.ID] = &c
	return nil
}

func (r *CommentRepo) GetByRequestID(ctx context.Context, requestID string) ([]model.Comment, error) {
	if r.db != nil {
		query := `
			SELECT c.id, c.request_id, c.author_id, c.content, c.created_at, c.updated_at,
			       u.full_name as author, om.role
			FROM comments c
			JOIN users u ON c.author_id = u.id
			JOIN requests req ON req.id = c.request_id
			JOIN org_members om ON om.user_id = u.id AND om.org_id = req.org_id
			WHERE c.request_id = $1
			ORDER BY c.created_at ASC`
		rows, err := r.db.QueryContext(ctx, query, requestID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var comments []model.Comment
		for rows.Next() {
			var c model.Comment
			var author, role string
			if err := rows.Scan(&c.ID, &c.RequestID, &c.AuthorID, &c.Content, &c.CreatedAt, &c.UpdatedAt, &author, &role); err != nil {
				return nil, err
			}
			c.AuthorID = c.AuthorID
			c.Author = author
			c.Role = role
			c.Content = c.Content
			c.CreatedAt = c.CreatedAt
			c.UpdatedAt = c.UpdatedAt
			comments = append(comments, c)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return comments, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	var comments []model.Comment
	for _, c := range r.memoryComments {
		if c.RequestID == requestID {
			comments = append(comments, *c)
		}
	}
	return comments, nil
}
