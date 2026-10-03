package model

import "time"

type Comment struct {
	ID        string    `json:"id" db:"id"`
	RequestID string    `json:"request_id" db:"request_id"`
	AuthorID  string    `json:"author_id" db:"author_id"`
	Author    string    `json:"author" db:"author"`
	Role      string    `json:"role" db:"role"`
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
