package entitlement

import (
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type HasType interface {
	GetType() EntitlementType
}

type CreateEntitlementInputs struct {
	Namespace       string
	FeatureID       string
	SubjectKey      string
	EntitlementType EntitlementType

	MeasureUsageFrom *time.Time
	IssueAfterReset  *float64
	IsSoftLimit      *bool
	Config           *string
}

func (c CreateEntitlementInputs) GetType() EntitlementType {
	return c.EntitlementType
}

// Normalized representation of an entitlement in the system
type Entitlement struct {
	GenericProperties

	// All none-core fields are optional

	// metered
	MeasureUsageFrom *time.Time
	IssueAfterReset  *float64
	IsSoftLimit      *bool

	// static
	Config *string
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

func (e EntitlementType) String() string {
	return string(e)
}

// GenericProperties is the core fields of an entitlement that are always applicable regadless of type
type GenericProperties struct {
	models.NamespacedModel
	models.ManagedModel

	ID              string          `json:"id,omitempty"`
	FeatureID       string          `json:"featureId,omitempty"`
	SubjectKey      string          `json:"subjectKey,omitempty"`
	EntitlementType EntitlementType `json:"type,omitempty"`
}
