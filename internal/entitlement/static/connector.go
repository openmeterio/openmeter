package staticentitlement

import (
	"context"
	"encoding/json"
	"time"

	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/internal/productcatalog"
)

type Connector interface {
	entitlement.SubTypeConnector
}

type connector struct{}

func NewStaticEntitlementConnector() Connector {
	return &connector{}
}

func (c *connector) GetValue(entitlement *entitlement.Entitlement, at time.Time) (entitlement.EntitlementValue, error) {
	static, err := ParseFromGenericEntitlement(entitlement)
	if err != nil {
		return nil, err
	}

	return &StaticEntitlementValue{
		Config: static.Config,
	}, nil
}

func (c *connector) BeforeCreate(model *entitlement.CreateEntitlementInputs, feature *productcatalog.Feature) error {
	model.EntitlementType = entitlement.EntitlementTypeStatic

	if model.MeasureUsageFrom != nil ||
		model.IssueAfterReset != nil ||
		model.IsSoftLimit != nil {
		return &entitlement.InvalidValueError{Type: model.EntitlementType, Message: "Invalid inputs for type"}
	}

	// validate that config is JSON parseable
	if model.Config == nil {
		return &entitlement.InvalidValueError{Type: model.EntitlementType, Message: "Config is required"}
	}

	if !json.Valid([]byte(*model.Config)) {
		return &entitlement.InvalidValueError{Type: model.EntitlementType, Message: "Config is not valid JSON"}
	}

	if err := json.Unmarshal([]byte(*model.Config), &map[string]interface{}{}); err != nil {
		return &entitlement.InvalidValueError{Type: model.EntitlementType, Message: "Config is not a valid JSON object"}
	}

	return nil
}

func (c *connector) AfterCreate(ctx context.Context, entitlement *entitlement.Entitlement) error {
	return nil
}

type StaticEntitlementValue struct {
	Config string `json:"config,omitempty"`
}

var _ entitlement.EntitlementValue = &StaticEntitlementValue{}

func (s *StaticEntitlementValue) HasAccess() bool {
	return true
}
