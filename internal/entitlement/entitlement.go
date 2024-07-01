package entitlement

import (
	"time"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type TypedEntitlement interface {
	GetType() EntitlementType
}

type CreateEntitlementInputs struct {
	Namespace       string            `json:"namespace"`
	FeatureID       *string           `json:"featureId"`
	FeatureKey      *string           `json:"featureKey"`
	SubjectKey      string            `json:"subjectKey"`
	EntitlementType EntitlementType   `json:"type"`
	Metadata        map[string]string `json:"metadata,omitempty"`

	MeasureUsageFrom *time.Time   `json:"measureUsageFrom,omitempty"`
	IssueAfterReset  *float64     `json:"issueAfterReset,omitempty"`
	IsSoftLimit      *bool        `json:"isSoftLimit,omitempty"`
	Config           *string      `json:"config,omitempty"`
	UsagePeriod      *UsagePeriod `json:"usagePeriod,omitempty"`
}

func (c CreateEntitlementInputs) GetType() EntitlementType {
	return c.EntitlementType
}

// Normalized representation of an entitlement in the system
type Entitlement struct {
	GenericProperties

	// All none-core fields are optional
	// metered
	MeasureUsageFrom *time.Time `json:"_,omitempty"`
	IssueAfterReset  *float64   `json:"issueAfterReset,omitempty"`
	IsSoftLimit      *bool      `json:"isSoftLimit,omitempty"`
	LastReset        *time.Time `json:"lastReset,omitempty"`

	// static
	Config *string `json:"config,omitempty"`
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

	ID              string          `json:"id,omitempty"`
	FeatureID       string          `json:"featureId,omitempty"`
	FeatureKey      string          `json:"featureKey,omitempty"`
	SubjectKey      string          `json:"subjectKey,omitempty"`
	EntitlementType EntitlementType `json:"type,omitempty"`

	UsagePeriod        *UsagePeriod       `json:"usagePeriod,omitempty"`
	CurrentUsagePeriod *recurrence.Period `json:"currentUsagePeriod,omitempty"`
}

type UsagePeriod recurrence.Recurrence

func (u UsagePeriod) GetCurrentPeriod() (recurrence.Period, error) {
	rec := recurrence.Recurrence{
		Anchor:   u.Anchor,
		Interval: recurrence.RecurrenceInterval(u.Interval),
	}

	now := clock.Now()

	currentPeriodEnd, err := rec.NextAfter(now)
	if err != nil {
		return recurrence.Period{}, err
	}

	currentPeriodStart, err := rec.PrevBefore(now)
	if err != nil {
		return recurrence.Period{}, err
	}

	return recurrence.Period{
		From: currentPeriodStart,
		To:   currentPeriodEnd,
	}, nil
}
