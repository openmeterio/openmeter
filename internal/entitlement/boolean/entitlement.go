package booleanentitlement

import "github.com/openmeterio/openmeter/internal/entitlement"

type Entitlement struct {
	entitlement.GenericProperties
}

func ParseFromGenericEntitlement(model *entitlement.Entitlement) (*Entitlement, error) {
	if model.EntitlementType != entitlement.EntitlementTypeBoolean {
		return nil, &entitlement.WrongTypeError{Expected: entitlement.EntitlementTypeBoolean, Actual: model.EntitlementType}
	}

	return &Entitlement{
		GenericProperties: model.GenericProperties,
	}, nil
}
