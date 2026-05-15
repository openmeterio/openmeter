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

type OnAllocateCreditsInput struct {
	Charge        Charge                `json:"charge"`
	ServicePeriod timeutil.ClosedPeriod `json:"servicePeriod"`
	// PreTaxAmountToAllocate is the pre-tax amount to allocate from credits.
	// The input charge's settlement mode governs whether this may create a negative balance.
	PreTaxAmountToAllocate alpacadecimal.Decimal `json:"preTaxAmountToAllocate"`
}

func (i OnAllocateCreditsInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if i.PreTaxAmountToAllocate.IsNegative() {
		errs = append(errs, fmt.Errorf("pre tax amount to allocate cannot be negative"))
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

type CorrectCreditAllocationsInput struct {
	Charge     Charge    `json:"charge"`
	AllocateAt time.Time `json:"allocateAt"`

	Corrections                  creditrealization.CorrectionRequest   `json:"corrections"`
	LineageSegmentsByRealization lineage.ActiveSegmentsByRealizationID `json:"-"`
}

func (i CorrectCreditAllocationsInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if i.AllocateAt.IsZero() {
		errs = append(errs, fmt.Errorf("allocate at is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (i CorrectCreditAllocationsInput) ValidateWith(currencyCalculator currencyx.Calculator) error {
	var errs []error

	if err := i.Validate(); err != nil {
		return err
	}

	if err := i.Corrections.ValidateWith(currencyCalculator); err != nil {
		errs = append(errs, fmt.Errorf("corrections: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type PaymentEventInput struct {
	Charge Charge                `json:"charge"`
	Amount alpacadecimal.Decimal `json:"amount"`
}

func (i PaymentEventInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if i.Amount.IsNegative() {
		errs = append(errs, fmt.Errorf("amount cannot be negative"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type (
	OnPaymentAuthorizedInput = PaymentEventInput
	OnPaymentSettledInput    = PaymentEventInput
)

type Handler interface {
	// OnAllocateCredits is called when a flat fee allocates credits.
	OnAllocateCredits(ctx context.Context, input OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error)

	// OnFlatFeeStandardInvoiceUsageAccrued is called when the remaining usage is sent to the customer on a standard invoice.
	OnInvoiceUsageAccrued(ctx context.Context, input OnInvoiceUsageAccruedInput) (ledgertransaction.GroupReference, error)

	// OnCorrectCreditAllocations is called when a credit allocation needs to be corrected.
	OnCorrectCreditAllocations(ctx context.Context, input CorrectCreditAllocationsInput) (creditrealization.CreateCorrectionInputs, error)

	// OnFlatFeePaymentAuthorized is called when a flat fee payment is authorized.
	OnPaymentAuthorized(ctx context.Context, input OnPaymentAuthorizedInput) (ledgertransaction.GroupReference, error)

	// OnFlatFeePaymentSettled is called when a flat fee payment is settled.
	OnPaymentSettled(ctx context.Context, input OnPaymentSettledInput) (ledgertransaction.GroupReference, error)

	// OnFlatFeePaymentUncollectible is called when a flat fee payment is uncollectible
	OnPaymentUncollectible(ctx context.Context, charge Charge) (ledgertransaction.GroupReference, error)
}
