package flatfee

import (
	"context"
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type OnAssignedToInvoiceInput struct {
	Charge            Charge                `json:"charge"`
	ServicePeriod     timeutil.ClosedPeriod `json:"servicePeriod"`
	PreTaxTotalAmount alpacadecimal.Decimal `json:"totalAmount"`
}

func (i OnAssignedToInvoiceInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if i.PreTaxTotalAmount.IsNegative() {
		errs = append(errs, fmt.Errorf("pre tax total amount cannot be negative"))
	}

	return errors.Join(errs...)
}

type OnInvoiceUsageAccruedInput struct {
	Charge        Charge                `json:"charge"`
	ServicePeriod timeutil.ClosedPeriod `json:"servicePeriod"`
	Totals        totals.Totals         `json:"totals"`
}

func (i OnInvoiceUsageAccruedInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if err := i.Totals.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("totals: %w", err))
	}

	return errors.Join(errs...)
}

type Handler interface {
	// OnFlatFeeAssignedToInvoice is called when a flat fee is being assigned to an invoice
	OnAssignedToInvoice(ctx context.Context, input OnAssignedToInvoiceInput) ([]creditrealization.CreateInput, error)

	// OnFlatFeeStandardInvoiceUsageAccrued is called when the remaining usage is sent to the customer on a standard invoice.
	OnInvoiceUsageAccrued(ctx context.Context, input OnInvoiceUsageAccruedInput) (ledgertransaction.GroupReference, error)

	// OnFlatFeePaymentAuthorized is called when a flat fee payment is authorized
	OnPaymentAuthorized(ctx context.Context, charge Charge) (ledgertransaction.GroupReference, error)

	// OnFlatFeePaymentSettled is called when a flat fee payment is settled
	OnPaymentSettled(ctx context.Context, charge Charge) (ledgertransaction.GroupReference, error)

	// OnFlatFeePaymentUncollectible is called when a flat fee payment is uncollectible
	OnPaymentUncollectible(ctx context.Context, charge Charge) (ledgertransaction.GroupReference, error)
}
