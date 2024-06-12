package entitlement

import "time"

type EntitlementBalance struct {
	EntitlementID EntitlementID `json:"entitlement_id"`
	Balance       float64       `json:"balance"`
	UsageInPeriod float64       `json:"usage_in_period"`
	Overage       float64       `json:"overage"`
	PeriodStart   string        `json:"period_start"`
}

type EntitlementBalanceHistoryWindow struct {
	From           time.Time
	To             time.Time
	UsageInPeriod  float64
	BalanceAtStart float64
	BalanceAtEnd   float64
	Overage        float64
}

type EntitlementGrantID string
