package flatfee

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
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

	return models.NewNillableGenericValidationError(errors.Join(errs...))
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

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type OnCreditsOnlyUsageAccruedInput struct {
	Charge           Charge                `json:"charge"`
	AmountToAllocate alpacadecimal.Decimal `json:"amountToAllocate"`
}

func (i OnCreditsOnlyUsageAccruedInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if i.AmountToAllocate.IsNegative() {
		errs = append(errs, fmt.Errorf("amount to allocate cannot be negative"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CreditsOnlyUsageAccruedCorrectionInput struct {
	Charge     Charge    `json:"charge"`
	AllocateAt time.Time `json:"allocateAt"`

	Corrections                  creditrealization.CorrectionRequest   `json:"corrections"`
	LineageSegmentsByRealization lineage.ActiveSegmentsByRealizationID `json:"-"`
}

func (i CreditsOnlyUsageAccruedCorrectionInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if i.AllocateAt.IsZero() {
		errs = append(errs, fmt.Errorf("allocate at is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (i CreditsOnlyUsageAccruedCorrectionInput) ValidateWith(currencyCalculator currencyx.Calculator) error {
	var errs []error

	if err := i.Validate(); err != nil {
		return err
	}

	if err := i.Corrections.ValidateWith(currencyCalculator); err != nil {
		errs = append(errs, fmt.Errorf("corrections: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type Handler interface {
	// OnFlatFeeAssignedToInvoice is called when a flat fee is being assigned to an invoice
	OnAssignedToInvoice(ctx context.Context, input OnAssignedToInvoiceInput) (creditrealization.CreateAllocationInputs, error)

	// OnFlatFeeStandardInvoiceUsageAccrued is called when the remaining usage is sent to the customer on a standard invoice.
	OnInvoiceUsageAccrued(ctx context.Context, input OnInvoiceUsageAccruedInput) (ledgertransaction.GroupReference, error)

	// OnCreditsOnlyUsageAccrued is called when a credit-only flat fee becomes active (clock >= InvoiceAt)
	// and the full amount needs to be allocated as credits.
	OnCreditsOnlyUsageAccrued(ctx context.Context, input OnCreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error)

	// OnCreditsOnlyUsageAccruedCorrection is called when a credit allocation needs to be corrected.
	OnCreditsOnlyUsageAccruedCorrection(ctx context.Context, input CreditsOnlyUsageAccruedCorrectionInput) (creditrealization.CreateCorrectionInputs, error)

	// OnFlatFeePaymentAuthorized is called when a flat fee payment is authorized
	OnPaymentAuthorized(ctx context.Context, charge Charge) (ledgertransaction.GroupReference, error)

	// OnFlatFeePaymentSettled is called when a flat fee payment is settled
	OnPaymentSettled(ctx context.Context, charge Charge) (ledgertransaction.GroupReference, error)

	// OnFlatFeePaymentUncollectible is called when a flat fee payment is uncollectible
	OnPaymentUncollectible(ctx context.Context, charge Charge) (ledgertransaction.GroupReference, error)
}
