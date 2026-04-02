package meta

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

type ManagedResource struct {
	models.NamespacedModel
	models.ManagedModel
	ID string `json:"id"`
}

func (r ManagedResource) Validate() error {
	var errs []error

	if err := r.NamespacedModel.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("namespaced model: %w", err))
	}

	if err := r.ManagedModel.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("managed model: %w", err))
	}

	if r.ID == "" {
		errs = append(errs, fmt.Errorf("id is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (r ManagedResource) GetChargeID() ChargeID {
	return ChargeID{
		Namespace: r.Namespace,
		ID:        r.ID,
	}
}
