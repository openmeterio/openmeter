package types

import "time"

type CreateEntitlementJSONBody struct {
	// Config The JSON parsable config of the entitlement.
	Config *string `json:"config"`

	// FeatureId The feature the subject is entitled to use
	FeatureId string `json:"featureId"`

	// IsUnlimited If unlimited=true the subject can use the feature an unlimited amount.
	IsUnlimited *bool `json:"isUnlimited,omitempty"`

	// IssueAfterReset You can issue usage automatically after reset. This usage is not rolled over.
	IssueAfterReset *float64 `json:"issueAfterReset,omitempty"`

	// Metadata Additional metadata for the feature.
	Metadata *map[string]string            `json:"metadata,omitempty"`
	Type     CreateEntitlementJSONBodyType `json:"type"`

	// UsagePeriod Recurring period of an entitlement.
	UsagePeriod RecurringPeriod `json:"usagePeriod"`
}

type RecurringPeriod struct {
	// Anchor An arbitrary anchor to base the recurring period on.
	Anchor time.Time `json:"anchor"`

	// Interval List of pre-defined periods that can be used for recurring & scheduling.
	//
	// DAILY:      Every day
	// # MONDAY:     Every Monday
	// # TUESDAY:    Every Tuesday
	// # WEDNESDAY:  Every Wednesday
	// # THURSDAY:   Every Thursday
	// # FRIDAY:     Every Friday
	// # SATURDAY:   Every Saturday
	// # SUNDAY:     Every Sunday
	// WEEKLY:     Every week
	// MONTHLY:    Every month
	// YEARLY:     Every year
	// BILLING:    Every billing cycle
	Interval RecurringPeriodEnum `json:"interval"`
}

// RecurringPeriodEnum List of pre-defined periods that can be used for recurring & scheduling.
//
// DAILY:      Every day
// # MONDAY:     Every Monday
// # TUESDAY:    Every Tuesday
// # WEDNESDAY:  Every Wednesday
// # THURSDAY:   Every Thursday
// # FRIDAY:     Every Friday
// # SATURDAY:   Every Saturday
// # SUNDAY:     Every Sunday
// WEEKLY:     Every week
// MONTHLY:    Every month
// YEARLY:     Every year
// BILLING:    Every billing cycle
type RecurringPeriodEnum string

// CreateEntitlementJSONBodyType defines parameters for CreateEntitlement.
//
// ENUM: "metered", "static", "boolean"
type CreateEntitlementJSONBodyType string
