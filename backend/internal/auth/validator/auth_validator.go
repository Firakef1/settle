package validator

import (
	"errors"
	"regexp"
	"strings"

	"github.com/Firakef1/settle/backend/internal/auth/dto"
)

var (
	ErrInvalidEmail     = errors.New("invalid email format")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters long")
	ErrNameRequired     = errors.New("name is required")
	ErrEmailRequired    = errors.New("email is required")
	ErrPasswordRequired = errors.New("password is required")

	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// ValidateEmail checks if email matches valid format.
func ValidateEmail(email string) bool {
	return emailRegex.MatchString(strings.TrimSpace(email))
}

// ValidateLoginRequest validates email and password presence.
func ValidateLoginRequest(req *dto.LoginRequest) error {
	if strings.TrimSpace(req.Email) == "" {
		return ErrEmailRequired
	}
	if !ValidateEmail(req.Email) {
		return ErrInvalidEmail
	}
	if req.Password == "" {
		return ErrPasswordRequired
	}
	return nil
}

// ValidateSignupRequest validates email, password min length, and name.
func ValidateSignupRequest(req *dto.SignupRequest) error {
	if strings.TrimSpace(req.Email) == "" {
		return ErrEmailRequired
	}
	if !ValidateEmail(req.Email) {
		return ErrInvalidEmail
	}
	if strings.TrimSpace(req.Name) == "" {
		return ErrNameRequired
	}
	if len(req.Password) < 8 {
		return ErrPasswordTooShort
	}
	return nil
}
