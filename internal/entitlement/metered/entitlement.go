package meteredentitlement

import (
	"time"

	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/pkg/recurrence"
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

	// UsagePeriod defines the recurring period for usage calculations.
	UsagePeriod entitlement.UsagePeriod `json:"usagePeriod,omitempty"`

	// CurrentPeriod defines the current period for usage calculations.
	CurrentUsagePeriod recurrence.Period `json:"currentUsagePeriod,omitempty"`

	// LastReset defines the last time the entitlement was reset.
	LastReset time.Time `json:"lastReset"`
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

	if model.UsagePeriod == nil {
		return nil, &entitlement.InvalidValueError{Message: "UsagePeriod is required", Type: model.EntitlementType}
	}

	if model.LastReset == nil {
		return nil, &entitlement.InvalidValueError{Message: "LastReset is required", Type: model.EntitlementType}
	}

	if model.CurrentUsagePeriod == nil {
		return nil, &entitlement.InvalidValueError{Message: "CurrentUsagePeriod is required", Type: model.EntitlementType}
	}

	return &Entitlement{
		GenericProperties: model.GenericProperties,

		MeasureUsageFrom:   *model.MeasureUsageFrom,
		IssuesAfterReset:   model.IssueAfterReset,
		IsSoftLimit:        *model.IsSoftLimit,
		UsagePeriod:        *model.UsagePeriod,
		LastReset:          *model.LastReset,
		CurrentUsagePeriod: *model.CurrentUsagePeriod,
	}, nil
}
