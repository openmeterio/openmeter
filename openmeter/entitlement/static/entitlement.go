package staticentitlement

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
)

type Entitlement struct {
	entitlement.GenericProperties

	Config string `json:"config"`
}

func ParseFromGenericEntitlement(model *entitlement.Entitlement) (*Entitlement, error) {
	if model.EntitlementType != entitlement.EntitlementTypeStatic {
		return nil, &entitlement.WrongTypeError{Expected: entitlement.EntitlementTypeStatic, Actual: model.EntitlementType}
	}

	if lo.FromPtr(model.Config) == "" {
		return nil, &entitlement.InvalidValueError{Type: model.EntitlementType, Message: "Config is required"}
	}

	return &Entitlement{
		GenericProperties: model.GenericProperties,
		Config:            *model.Config,
	}, nil
}
