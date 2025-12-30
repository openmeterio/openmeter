package staticentitlement

import (
	"context"
	"encoding/json"
	"time"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type Connector interface {
	entitlement.SubTypeConnector
}

type connector struct {
	granularity time.Duration
}

func NewStaticEntitlementConnector() Connector {
	return &connector{
		granularity: time.Minute,
	}
}

func (c *connector) GetValue(ctx context.Context, entitlement *entitlement.Entitlement, at time.Time) (entitlement.EntitlementValue, error) {
	static, err := ParseFromGenericEntitlement(entitlement)
	if err != nil {
		return nil, err
	}

	return &StaticEntitlementValue{
		Config: static.Config,
	}, nil
}

func (c *connector) BeforeCreate(model entitlement.CreateEntitlementInputs, feature feature.Feature) (*entitlement.CreateEntitlementRepoInputs, error) {
	model.EntitlementType = entitlement.EntitlementTypeStatic

	if model.MeasureUsageFrom != nil ||
		model.IssueAfterReset != nil ||
		model.IsSoftLimit != nil {
		return nil, &entitlement.InvalidValueError{Type: model.EntitlementType, Message: "Invalid inputs for type"}
	}

	// validate that config is JSON parseable
	if model.Config == nil {
		return nil, &entitlement.InvalidValueError{Type: model.EntitlementType, Message: "Config is required"}
	}

	if !json.Valid([]byte(*model.Config)) {
		return nil, &entitlement.InvalidValueError{Type: model.EntitlementType, Message: "Config is not valid JSON"}
	}

	var usagePeriod *entitlement.UsagePeriodInput
	var currentUsagePeriod *timeutil.ClosedPeriod

	if model.UsagePeriod != nil {
		usagePeriod = model.UsagePeriod

		if err := usagePeriod.GetValue().Validate(); err != nil {
			return nil, &entitlement.InvalidValueError{Type: model.EntitlementType, Message: err.Error()}
		}

		calculatedPeriod, err := usagePeriod.GetValue().GetPeriodAt(clock.Now())
		if err != nil {
			return nil, err
		}

		currentUsagePeriod = &calculatedPeriod
	}

	return &entitlement.CreateEntitlementRepoInputs{
		Namespace:          model.Namespace,
		FeatureID:          feature.ID,
		FeatureKey:         feature.Key,
		UsageAttribution:   model.UsageAttribution,
		EntitlementType:    model.EntitlementType,
		Metadata:           model.Metadata,
		Annotations:        model.Annotations,
		UsagePeriod:        model.UsagePeriod,
		CurrentUsagePeriod: currentUsagePeriod,
		Config:             model.Config,
		ActiveFrom:         model.ActiveFrom,
		ActiveTo:           model.ActiveTo,
	}, nil
}

func (c *connector) AfterCreate(ctx context.Context, entitlement *entitlement.Entitlement) error {
	return nil
}

type StaticEntitlementValue struct {
	Config string `json:"config"`
}

var _ entitlement.EntitlementValue = &StaticEntitlementValue{}

func (s *StaticEntitlementValue) HasAccess() bool {
	return true
}
