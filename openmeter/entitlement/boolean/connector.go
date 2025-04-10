package booleanentitlement

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type Connector interface {
	entitlement.SubTypeConnector
}

type connector struct{}

func NewBooleanEntitlementConnector() Connector {
	return &connector{}
}

func (c *connector) GetValue(ctx context.Context, entitlement *entitlement.Entitlement, at time.Time) (entitlement.EntitlementValue, error) {
	_, err := ParseFromGenericEntitlement(entitlement)
	if err != nil {
		return nil, err
	}

	return &BooleanEntitlementValue{}, nil
}

func (c *connector) BeforeCreate(model entitlement.CreateEntitlementInputs, feature feature.Feature) (*entitlement.CreateEntitlementRepoInputs, error) {
	model.EntitlementType = entitlement.EntitlementTypeBoolean
	if model.MeasureUsageFrom != nil ||
		model.IssueAfterReset != nil ||
		model.IsSoftLimit != nil ||
		model.Config != nil {
		return nil, &entitlement.InvalidValueError{Type: model.EntitlementType, Message: "Invalid inputs for type"}
	}

	var usagePeriod *entitlement.UsagePeriod
	var currentUsagePeriod *timeutil.ClosedPeriod

	if model.UsagePeriod != nil {
		usagePeriod = model.UsagePeriod

		if err := usagePeriod.Validate(); err != nil {
			return nil, &entitlement.InvalidValueError{Type: model.EntitlementType, Message: err.Error()}
		}

		calculatedPeriod, err := usagePeriod.GetCurrentPeriodAt(clock.Now())
		if err != nil {
			return nil, err
		}

		currentUsagePeriod = &calculatedPeriod
	}

	return &entitlement.CreateEntitlementRepoInputs{
		Namespace:          model.Namespace,
		FeatureID:          feature.ID,
		FeatureKey:         feature.Key,
		SubjectKey:         model.SubjectKey,
		EntitlementType:    model.EntitlementType,
		Metadata:           model.Metadata,
		Annotations:        model.Annotations,
		UsagePeriod:        model.UsagePeriod,
		CurrentUsagePeriod: currentUsagePeriod,
		ActiveFrom:         model.ActiveFrom,
		ActiveTo:           model.ActiveTo,
	}, nil
}

func (c *connector) AfterCreate(ctx context.Context, entitlement *entitlement.Entitlement) error {
	return nil
}

type BooleanEntitlementValue struct{}

var _ entitlement.EntitlementValue = &BooleanEntitlementValue{}

func (v *BooleanEntitlementValue) HasAccess() bool {
	return true
}
