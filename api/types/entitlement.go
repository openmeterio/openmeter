package types

import (
	"errors"
	"time"
)

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
	UsagePeriod RecurringPeriodCreateInputs `json:"usagePeriod"`
}

type RecurringPeriodCreateInputs struct {
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

type RecurringPeriod struct {
	RecurringPeriodCreateInputs

	// NextRecurrence contains the next recurrance of the period.
	NextRecurrence time.Time `json:"nextRecurrence"`
}

func (p *RecurringPeriodCreateInputs) NextRecurrence() (time.Time, error) {
	switch p.Interval {
	case RecurringPeriodEnum("DAILY"):
		return p.Anchor.AddDate(0, 0, 1), nil
	case RecurringPeriodEnum("WEEKLY"):
		return p.Anchor.AddDate(0, 0, 7), nil
	case RecurringPeriodEnum("MONTHLY"):
		return p.Anchor.AddDate(0, 1, 0), nil
	case RecurringPeriodEnum("YEARLY"):
		return p.Anchor.AddDate(1, 0, 0), nil
	case RecurringPeriodEnum("BILLING"):
		// TODO: Implement billing period handling
		return time.Time{}, errors.New("not implemented")
	}

	return time.Time{}, errors.New("invalid interval")
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

const RecurringPeriodBilling RecurringPeriodEnum = "BILLING"

// CreateEntitlementJSONBodyType defines parameters for CreateEntitlement.
//
// ENUM: "metered", "static", "boolean"
type CreateEntitlementJSONBodyType string
