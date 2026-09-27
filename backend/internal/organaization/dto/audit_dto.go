package dto

import "time"

// AuditFilters holds query filtering and pagination options for audit logs.
type AuditFilters struct {
	Action    string     `form:"action"`
	ActorID   string     `form:"actor_id"`
	StartDate *time.Time `form:"start_date"`
	EndDate   *time.Time `form:"end_date"`
	Page      int        `form:"page"`
	Limit     int        `form:"limit"`
	Offset    int        `form:"offset"`
}

// AuditLogItem represents a sanitized, decorated audit log entry returned to clients.
type AuditLogItem struct {
	ID        string         `json:"id"`
	OrgID     string         `json:"org_id"`
	ActorID   string         `json:"actor_id"`
	ActorName string         `json:"actor_name,omitempty"`
	Action    string         `json:"action"`
	TargetID  *string        `json:"target_id,omitempty"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
}

// AuditPagination contains pagination metadata.
type AuditPagination struct {
	Total   int  `json:"total"`
	Page    int  `json:"page"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"has_more"`
}

// AuditLogListResponse wraps the paginated audit logs list.
type AuditLogListResponse struct {
	Data       []*AuditLogItem `json:"data"`
	Pagination AuditPagination `json:"pagination"`
}
