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
