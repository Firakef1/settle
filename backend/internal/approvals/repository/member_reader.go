package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type pgMemberReader struct{}

// NewMemberReader returns a SQL reader for active organization roles.
func NewMemberReader() MemberReader {
	return &pgMemberReader{}
}

func (r *pgMemberReader) ActiveRole(ctx context.Context, db DBTX, orgID, userID string) (string, error) {
	if db == nil {
		return "", errors.New("database executor is nil")
	}
	const query = `
		SELECT role
		FROM org_members
		WHERE org_id = $1 AND user_id = $2 AND status = 'active'
	`
	var role string
	err := db.QueryRowContext(ctx, query, orgID, userID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get member role: %w", err)
	}
	return role, nil
}
