package dto

// MarkPaidRequest records how an approved request was paid.
type MarkPaidRequest struct {
	PaymentMethod string `json:"payment_method"`
}

// PaymentFailedRequest records why an approved payment failed.
type PaymentFailedRequest struct {
	FailureReason string `json:"failure_reason"`
}
