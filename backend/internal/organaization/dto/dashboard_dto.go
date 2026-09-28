package dto

import "time"

// EscalatedItem represents an aged or urgent request requiring attention.
type EscalatedItem struct {
	ID            string    `json:"id"`
	RequesterID   string    `json:"requester_id"`
	RequesterName string    `json:"requester_name"`
	Amount        float64   `json:"amount"`
	Urgency       string    `json:"urgency"`
	DaysPending   int       `json:"days_pending"`
	CreatedAt     time.Time `json:"created_at"`
}

// UrgencyBreakdown breaks down pending requests by priority level.
type UrgencyBreakdown struct {
	Routine  int `json:"routine"`
	Urgent   int `json:"urgent"`
	Critical int `json:"critical"`
}

// AgingBreakdown categorizes pending requests by age in days.
type AgingBreakdown struct {
	ZeroToThreeDays  int `json:"0_to_3_days"`
	ThreeToSevenDays int `json:"3_to_7_days"`
	SevenPlusDays    int `json:"7_plus_days"`
}

// DashboardSummaryResponse summarizes pending workload, urgency, aging, and escalated items.
type DashboardSummaryResponse struct {
	PendingCount     int              `json:"pending_count"`
	UrgentCount      int              `json:"urgent_count"`
	CriticalCount    int              `json:"critical_count"`
	UrgencyBreakdown UrgencyBreakdown `json:"urgency_breakdown"`
	AgingBreakdown   AgingBreakdown   `json:"aging_breakdown"`
	EscalatedItems   []EscalatedItem  `json:"escalated_items"`
}
