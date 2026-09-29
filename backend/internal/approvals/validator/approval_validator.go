package validator

import (
	"errors"
	"strings"
)

var (
	// ErrReasonRequired is returned when a rejection or failure reason is blank.
	ErrReasonRequired = errors.New("reason is required")
	// ErrInvalidPaymentMethod is returned when the method is outside the allowed set.
	ErrInvalidPaymentMethod = errors.New("invalid payment method: must be one of bank_transfer, check, cash, other")
)

var allowedPaymentMethods = map[string]struct{}{
	"bank_transfer": {},
	"check":         {},
	"cash":          {},
	"other":         {},
}

// ValidateReason requires a non-empty rejection or failure reason.
func ValidateReason(reason string) error {
	if strings.TrimSpace(reason) == "" {
		return ErrReasonRequired
	}
	return nil
}

// NormalizePaymentMethod returns the canonical method or ErrInvalidPaymentMethod.
func NormalizePaymentMethod(method string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(method))
	if _, ok := allowedPaymentMethods[normalized]; !ok {
		return "", ErrInvalidPaymentMethod
	}
	return normalized, nil
}
