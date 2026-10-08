package model

import "time"

// UserIdentity links a user to an external sign-in provider account.
type UserIdentity struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Provider  string    `json:"provider"` // "google", "microsoft"
	Subject   string    `json:"subject"`  // the provider's stable user ID
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}
