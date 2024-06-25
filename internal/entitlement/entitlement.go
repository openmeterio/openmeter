package entitlement

import (
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type TypedEntitlement interface {
	GetType() EntitlementType
}

type CreateEntitlementInputs struct {
	Namespace       string          `json:"namespace"`
	FeatureID       *string         `json:"featureId,omitempty"`
	FeatureKey      *string         `json:"featureKey,omitempty"`
	SubjectKey      string          `json:"subjectKey"`
	EntitlementType EntitlementType `json:"type"`

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

	ID              string          `json:"id,omitempty"`
	FeatureID       string          `json:"featureId,omitempty"`
	FeatureKey      string          `json:"featureKey,omitempty"`
	SubjectKey      string          `json:"subjectKey,omitempty"`
	EntitlementType EntitlementType `json:"type,omitempty"`

	UsagePeriod *UsagePeriod `json:"usagePeriod,omitempty"`
}

type UsagePeriod struct {
	Anchor   time.Time           `json:"anchor"`
	Interval UsagePeriodInterval `json:"interval"`
}

type UsagePeriodInterval string

const (
	UsagePeriodIntervalDay   UsagePeriodInterval = "DAY"
	UsagePeriodIntervalWeek  UsagePeriodInterval = "WEEK"
	UsagePeriodIntervalMonth UsagePeriodInterval = "MONTH"
	UsagePeriodIntervalYear  UsagePeriodInterval = "YEAR"
)

func (u UsagePeriodInterval) Values() []UsagePeriodInterval {
	return []UsagePeriodInterval{UsagePeriodIntervalDay, UsagePeriodIntervalWeek, UsagePeriodIntervalMonth, UsagePeriodIntervalYear}
}

func (u UsagePeriodInterval) StrValues() []string {
	return slicesx.Map(u.Values(), func(i UsagePeriodInterval) string {
		return string(i)
	})
}
