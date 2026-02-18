package charges

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

type Charge struct {
	models.ManagedResource

	Intent       Intent       `json:"intent"`
	Realizations Realizations `json:"realizations"`
}

func (c Charge) Validate() error {
	var errs []error

	if err := c.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent base: %w", err))
	}

	if c.Name == "" {
		errs = append(errs, fmt.Errorf("name is required"))
	}

	if c.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace is required"))
	}

	if err := c.Realizations.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("realizations: %w", err))
	}

	return errors.Join(errs...)
}

type Realizations struct {
	StandardInvoice []StandardInvoiceRealization `json:"standardInvoice"`
}

func (r Realizations) Validate() error {
	var errs []error

	for idx, realization := range r.StandardInvoice {
		if err := realization.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("standard invoice realization[%d]: %w", idx, err))
		}
	}

	return errors.Join(errs...)
}
