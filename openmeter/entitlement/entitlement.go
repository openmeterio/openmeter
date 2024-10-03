package entitlement

import (
	"fmt"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type TypedEntitlement interface {
	GetType() EntitlementType
}

type MeasureUsageFromEnum string

const (
	MeasureUsageFromCurrentPeriodStart MeasureUsageFromEnum = "CURRENT_PERIOD_START"
	MeasureUsageFromNow                MeasureUsageFromEnum = "NOW"
)

func (e MeasureUsageFromEnum) Values() []MeasureUsageFromEnum {
	return []MeasureUsageFromEnum{MeasureUsageFromCurrentPeriodStart, MeasureUsageFromNow}
}

func (e MeasureUsageFromEnum) Validate() error {
	if !slices.Contains(e.Values(), e) {
		return fmt.Errorf("invalid value")
	}
	return nil
}

type MeasureUsageFromInput struct {
	ts time.Time
}

func (m MeasureUsageFromInput) Get() time.Time {
	return m.ts
}

func (m *MeasureUsageFromInput) FromTime(t time.Time) error {
	if t.IsZero() {
		return fmt.Errorf("time is zero")
	}

	m.ts = t
	return nil
}

func (m *MeasureUsageFromInput) FromEnum(e MeasureUsageFromEnum, p UsagePeriod, t time.Time) error {
	if err := e.Validate(); err != nil {
		return err
	}
	switch e {
	case MeasureUsageFromCurrentPeriodStart:
		period, err := p.GetCurrentPeriodAt(clock.Now())
		if err != nil {
			return err
		}
		m.ts = period.From
	case MeasureUsageFromNow:
		m.ts = t
	default:
		return fmt.Errorf("unsupported enum value")
	}
	return nil
}

type CreateEntitlementInputs struct {
	Namespace       string            `json:"namespace"`
	FeatureID       *string           `json:"featureId"`
	FeatureKey      *string           `json:"featureKey"`
	SubjectKey      string            `json:"subjectKey"`
	EntitlementType EntitlementType   `json:"type"`
	Metadata        map[string]string `json:"metadata,omitempty"`

	// ActiveFrom allows entitlements to be scheduled for future activation.
	// If not set, the entitlement is active immediately.
	ActiveFrom *time.Time `json:"activeFrom,omitempty"`
	// ActiveTo allows entitlements to be descheduled for future activation.
	// If not set, the entitlement is active until deletion.
	ActiveTo *time.Time `json:"activeTo,omitempty"`

	MeasureUsageFrom        *MeasureUsageFromInput `json:"measureUsageFrom,omitempty"`
	IssueAfterReset         *float64               `json:"issueAfterReset,omitempty"`
	IssueAfterResetPriority *uint8                 `json:"issueAfterResetPriority,omitempty"`
	IsSoftLimit             *bool                  `json:"isSoftLimit,omitempty"`
	Config                  []byte                 `json:"config,omitempty"`
	UsagePeriod             *UsagePeriod           `json:"usagePeriod,omitempty"`
	PreserveOverageAtReset  *bool                  `json:"preserveOverageAtReset,omitempty"`
}

func (c CreateEntitlementInputs) GetType() EntitlementType {
	return c.EntitlementType
}

// Normalized representation of an entitlement in the system
type Entitlement struct {
	GenericProperties

	// All none-core fields are optional
	// metered
	MeasureUsageFrom        *time.Time `json:"measureUsageFrom,omitempty"`
	IssueAfterReset         *float64   `json:"issueAfterReset,omitempty"`
	IssueAfterResetPriority *uint8     `json:"issueAfterResetPriority,omitempty"`
	IsSoftLimit             *bool      `json:"isSoftLimit,omitempty"`
	LastReset               *time.Time `json:"lastReset,omitempty"`
	PreserveOverageAtReset  *bool      `json:"preserveOverageAtReset,omitempty"`

	// static
	Config []byte `json:"config,omitempty"`
}

// ActiveFromTime returns the time the entitlement is active from. Its either the ActiveFrom field or the CreatedAt field
func (e Entitlement) ActiveFromTime() time.Time {
	return defaultx.WithDefault(e.ActiveFrom, e.CreatedAt)
}

// ActiveToTime returns the time the entitlement is active to. Its either the ActiveTo field or the DeletedAt field or nil
func (e Entitlement) ActiveToTime() *time.Time {
	if e.ActiveTo != nil {
		return e.ActiveTo
	}
	return e.DeletedAt
}

// IsActive returns if the entitlement is active at the given time
// Period start is determined by
func (e Entitlement) IsActive(at time.Time) bool {
	if e.DeletedAt != nil && !at.Before(*e.DeletedAt) {
		return false
	}

	if e.ActiveFromTime().After(at) {
		return false
	}

	if e.ActiveTo != nil && !at.Before(*e.ActiveTo) {
		return false
	}

	return true
}

func (e Entitlement) GetType() EntitlementType {
	return e.EntitlementType
}

type EntitlementType string

const (
	// EntitlementTypeMetered represents entitlements where access is determined by usage and balance calculations
	EntitlementTypeMetered EntitlementType = "metered"
	// EntitlementTypeStatic represents entitlements where access is described by a static configuration
	EntitlementTypeStatic EntitlementType = "static"
	// EntitlementTypeBoolean represents boolean access
	EntitlementTypeBoolean EntitlementType = "boolean"
)

func (e EntitlementType) Values() []EntitlementType {
	return []EntitlementType{EntitlementTypeMetered, EntitlementTypeStatic, EntitlementTypeBoolean}
}

func (e EntitlementType) StrValues() []string {
	return slicesx.Map(e.Values(), func(i EntitlementType) string {
		return string(i)
	})
}

func (e EntitlementType) String() string {
	return string(e)
}

// GenericProperties is the core fields of an entitlement that are always applicable regadless of type
type GenericProperties struct {
	models.NamespacedModel
	models.ManagedModel

	Metadata map[string]string `json:"metadata,omitempty"`

	// ActiveFrom allows entitlements to be scheduled for future activation.
	// If not set, the entitlement is active immediately.
	ActiveFrom *time.Time `json:"activeFrom,omitempty"`
	// ActiveTo allows entitlements to be descheduled for future activation.
	// If not set, the entitlement is active until deletion.
	ActiveTo *time.Time `json:"activeTo,omitempty"`

	ID              string          `json:"id,omitempty"`
	FeatureID       string          `json:"featureId,omitempty"`
	FeatureKey      string          `json:"featureKey,omitempty"`
	SubjectKey      string          `json:"subjectKey,omitempty"`
	EntitlementType EntitlementType `json:"type,omitempty"`

	UsagePeriod        *UsagePeriod       `json:"usagePeriod,omitempty"`
	CurrentUsagePeriod *recurrence.Period `json:"currentUsagePeriod,omitempty"`
}

type UsagePeriod recurrence.Recurrence

// The returned period is exclusive at the end end inclusive in the start
func (u UsagePeriod) GetCurrentPeriodAt(at time.Time) (recurrence.Period, error) {
	rec := recurrence.Recurrence{
		Anchor:   u.Anchor,
		Interval: u.Interval,
	}

	nextAfter, err := rec.NextAfter(at)
	if err != nil {
		return recurrence.Period{}, err
	}

	// The edgecase behavior of recurrence.Period doesn't work for us here
	// as for usage periods we want to have the period end exclusive
	if nextAfter.Equal(at) {
		from := nextAfter
		to, err := rec.Next(from)
		if err != nil {
			return recurrence.Period{}, err
		}
		return recurrence.Period{
			From: from,
			To:   to,
		}, nil
	}

	prevBefore, err := rec.PrevBefore(at)
	if err != nil {
		return recurrence.Period{}, err
	}

	return recurrence.Period{
		From: prevBefore,
		To:   nextAfter,
	}, nil
}
