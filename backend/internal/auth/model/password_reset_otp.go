package model

import "time"

// PasswordResetOTP represents a single-use OTP for password reset.
type PasswordResetOTP struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	OTPHash      string    `json:"-"`
	ExpiresAt    time.Time `json:"expires_at"`
	AttemptCount int       `json:"attempt_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
