package meteredentitlement

import (
	"time"

	"github.com/openmeterio/openmeter/internal/entitlement"
)

type Entitlement struct {
	entitlement.GenericProperties

	// MeasureUsageFrom defines the time from which usage should be measured.
	// This is a global value, in most cases the same value as `CreatedAt` should be fine.
	MeasureUsageFrom time.Time `json:"measureUsageFrom,omitempty"`

	// IssueAfterReset defines an amount of usage that will be issued after a reset.
	// This affordance will only be usable until the next reset.
	IssuesAfterReset *float64 `json:"issueAfterReset,omitempty"`

	// IsSoftLimit defines if the entitlement is a soft limit. By default when balance falls to 0
	// access will be disabled. If this is a soft limit, access will be allowed nonetheless.
	IsSoftLimit bool `json:"isSoftLimit,omitempty"`
}

func ParseFromGenericEntitlement(model *entitlement.Entitlement) (*Entitlement, error) {
	if model.EntitlementType != entitlement.EntitlementTypeMetered {
		return nil, &entitlement.WrongTypeError{Expected: entitlement.EntitlementTypeMetered, Actual: model.EntitlementType}
	}

	if model.MeasureUsageFrom == nil {
		return nil, &entitlement.InvalidValueError{Message: "MeasureUsageFrom is required", Type: model.EntitlementType}
	}

	if model.IsSoftLimit == nil {
		return nil, &entitlement.InvalidValueError{Message: "IsSoftLimit is required", Type: model.EntitlementType}
	}

	return &Entitlement{
		GenericProperties: model.GenericProperties,

		MeasureUsageFrom: *model.MeasureUsageFrom,
		IssuesAfterReset: model.IssueAfterReset,
		IsSoftLimit:      *model.IsSoftLimit,
	}, nil
}
