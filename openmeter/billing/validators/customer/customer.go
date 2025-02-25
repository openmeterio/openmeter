package customer

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

var _ customer.RequestValidator = (*Validator)(nil)

func NewValidator(billingService billing.Service) (*Validator, error) {
	if billingService == nil {
		return nil, fmt.Errorf("billing service is required")
	}

	return &Validator{
		billingService: billingService,
	}, nil
}

type Validator struct {
	customer.NoopRequestValidator
	billingService billing.Service
}

func (v *Validator) ValidateDeleteCustomer(ctx context.Context, input customer.DeleteCustomerInput) error {
	// A customer can only be deleted if all of his invocies are in final state

	if err := input.Validate(); err != nil {
		return err
	}

	gatheringInvoices, err := v.billingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Namespaces: []string{input.Namespace},
		Customers:  []string{input.ID},
	})
	if err != nil {
		return err
	}

	errs := make([]error, 0, len(gatheringInvoices.Items))
	for _, inv := range gatheringInvoices.Items {
		if inv.Status == billing.InvoiceStatusGathering {
			errs = append(errs, fmt.Errorf("invoice %s is still in gathering state", inv.ID))

			continue
		}

		if !inv.Status.IsFinal() {
			errs = append(errs, fmt.Errorf("invoice %s is not in final state, please either delete the invoice or mark it uncollectible", inv.ID))
		}
	}

	return errors.Join(errs...)
}
