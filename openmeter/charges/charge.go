package charges

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
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

type Charges []Charge

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

type DeleteChargesByUniqueReferenceIDInput struct {
	Customer           customer.CustomerID `json:"customer"`
	UniqueReferenceIDs []string            `json:"uniqueReferenceIDs"`
}

func (i DeleteChargesByUniqueReferenceIDInput) Validate() error {
	if err := i.Customer.Validate(); err != nil {
		return fmt.Errorf("customer ID: %w", err)
	}

	return nil
}

type UpsertChargesByChildUniqueReferenceIDInput struct {
	Customer customer.CustomerID `json:"customer"`

	Charges
}

func (i UpsertChargesByChildUniqueReferenceIDInput) Validate() error {
	if err := i.Customer.Validate(); err != nil {
		return fmt.Errorf("namespace is required")
	}

	return errors.Join(
		lo.Map(i.Charges, func(charge Charge, _ int) error {
			intentMeta := charge.Intent.IntentMeta

			if charge.Namespace != i.Customer.Namespace {
				return fmt.Errorf("charge namespace %s does not match input namespace %s", charge.Namespace, i.Customer.Namespace)
			}

			if intentMeta.CustomerID != i.Customer.ID {
				return fmt.Errorf("charge customer ID %s does not match input customer ID %s", intentMeta.CustomerID, i.Customer.ID)
			}

			if charge.Intent.UniqueReferenceID == nil {
				return fmt.Errorf("charge unique reference ID cannot be empty for upsert")
			}

			return charge.Validate()
		})...,
	)
}
