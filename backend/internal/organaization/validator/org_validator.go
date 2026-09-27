package validator

import (
	"errors"
	"net/mail"
	"strings"
)

var (
	ErrEmptyOrgName     = errors.New("organization name cannot be empty")
	ErrOrgNameTooLong   = errors.New("organization name exceeds maximum length of 255 characters")
	ErrInvalidCurrency  = errors.New("invalid currency code: must be one of USD, EUR, GBP")
	ErrInvalidRole      = errors.New("invalid role: must be either staff or finance")
	ErrInvalidEmail     = errors.New("invalid email address format")
	ErrPasswordTooShort = errors.New("password must be at least 6 characters")
)

var allowedCurrencies = map[string]bool{
	"USD": true,
	"EUR": true,
	"GBP": true,
}

var allowedInviteRoles = map[string]bool{
	"staff":   true,
	"finance": true,
}

// ValidateOrgName checks if the organization name is valid.
func ValidateOrgName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ErrEmptyOrgName
	}
	if len(trimmed) > 255 {
		return ErrOrgNameTooLong
	}
	return nil
}

// ValidateCurrency checks if the currency is one of USD, EUR, GBP.
func ValidateCurrency(currency string) error {
	upper := strings.ToUpper(strings.TrimSpace(currency))
	if !allowedCurrencies[upper] {
		return ErrInvalidCurrency
	}
	return nil
}

// ValidateInviteRole checks if the role is staff or finance.
func ValidateInviteRole(role string) error {
	lower := strings.ToLower(strings.TrimSpace(role))
	if !allowedInviteRoles[lower] {
		return ErrInvalidRole
	}
	return nil
}

// ValidateEmail checks if the email address is valid.
func ValidateEmail(email string) error {
	trimmed := strings.TrimSpace(email)
	if trimmed == "" {
		return ErrInvalidEmail
	}
	_, err := mail.ParseAddress(trimmed)
	if err != nil {
		return ErrInvalidEmail
	}
	return nil
}

// ValidatePassword checks password requirements for new registration.
func ValidatePassword(password string) error {
	if len(password) < 6 {
		return ErrPasswordTooShort
	}
	return nil
}
