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
	// DAY:      Every day
	// # MONDAY:     Every Monday
	// # TUESDAY:    Every Tuesday
	// # WEDNESDAY:  Every Wednesday
	// # THURSDAY:   Every Thursday
	// # FRIDAY:     Every Friday
	// # SATURDAY:   Every Saturday
	// # SUNDAY:     Every Sunday
	// WEEK:     Every week
	// MONTH:    Every month
	// YEAR:     Every year
	Interval RecurringPeriodEnum `json:"interval"`
}

// RecurringPeriodEnum List of pre-defined periods that can be used for recurring & scheduling.
type RecurringPeriodEnum string

// CreateEntitlementJSONBodyType defines parameters for CreateEntitlement.
//
// ENUM: "metered", "static", "boolean"
type CreateEntitlementJSONBodyType string

type RecurringPeriodWithNextReset struct {
	RecurringPeriod
	NextReset time.Time `json:"nextReset"`
}
