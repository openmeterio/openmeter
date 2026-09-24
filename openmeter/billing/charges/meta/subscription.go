package meta

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionReference struct {
	SubscriptionID string `json:"subscriptionID"`
	PhaseID        string `json:"phaseID"`
	ItemID         string `json:"itemID"`
}

func (r SubscriptionReference) Equal(other SubscriptionReference) bool {
	return r == other
}

type UpdateSubscriptionReferenceInput struct {
	ChargeID ChargeID
	Target   SubscriptionReference
}

func (i UpdateSubscriptionReferenceInput) Validate() error {
	var errs []error

	if err := i.ChargeID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge ID: %w", err))
	}

	if err := i.Target.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("target: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (r SubscriptionReference) Validate() error {
	var errs []error

	if r.SubscriptionID == "" {
		errs = append(errs, fmt.Errorf("subscription ID is required"))
	}

	if r.PhaseID == "" {
		errs = append(errs, fmt.Errorf("phase ID is required"))
	}

	if r.ItemID == "" {
		errs = append(errs, fmt.Errorf("item ID is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// SubscriptionPlan snapshots the subscription's catalog reference when a charge is created.
// Later subscription migrations do not change the charge's recorded attribution.
type SubscriptionPlan struct {
	Key     string `json:"key"`
	Version int    `json:"version"`
}

func (p SubscriptionPlan) Validate() error {
	var errs []error
	if p.Key == "" {
		errs = append(errs, errors.New("plan key is required"))
	}
	if p.Version < 1 {
		errs = append(errs, errors.New("plan version must be positive"))
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
