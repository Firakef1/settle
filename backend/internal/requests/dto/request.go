package dto

import "time"

type CreateRequestDTO struct {
	Type    string  `json:"type"`
	Amount  float64 `json:"amount"`
	Purpose string  `json:"purpose"`
	Urgency string  `json:"urgency"`
}

type ResubmitRequestDTO struct {
	Amount      *float64 `json:"amount,omitempty"`
	Purpose     *string  `json:"purpose,omitempty"`
	Urgency     *string  `json:"urgency,omitempty"`
	ReceiptMode *string  `json:"receipt_mode,omitempty"`
}

type AddCommentDTO struct {
	Content string `json:"content"`
}

type RequesterResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type ReceiptResponse struct {
	ID         string    `json:"id"`
	FilePath   string    `json:"file_path"`
	OCRStatus  string    `json:"ocr_status"`
	OCRResults *string   `json:"ocr_results,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type CommentResponse struct {
	ID        string    `json:"id"`
	AuthorID  string    `json:"author_id"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type ApprovalResponse struct {
	// Simple stub for approval domain interactions if needed later
	Status string `json:"status"`
}

type TimelineEvent struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Actor     string    `json:"actor"`
	Details   string    `json:"details,omitempty"`
}

type RequestResponse struct {
	ID      string  `json:"id"`
	Type    string  `json:"type"`
	Amount  float64 `json:"amount"`
	Purpose string  `json:"purpose"`
	Urgency string  `json:"urgency"`
	Status  string  `json:"status"`

	Requester RequesterResponse `json:"requester"`

	Receipts []ReceiptResponse `json:"receipts"`
	Approval *ApprovalResponse `json:"approval,omitempty"`

	Comments []CommentResponse `json:"comments"`
	Timeline []TimelineEvent   `json:"timeline"`

	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type RequestListItem struct {
	ID          string            `json:"id"`
	Amount      float64           `json:"amount"`
	Purpose     string            `json:"purpose"`
	Urgency     string            `json:"urgency"`
	Status      string            `json:"status"`
	Requester   RequesterResponse `json:"requester"`
	DaysPending int               `json:"days_pending"`
	CreatedAt   time.Time         `json:"created_at"`
}

type PaginationMeta struct {
	Total   int  `json:"total"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"has_more"`
}

type RequestListResponse struct {
	Data []RequestListItem `json:"data"`
	Meta PaginationMeta    `json:"meta"`
}

type PreviousRequestsResponse struct {
	Data []RequestListItem `json:"data"`
}

type ListFilters struct {
	Status      string
	Urgency     string
	RequesterID string

	ExcludeDrafts bool

	SortBy    string
	SortOrder string

	Limit  int
	Offset int
}
