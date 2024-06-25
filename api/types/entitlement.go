package types

import "time"

type RecurringPeriod struct {
	// Anchor An arbitrary anchor to base the recurring period on.
	Anchor time.Time `json:"anchor"`

	// Interval List of pre-defined periods that can be used for recurring & scheduling.
	//
	// DAY:      Every day
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

const (
	CreateEntitlementJSONBodyTypeMetered CreateEntitlementJSONBodyType = "metered"
	CreateEntitlementJSONBodyTypeStatic  CreateEntitlementJSONBodyType = "static"
	CreateEntitlementJSONBodyTypeBoolean CreateEntitlementJSONBodyType = "boolean"
)
