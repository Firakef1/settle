package validator

import (
	"errors"
	"strings"

	"github.com/Firakef1/settle/backend/internal/requests/dto"
)

var (
	ErrInvalidAmount  = errors.New("amount must be greater than 0 and less than or equal to 999999.99")
	ErrInvalidPurpose = errors.New("purpose must be between 10 and 500 characters")
	ErrInvalidType    = errors.New("type must be one of: reimbursement, advance, stipend")
	ErrInvalidUrgency = errors.New("urgency must be one of: routine, urgent, critical")
	ErrInvalidMode    = errors.New("receipt_mode must be one of: carry, new")
	ErrEmptyContent   = errors.New("comment content cannot be empty")
)

func ValidateCreateRequest(req *dto.CreateRequestDTO) error {
	if req.Amount <= 0 || req.Amount > 999999.99 {
		return ErrInvalidAmount
	}

	purpose := strings.TrimSpace(req.Purpose)
	if len(purpose) < 10 || len(purpose) > 500 {
		return ErrInvalidPurpose
	}

	validTypes := map[string]bool{
		"reimbursement": true,
		"advance":       true,
		"stipend":       true,
	}
	if !validTypes[req.Type] {
		return ErrInvalidType
	}

	validUrgencies := map[string]bool{
		"routine":  true,
		"urgent":   true,
		"critical": true,
	}
	if !validUrgencies[req.Urgency] {
		return ErrInvalidUrgency
	}

	return nil
}

func ValidateResubmitRequest(req *dto.ResubmitRequestDTO) error {
	if req.Amount != nil {
		if *req.Amount <= 0 || *req.Amount > 999999.99 {
			return ErrInvalidAmount
		}
	}
	if req.Purpose != nil {
		purpose := strings.TrimSpace(*req.Purpose)
		if len(purpose) < 10 || len(purpose) > 500 {
			return ErrInvalidPurpose
		}
	}
	if req.Urgency != nil {
		validUrgencies := map[string]bool{
			"routine":  true,
			"urgent":   true,
			"critical": true,
		}
		if !validUrgencies[*req.Urgency] {
			return ErrInvalidUrgency
		}
	}
	if req.ReceiptMode != nil {
		validModes := map[string]bool{
			"carry": true,
			"new":   true,
		}
		if !validModes[*req.ReceiptMode] {
			return ErrInvalidMode
		}
	}
	return nil
}

func ValidateAddComment(req *dto.AddCommentDTO) error {
	if strings.TrimSpace(req.Content) == "" {
		return ErrEmptyContent
	}
	return nil
}
