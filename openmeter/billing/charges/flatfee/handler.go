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
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type OnAllocateCreditsInput struct {
	Charge        Charge                `json:"charge"`
	ServicePeriod timeutil.ClosedPeriod `json:"servicePeriod"`
	BookedAt      time.Time             `json:"bookedAt"`
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

	if i.BookedAt.IsZero() {
		errs = append(errs, fmt.Errorf("booked at is required"))
	}

	if i.PreTaxAmountToAllocate.IsNegative() {
		errs = append(errs, fmt.Errorf("pre tax amount to allocate cannot be negative"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type OnInvoiceUsageAccruedInput struct {
	Charge        Charge                `json:"charge"`
	ServicePeriod timeutil.ClosedPeriod `json:"servicePeriod"`
	BookedAt      time.Time             `json:"bookedAt"`
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

	if i.BookedAt.IsZero() {
		errs = append(errs, fmt.Errorf("booked at is required"))
	}

	if err := i.Totals.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("totals: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type OnCustomCurrencyOverageAccruedInput struct {
	Charge Charge         `json:"charge"`
	Run    RealizationRun `json:"run"`
}

func (i OnCustomCurrencyOverageAccruedInput) CustomCurrency() currencies.Currency {
	return i.Charge.Intent.GetCurrency()
}

func (i OnCustomCurrencyOverageAccruedInput) GetFiatCurrency() (*currencyx.FiatCurrency, error) {
	costBasisIntent := i.Charge.Intent.GetCostBasisIntent()
	if costBasisIntent == nil {
		return nil, fmt.Errorf("cost basis intent is required")
	}

	return costBasisIntent.GetFiatCurrency()
}

func (i OnCustomCurrencyOverageAccruedInput) GetCostBasis() (alpacadecimal.Decimal, error) {
	if i.Charge.State.ResolvedCostBasis == nil {
		return alpacadecimal.Decimal{}, fmt.Errorf("cost basis is not resolved")
	}

	return i.Charge.State.ResolvedCostBasis.CostBasis, nil
}

func (i OnCustomCurrencyOverageAccruedInput) GetCustomCurrencyAmountAccrued() alpacadecimal.Decimal {
	return i.Run.Totals.Total
}

func (i OnCustomCurrencyOverageAccruedInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.Run.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("run: %w", err))
	}

	if err := i.CustomCurrency().Validate(); err != nil {
		errs = append(errs, fmt.Errorf("custom currency: %w", err))
	}

	if !i.CustomCurrency().IsCustom() {
		errs = append(errs, fmt.Errorf("custom currency must be custom typed currency"))
	}

	if !i.GetCustomCurrencyAmountAccrued().IsPositive() {
		errs = append(errs, fmt.Errorf("amount must be positive"))
	}

	if _, err := i.GetCostBasis(); err != nil {
		errs = append(errs, fmt.Errorf("cost basis: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type OnCustomCurrencyOverageAccruedResult struct {
	TransactionGroup ledgertransaction.GroupReference `json:"transactionGroup"`
	TotalFiatAmount  alpacadecimal.Decimal            `json:"totalFiatAmount"`
}

func (r OnCustomCurrencyOverageAccruedResult) Validate() error {
	var errs []error

	if err := r.TransactionGroup.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("transaction group: %w", err))
	}

	if r.TotalFiatAmount.IsNegative() {
		errs = append(errs, fmt.Errorf("total fiat amount cannot be negative"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CorrectCreditAllocationsInput struct {
	Charge   Charge    `json:"charge"`
	BookedAt time.Time `json:"bookedAt"`

	Corrections                  creditrealization.CorrectionRequest   `json:"corrections"`
	LineageSegmentsByRealization lineage.ActiveSegmentsByRealizationID `json:"-"`
}

func (i CorrectCreditAllocationsInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if i.BookedAt.IsZero() {
		errs = append(errs, fmt.Errorf("booked at is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (i CorrectCreditAllocationsInput) ValidateWith(currencyCalculator currencyx.Currency) error {
	var errs []error

	if err := i.Validate(); err != nil {
		return err
	}

	if err := i.Corrections.ValidateWith(currencyCalculator); err != nil {
		errs = append(errs, fmt.Errorf("corrections: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type AllocateFiatOverageCreditsInput struct {
	Charge Charge         `json:"charge"`
	Run    RealizationRun `json:"run"`

	BookedAt time.Time `json:"bookedAt"`

	// AmountToAllocate is denominated in the settlement fiat currency.
	AmountToAllocate alpacadecimal.Decimal `json:"amountToAllocate"`
}

func (i AllocateFiatOverageCreditsInput) GetFiatCurrency() (*currencyx.FiatCurrency, error) {
	costBasisIntent := i.Charge.Intent.GetCostBasisIntent()
	if costBasisIntent == nil {
		return nil, errors.New("cost basis intent is required")
	}

	return costBasisIntent.GetFiatCurrency()
}

func (i AllocateFiatOverageCreditsInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.Run.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("run: %w", err))
	}

	if !i.Charge.Intent.GetCurrency().IsCustom() {
		errs = append(errs, errors.New("charge currency must be custom"))
	}

	if i.Charge.Intent.GetSettlementMode() != productcatalog.CreditThenInvoiceSettlementMode {
		errs = append(errs, errors.New("settlement mode must be credit_then_invoice"))
	}

	if _, err := i.GetFiatCurrency(); err != nil {
		errs = append(errs, fmt.Errorf("fiat currency: %w", err))
	}

	if i.BookedAt.IsZero() {
		errs = append(errs, errors.New("booked at is required"))
	}

	if !i.AmountToAllocate.IsPositive() {
		errs = append(errs, errors.New("amount to allocate must be positive"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CorrectFiatOverageCreditAllocationsInput struct {
	Charge   Charge         `json:"charge"`
	Run      RealizationRun `json:"run"`
	BookedAt time.Time      `json:"bookedAt"`

	Corrections                  creditrealization.CorrectionRequest   `json:"corrections"`
	LineageSegmentsByRealization lineage.ActiveSegmentsByRealizationID `json:"-"`
}

func (i CorrectFiatOverageCreditAllocationsInput) GetFiatCurrency() (*currencyx.FiatCurrency, error) {
	costBasisIntent := i.Charge.Intent.GetCostBasisIntent()
	if costBasisIntent == nil {
		return nil, errors.New("cost basis intent is required")
	}

	return costBasisIntent.GetFiatCurrency()
}

func (i CorrectFiatOverageCreditAllocationsInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.Run.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("run: %w", err))
	}

	if !i.Charge.Intent.GetCurrency().IsCustom() {
		errs = append(errs, errors.New("charge currency must be custom"))
	}

	if i.Charge.Intent.GetSettlementMode() != productcatalog.CreditThenInvoiceSettlementMode {
		errs = append(errs, errors.New("settlement mode must be credit_then_invoice"))
	}

	fiatCurrency, err := i.GetFiatCurrency()
	if err != nil {
		errs = append(errs, fmt.Errorf("fiat currency: %w", err))
	} else if err := i.Corrections.ValidateWith(fiatCurrency); err != nil {
		errs = append(errs, fmt.Errorf("corrections: %w", err))
	}

	if i.BookedAt.IsZero() {
		errs = append(errs, errors.New("booked at is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type PaymentEventInput struct {
	Charge     Charge                `json:"charge"`
	Run        RealizationRun        `json:"run"`
	EventAt    time.Time             `json:"eventAt"`
	FiatAmount alpacadecimal.Decimal `json:"fiatAmount"`
}

func (i PaymentEventInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.Run.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("run: %w", err))
	}

	if i.EventAt.IsZero() {
		errs = append(errs, fmt.Errorf("event at is required"))
	}

	if !i.FiatAmount.IsPositive() {
		errs = append(errs, fmt.Errorf("fiat amount must be positive"))
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

	// OnCustomCurrencyOverageAccrued is called when uncovered custom-currency flat-fee value is accrued in fiat.
	// This must be modeled as a credit purchase flow from the ledger point of view.
	OnCustomCurrencyOverageAccrued(ctx context.Context, input OnCustomCurrencyOverageAccruedInput) (OnCustomCurrencyOverageAccruedResult, error)

	// OnCorrectCreditAllocations is called when a credit allocation needs to be corrected.
	OnCorrectCreditAllocations(ctx context.Context, input CorrectCreditAllocationsInput) (creditrealization.CreateCorrectionInputs, error)

	// OnAllocateFiatOverageCredits allocates settlement-fiat credits against a custom-currency overage.
	OnAllocateFiatOverageCredits(ctx context.Context, input AllocateFiatOverageCreditsInput) (creditrealization.CreateAllocationInputs, error)

	// OnCorrectFiatOverageCreditAllocations corrects settlement-fiat allocations for a custom-currency overage.
	OnCorrectFiatOverageCreditAllocations(ctx context.Context, input CorrectFiatOverageCreditAllocationsInput) (creditrealization.CreateCorrectionInputs, error)

	// OnFlatFeePaymentAuthorized is called when a flat fee payment is authorized.
	OnPaymentAuthorized(ctx context.Context, input OnPaymentAuthorizedInput) (ledgertransaction.GroupReference, error)

	// OnFlatFeePaymentSettled is called when a flat fee payment is settled.
	OnPaymentSettled(ctx context.Context, input OnPaymentSettledInput) (ledgertransaction.GroupReference, error)

	// OnFlatFeePaymentUncollectible is called when a flat fee payment is uncollectible
	OnPaymentUncollectible(ctx context.Context, charge Charge) (ledgertransaction.GroupReference, error)
}
