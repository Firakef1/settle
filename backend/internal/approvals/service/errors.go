package service

import "errors"

var (
	// ErrRequestNotFound means the request is missing in this organization.
	ErrRequestNotFound = errors.New("request not found")
	// ErrApprovalNotFound means no finance decision exists for the request.
	ErrApprovalNotFound = errors.New("approval not found")
	// ErrNotPending means approve or reject was called on a request that is not pending.
	ErrNotPending = errors.New("request is not pending")
	// ErrNotApproved means payment was recorded for a request that is not approved.
	ErrNotApproved = errors.New("request is not approved")
	// ErrPaymentAlreadySettled means the payment was already marked paid or failed.
	ErrPaymentAlreadySettled = errors.New("payment is already settled")
	// ErrForbidden means the caller is not an active finance user or org admin.
	ErrForbidden = errors.New("forbidden: finance or org admin role required")
)
