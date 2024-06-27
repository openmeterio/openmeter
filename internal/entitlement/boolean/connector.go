package booleanentitlement

import (
	"time"

	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/internal/productcatalog"
)

type Connector interface {
	entitlement.SubTypeConnector
}

type connector struct{}

func NewBooleanEntitlementConnector() Connector {
	return &connector{}
}

func (c *connector) GetValue(entitlement *entitlement.Entitlement, at time.Time) (entitlement.EntitlementValue, error) {
	_, err := ParseFromGenericEntitlement(entitlement)
	if err != nil {
		return nil, err
	}

	return &BooleanEntitlementValue{}, nil
}

func (c *connector) BeforeCreate(model *entitlement.CreateEntitlementInputs, feature *productcatalog.Feature) error {
	model.EntitlementType = entitlement.EntitlementTypeBoolean
	if model.MeasureUsageFrom != nil ||
		model.IssueAfterReset != nil ||
		model.IsSoftLimit != nil ||
		model.Config != nil {
		return &entitlement.InvalidValueError{Type: model.EntitlementType, Message: "Invalid inputs for type"}
	}

	return nil
}

func (c *connector) AfterCreate(entitlement *entitlement.Entitlement) error {
	return nil
}

type BooleanEntitlementValue struct {
}

var _ entitlement.EntitlementValue = &BooleanEntitlementValue{}

func (v *BooleanEntitlementValue) HasAccess() bool {
	return true
}
