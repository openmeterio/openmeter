package charges

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/expand"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Charge struct {
	models.ManagedResource

	Intent       Intent       `json:"intent"`
	Realizations Realizations `json:"realizations"`
}

type Realizations struct {
	StandardInvoice []StandardInvoiceRealization `json:"standardInvoice"`
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

	return errors.Join(errs...)
}

type ChargeExpand string

func (e ChargeExpand) Values() []ChargeExpand {
	return nil
}

type ChargeExpands = expand.Expand[ChargeExpand]

type Charges []Charge

type CreateChargeInput struct {
	Intent Intent `json:"intent"`
}

type GetChargesForSubscriptionInput struct {
	SubscriptionID models.NamespacedID `json:"subscriptionID"`

	// Temp until schema migrations are done
	CustomerID customer.CustomerID `json:"customerID"`
	Expand     ChargeExpands       `json:"expand"`
}

func (i GetChargesForSubscriptionInput) Validate() error {
	var errs []error

	if err := i.SubscriptionID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("subscription ID: %w", err))
	}

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer ID: %w", err))
	}

	if i.CustomerID.Namespace != i.SubscriptionID.Namespace {
		errs = append(errs, fmt.Errorf("customer ID namespace %s does not match subscription ID namespace %s", i.CustomerID.Namespace, i.SubscriptionID.Namespace))
	}

	if err := i.Expand.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("expand: %w", err))
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
