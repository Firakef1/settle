package dto

// MemberCounts represents the breakdown of organization members by role.
type MemberCounts struct {
	Total    int `json:"total"`
	OrgAdmin int `json:"org_admin"`
	Finance  int `json:"finance"`
	Staff    int `json:"staff"`
}

// RequestStats represents the breakdown of payout requests by lifecycle status.
type RequestStats struct {
	Total     int `json:"total"`
	Pending   int `json:"pending"`
	Approved  int `json:"approved"`
	Paid      int `json:"paid"`
	Rejected  int `json:"rejected"`
	Failed    int `json:"failed"`
	Withdrawn int `json:"withdrawn"`
}

// FinancialStats represents total spending and current month spending.
type FinancialStats struct {
	TotalSpent     float64 `json:"total_spent"`
	ThisMonthSpent float64 `json:"this_month_spent"`
	Currency       string  `json:"currency"`
}

// PlanUsageStats represents current usage against plan limits.
type PlanUsageStats struct {
	CurrentPlan       string `json:"current_plan"`
	RequestsThisMonth int    `json:"requests_this_month"`
	RequestLimit      int    `json:"request_limit"` // 0 = unlimited
	UserCount         int    `json:"user_count"`
	UserLimit         int    `json:"user_limit"` // 0 = unlimited
}

// OrgStatsResponse wraps the complete organization metrics.
type OrgStatsResponse struct {
	OrgID        string         `json:"org_id"`
	MemberCounts MemberCounts   `json:"member_counts"`
	RequestStats RequestStats   `json:"request_stats"`
	Financials   FinancialStats `json:"financials"`
	PlanUsage    PlanUsageStats `json:"plan_usage"`
}
