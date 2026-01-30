package entitlement

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
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

func (m MeasureUsageFromInput) Equal(other MeasureUsageFromInput) bool {
	return m.ts.Equal(other.Get())
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

func (m *MeasureUsageFromInput) FromEnum(e MeasureUsageFromEnum, currPeriod timeutil.ClosedPeriod, now time.Time) error {
	if err := e.Validate(); err != nil {
		return err
	}
	switch e {
	case MeasureUsageFromCurrentPeriodStart:
		m.ts = currPeriod.From
	case MeasureUsageFromNow:
		m.ts = now
	default:
		return fmt.Errorf("unsupported enum value")
	}
	return nil
}

type CreateEntitlementInputs struct {
	Namespace        string                             `json:"namespace"`
	FeatureID        *string                            `json:"featureId"`
	FeatureKey       *string                            `json:"featureKey"`
	UsageAttribution streaming.CustomerUsageAttribution `json:"usageAttribution"`
	EntitlementType  EntitlementType                    `json:"type"`
	Metadata         map[string]string                  `json:"metadata,omitempty"`
	Annotations      models.Annotations                 `json:"annotations,omitempty"`

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
	Config                  *string                `json:"config,omitempty"`
	UsagePeriod             *UsagePeriodInput      `json:"usagePeriod,omitempty"`
	PreserveOverageAtReset  *bool                  `json:"preserveOverageAtReset,omitempty"`

	SubscriptionManaged bool `json:"subscriptionManaged,omitempty"`
}

func (c CreateEntitlementInputs) Equal(other CreateEntitlementInputs) bool {
	if c.Namespace != other.Namespace {
		return false
	}

	if !reflect.DeepEqual(c.FeatureID, other.FeatureID) {
		return false
	}

	if !reflect.DeepEqual(c.FeatureKey, other.FeatureKey) {
		return false
	}

	if !c.UsageAttribution.Equal(other.UsageAttribution) {
		return false
	}

	if c.EntitlementType != other.EntitlementType {
		return false
	}

	if !reflect.DeepEqual(c.Metadata, other.Metadata) {
		return false
	}

	if !reflect.DeepEqual(c.Annotations, other.Annotations) {
		return false
	}

	if (c.ActiveFrom == nil) != (other.ActiveFrom == nil) {
		return false
	}

	if (c.ActiveFrom != nil && other.ActiveFrom != nil) && !c.ActiveFrom.Equal(*other.ActiveFrom) {
		return false
	}

	if (c.ActiveTo == nil) != (other.ActiveTo == nil) {
		return false
	}

	if (c.ActiveTo != nil && other.ActiveTo != nil) && !c.ActiveTo.Equal(*other.ActiveTo) {
		return false
	}

	if (c.MeasureUsageFrom == nil) != (other.MeasureUsageFrom == nil) {
		return false
	}

	if (c.MeasureUsageFrom != nil && other.MeasureUsageFrom != nil) && !c.MeasureUsageFrom.Equal(*other.MeasureUsageFrom) {
		return false
	}

	if !reflect.DeepEqual(c.IssueAfterReset, other.IssueAfterReset) {
		return false
	}

	if !reflect.DeepEqual(c.IssueAfterResetPriority, other.IssueAfterResetPriority) {
		return false
	}

	if !reflect.DeepEqual(c.IsSoftLimit, other.IsSoftLimit) {
		return false
	}

	if !reflect.DeepEqual(c.Config, other.Config) {
		return false
	}

	if (c.UsagePeriod == nil) != (other.UsagePeriod == nil) {
		return false
	}

	if (c.UsagePeriod != nil && other.UsagePeriod != nil) && !c.UsagePeriod.Equal(*other.UsagePeriod) {
		return false
	}

	if !reflect.DeepEqual(c.PreserveOverageAtReset, other.PreserveOverageAtReset) {
		return false
	}

	if c.SubscriptionManaged != other.SubscriptionManaged {
		return false
	}

	return true
}

func (c CreateEntitlementInputs) Validate() error {
	if c.FeatureID == nil && c.FeatureKey == nil {
		return fmt.Errorf("feature id or key must be set")
	}

	if err := c.UsageAttribution.Validate(); err != nil {
		return err
	}

	// Let's validate the Scheduling Params
	activeFromTime := defaultx.WithDefault(c.ActiveFrom, clock.Now())

	if c.ActiveTo != nil && c.ActiveFrom == nil {
		return fmt.Errorf("ActiveFrom must be set if ActiveTo is set")
	}

	// We can allow an active period of 0 (ActiveFrom = ActiveTo)
	if c.ActiveTo != nil && c.ActiveTo.Before(activeFromTime) {
		return fmt.Errorf("ActiveTo cannot be before ActiveFrom")
	}

	// Let's validate the Usage Period
	if c.UsagePeriod != nil {
		if per, err := c.UsagePeriod.GetValue().Interval.ISODuration.Subtract(datetime.DurationHour); err == nil && per.Sign() == -1 {
			return fmt.Errorf("UsagePeriod must be at least 1 hour")
		}
	}

	if c.EntitlementType == EntitlementTypeStatic {
		if c.Config == nil {
			return fmt.Errorf("config is required for static entitlements")
		}

		if !json.Valid([]byte(*c.Config)) {
			return fmt.Errorf("invalid JSON config")
		}
	}

	return nil
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
	Config *string `json:"config,omitempty"`
}

func (e Entitlement) AsCreateEntitlementInputs(cust customer.Customer) CreateEntitlementInputs {
	i := CreateEntitlementInputs{
		Namespace:        e.Namespace,
		FeatureID:        &e.FeatureID,
		FeatureKey:       &e.FeatureKey,
		UsageAttribution: cust.GetUsageAttribution(),
		EntitlementType:  e.EntitlementType,
		Metadata:         e.Metadata,
		ActiveFrom:       e.ActiveFrom,
		ActiveTo:         e.ActiveTo,
		// MeasureUsageFrom:        &MeasureUsageFromInput{ts: e.MeasureUsageFrom},
		IssueAfterReset:         e.IssueAfterReset,
		IssueAfterResetPriority: e.IssueAfterResetPriority,
		IsSoftLimit:             e.IsSoftLimit,
		Config:                  e.Config,
		UsagePeriod:             e.UsagePeriod.GetOriginalValueAsUsagePeriodInput(),
		PreserveOverageAtReset:  e.PreserveOverageAtReset,
		Annotations:             e.Annotations,
	}

	if e.MeasureUsageFrom != nil {
		mu := &MeasureUsageFromInput{}
		// FIXME: manage error
		_ = mu.FromTime(*e.MeasureUsageFrom)
		i.MeasureUsageFrom = mu
	}

	return i
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

	if e.ActiveToTime() != nil && e.ActiveFromTime().Equal(*e.ActiveToTime()) {
		return false
	}

	return true
}

func (e Entitlement) GetType() EntitlementType {
	return e.EntitlementType
}

var _ models.CadenceComparable = Entitlement{}

func (e Entitlement) GetCadence() models.CadencedModel {
	return models.CadencedModel{
		ActiveFrom: e.ActiveFromTime(),
		ActiveTo:   e.ActiveToTime(),
	}
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
	models.MetadataModel
	models.Annotations

	// ActiveFrom allows entitlements to be scheduled for future activation.
	// If not set, the entitlement is active immediately.
	ActiveFrom *time.Time `json:"activeFrom,omitempty"`
	// ActiveTo allows entitlements to be descheduled for future activation.
	// If not set, the entitlement is active until deletion.
	ActiveTo *time.Time `json:"activeTo,omitempty"`

	ID         string `json:"id,omitempty"`
	FeatureID  string `json:"featureId,omitempty"`
	FeatureKey string `json:"featureKey,omitempty"`

	CustomerID string `json:"customerId,omitempty"`

	EntitlementType           EntitlementType        `json:"type,omitempty"`
	UsagePeriod               *UsagePeriod           `json:"usagePeriod,omitempty"`
	CurrentUsagePeriod        *timeutil.ClosedPeriod `json:"currentUsagePeriod,omitempty"`
	OriginalUsagePeriodAnchor *time.Time             `json:"originalUsagePeriodAnchor,omitempty"`
}

func (e GenericProperties) Validate() error {
	// TODO: there are no clear validation requirements now but lets implement the interface
	if e.CustomerID == "" {
		return fmt.Errorf("customer is required")
	}

	return nil
}

// ActiveFromTime returns the time the entitlement is active from. Its either the ActiveFrom field or the CreatedAt field
func (e GenericProperties) ActiveFromTime() time.Time {
	return defaultx.WithDefault(e.ActiveFrom, e.CreatedAt)
}

// ActiveToTime returns the time the entitlement is active to. Its either the ActiveTo field or the DeletedAt field or nil
func (e GenericProperties) ActiveToTime() *time.Time {
	if e.ActiveTo != nil {
		return e.ActiveTo
	}
	return e.DeletedAt
}
