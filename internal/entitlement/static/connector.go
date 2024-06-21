package staticentitlement

import (
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

func (c *connector) SetDefaults(model *entitlement.CreateEntitlementInputs) {
	model.EntitlementType = entitlement.EntitlementTypeStatic
	model.MeasureUsageFrom = nil
	model.IssueAfterReset = nil
	model.IsSoftLimit = nil
}

func (c *connector) ValidateForFeature(entitlement *entitlement.CreateEntitlementInputs, feature productcatalog.Feature) error {
	return nil
}

type StaticEntitlementValue struct {
	Config *string `json:"config,omitempty"`
}

var _ entitlement.EntitlementValue = &StaticEntitlementValue{}

func (s *StaticEntitlementValue) HasAccess() bool {
	return true
}
