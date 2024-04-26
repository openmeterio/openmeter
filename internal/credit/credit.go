package credit

import (
	"fmt"
	"net/http"
	"time"
)

type GrantNotFoundError struct {
	GrantID string
}

func (e *GrantNotFoundError) Error() string {
	return fmt.Sprintf("grant not found: %s", e.GrantID)
}

type ExpirationPeriodDuration string

// Defines values for ExpirationPeriodDuration.
const (
	ExpirationPeriodDurationHour  ExpirationPeriodDuration = "HOUR"
	ExpirationPeriodDurationDay   ExpirationPeriodDuration = "DAY"
	ExpirationPeriodDurationWeek  ExpirationPeriodDuration = "WEEK"
	ExpirationPeriodDurationMonth ExpirationPeriodDuration = "MONTH"
	ExpirationPeriodDurationYear  ExpirationPeriodDuration = "YEAR"
)

func (ExpirationPeriodDuration) Values() (kinds []string) {
	for _, s := range []ExpirationPeriodDuration{
		ExpirationPeriodDurationHour,
		ExpirationPeriodDurationDay,
		ExpirationPeriodDurationWeek,
		ExpirationPeriodDurationMonth,
		ExpirationPeriodDurationYear,
	} {
		kinds = append(kinds, string(s))
	}
	return
}

type EntryType string

// Defines values for EntryType.
const (
	EntryTypeGrant     EntryType = "GRANT"
	EntryTypeVoidGrant EntryType = "VOID_GRANT"
	EntryTypeReset     EntryType = "RESET"
)

func (EntryType) Values() (kinds []string) {
	for _, s := range []EntryType{
		EntryTypeGrant,
		EntryTypeVoidGrant,
		EntryTypeReset,
	} {
		kinds = append(kinds, string(s))
	}
	return
}

type GrantType string

// Defines values for GrantType.
const (
	GrantTypeUsage GrantType = "USAGE"
)

func (GrantType) Values() (kinds []string) {
	for _, s := range []GrantType{
		GrantTypeUsage,
	} {
		kinds = append(kinds, string(s))
	}
	return
}

type GrantRolloverType string

// Defines values for GrantRolloverType.
const (
	GrantRolloverTypeOriginalAmount  GrantRolloverType = "ORIGINAL_AMOUNT"
	GrantRolloverTypeRemainingAmount GrantRolloverType = "REMAINING_AMOUNT"
)

func (GrantRolloverType) Values() (kinds []string) {
	for _, s := range []GrantRolloverType{
		GrantRolloverTypeOriginalAmount,
		GrantRolloverTypeRemainingAmount,
	} {
		kinds = append(kinds, string(s))
	}
	return
}

// Reset is used to reset the balance of a specific subject.
type Reset struct {
	// ID is the readonly identifies of a reset.
	ID *string `json:"id,omitempty"`

	// Subject The subject to grant the amount to.
	Subject string `json:"subject"`

	// EffectiveAt The effective date, cannot be in the future.
	EffectiveAt time.Time `json:"effectiveAt"`
}

// Render implements the chi renderer interface.
func (c Reset) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// Grant is used to increase balance of specific subjects.
type Grant struct {
	// ID is the readonly identifies of a grant.
	ID *string `json:"id,omitempty"`

	// Parent ID is the readonly identifies of the grant's parent if any.
	ParentID *string `json:"parentID,omitempty"`

	// Subject The subject to grant the amount to.
	Subject string `json:"subject"`

	// Type The grant type.
	Type GrantType `json:"type"`

	// FeatureID The feature ID.
	FeatureID *string `json:"featureId"`

	// Amount The amount to grant. Can be positive or negative number.
	Amount float64 `json:"amount"`

	// Priority is a positive decimal numbers. With lower numbers indicating higher importance;
	// for example, a priority of 1 is more urgent than a priority of 2.
	// When there are several credit grants available for a single invoice, the system selects the credit with the highest priority.
	// In cases where credit grants share the same priority level, the grant closest to its expiration will be used first.
	// In the case of two credits have identical priorities and expiration dates, the system will use the credit that was created first.
	Priority uint8 `json:"priority"`

	// EffectiveAt The effective date.
	EffectiveAt time.Time `json:"effectiveAt"`

	// Expiration The expiration configuration.
	Expiration ExpirationPeriod  `json:"expiration"`
	Metadata   map[string]string `json:"metadata,omitempty"`

	// Rollover Grant rollover configuration.
	Rollover *GrantRollover `json:"rollover,omitempty"`

	// Void The voided date.
	Void bool `json:"void"`
}

func (c Grant) ExpirationDate() time.Time {
	return c.Expiration.GetExpiration(c.EffectiveAt)
}

// Render implements the chi renderer interface.
func (c Grant) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// GrantRollover configuration.
type GrantRollover struct {
	// MaxAmount Maximum amount to rollover.
	MaxAmount *float64 `json:"maxAmount,omitempty"`

	// Type The rollover type to use:
	Type GrantRolloverType `json:"type"`
}

// ExpirationPeriod of a credit grant.
type ExpirationPeriod struct {
	// Count The expiration period count like 12 months.
	Count uint8 `json:"count,omitempty"`

	// Duration The expiration period duration like month.
	Duration ExpirationPeriodDuration `json:"duration,omitempty"`
}

func (c ExpirationPeriod) GetExpiration(t time.Time) time.Time {
	switch c.Duration {
	case ExpirationPeriodDurationHour:
		return t.Add(time.Hour * time.Duration(c.Count))
	case ExpirationPeriodDurationDay:
		return t.AddDate(0, 0, int(c.Count))
	case ExpirationPeriodDurationWeek:
		return t.AddDate(0, 0, int(c.Count*7))
	case ExpirationPeriodDurationMonth:
		return t.AddDate(0, int(c.Count), 0)
	case ExpirationPeriodDurationYear:
		return t.AddDate(int(c.Count), 0, 0)
	default:
		return time.Time{}
	}
}

type HighWatermark struct {
	Subject string    `ch:"subject"`
	Time    time.Time `ch:"time"`
}

// HighWatermarBeforeError is returned when a lock cannot be obtained.
type HighWatermarBeforeError struct {
	Namespace     string
	Subject       string
	HighWatermark time.Time
}

func (e *HighWatermarBeforeError) Error() string {
	return fmt.Sprintf("ledger action for subject %s must be after highwatermark: %s", e.Subject, e.HighWatermark.Format(time.RFC3339))
}

// LockErrNotObtainedError is returned when a lock cannot be obtained.
type LockErrNotObtainedError struct {
	Namespace string
	Subject   string
}

func (e *LockErrNotObtainedError) Error() string {
	return fmt.Sprintf("lock not obtained for namespace %s, subject %s", e.Namespace, e.Subject)
}
