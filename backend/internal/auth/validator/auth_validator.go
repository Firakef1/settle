package validator

import (
	"errors"
	"regexp"
	"strings"

	"github.com/Firakef1/settle/backend/internal/auth/dto"
)

var (
	ErrInvalidEmail            = errors.New("invalid email format")
	ErrPasswordTooShort        = errors.New("password must be at least 8 characters long")
	ErrNameRequired            = errors.New("name is required")
	ErrEmailRequired           = errors.New("email is required")
	ErrPasswordRequired        = errors.New("password is required")
	ErrOTPRequired             = errors.New("otp is required")
	ErrInvalidOTP              = errors.New("otp must be 6 numeric digits")
	ErrCurrentPasswordRequired = errors.New("current password is required")

	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	otpRegex   = regexp.MustCompile(`^[0-9]{6}$`)
)

// ValidateEmail checks if email matches valid format.
func ValidateEmail(email string) bool {
	return emailRegex.MatchString(strings.TrimSpace(email))
}

// ValidateOTPFormat checks if OTP is exactly 6 digits.
func ValidateOTPFormat(otp string) bool {
	return otpRegex.MatchString(strings.TrimSpace(otp))
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

// ValidateForgotPasswordRequest validates forgot password request payload.
func ValidateForgotPasswordRequest(req *dto.ForgotPasswordRequest) error {
	if req == nil || strings.TrimSpace(req.Email) == "" {
		return ErrEmailRequired
	}
	if !ValidateEmail(req.Email) {
		return ErrInvalidEmail
	}
	return nil
}

// ValidateResetPasswordRequest validates reset password request payload.
func ValidateResetPasswordRequest(req *dto.ResetPasswordRequest) error {
	if req == nil || strings.TrimSpace(req.Email) == "" {
		return ErrEmailRequired
	}
	if !ValidateEmail(req.Email) {
		return ErrInvalidEmail
	}
	if strings.TrimSpace(req.OTP) == "" {
		return ErrOTPRequired
	}
	if !ValidateOTPFormat(req.OTP) {
		return ErrInvalidOTP
	}
	if len(req.NewPassword) < 8 {
		return ErrPasswordTooShort
	}
	return nil
}

// ValidateDeleteAccountRequest validates account deletion request payload.
func ValidateDeleteAccountRequest(req *dto.DeleteAccountRequest) error {
	if req == nil || req.CurrentPassword == "" {
		return ErrCurrentPasswordRequired
	}
	return nil
}
