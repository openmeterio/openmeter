package staticentitlement

import (
	"github.com/openmeterio/openmeter/internal/entitlement"
)

type Entitlement struct {
	entitlement.GenericProperties

	Config []byte `json:"config,omitempty"`
}

func ParseFromGenericEntitlement(model *entitlement.Entitlement) (*Entitlement, error) {
	if model.EntitlementType != entitlement.EntitlementTypeStatic {
		return nil, &entitlement.WrongTypeError{Expected: entitlement.EntitlementTypeStatic, Actual: model.EntitlementType}
	}

	if model.Config == nil {
		return nil, &entitlement.InvalidValueError{Type: model.EntitlementType, Message: "Config is required"}
	}

	return &Entitlement{
		GenericProperties: model.GenericProperties,
		Config:            model.Config,
	}, nil
}
