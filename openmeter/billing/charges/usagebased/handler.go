package usagebased

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type CreditsOnlyUsageAccruedInput struct {
	Charge           Charge                `json:"charge"`
	Run              RealizationRun        `json:"run"`
	BookedAt         time.Time             `json:"bookedAt"`
	AmountToAllocate alpacadecimal.Decimal `json:"amountToAllocate"`
}

func (i CreditsOnlyUsageAccruedInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.Run.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("run: %w", err))
	}

	if i.BookedAt.IsZero() {
		errs = append(errs, fmt.Errorf("booked at is required"))
	}

	if !i.AmountToAllocate.IsPositive() {
		errs = append(errs, fmt.Errorf("amount to allocate must be positive"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CreditsOnlyUsageAccruedCorrectionInput struct {
	Charge   Charge         `json:"charge"`
	Run      RealizationRun `json:"run"`
	BookedAt time.Time      `json:"bookedAt"`

	Corrections                  creditrealization.CorrectionRequest   `json:"corrections"`
	LineageSegmentsByRealization lineage.ActiveSegmentsByRealizationID `json:"-"`
}

func (i CreditsOnlyUsageAccruedCorrectionInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.Run.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("run: %w", err))
	}

	if i.BookedAt.IsZero() {
		errs = append(errs, fmt.Errorf("booked at is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (i CreditsOnlyUsageAccruedCorrectionInput) ValidateWith(currency currencyx.Currency) error {
	var errs []error

	if err := i.Validate(); err != nil {
		return err
	}

	if currency == nil {
		errs = append(errs, fmt.Errorf("currency is required"))
	}

	if currency != nil {
		if err := i.Corrections.ValidateWith(currency); err != nil {
			errs = append(errs, fmt.Errorf("corrections: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type OnInvoiceUsageAccruedInput struct {
	Charge        Charge                `json:"charge"`
	Run           RealizationRun        `json:"run"`
	ServicePeriod timeutil.ClosedPeriod `json:"servicePeriod"`
	BookedAt      time.Time             `json:"bookedAt"`
	Amount        alpacadecimal.Decimal `json:"amount"`
}

func (i OnInvoiceUsageAccruedInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.Run.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("run: %w", err))
	}

	if err := i.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if i.BookedAt.IsZero() {
		errs = append(errs, fmt.Errorf("booked at is required"))
	}

	if i.Amount.IsNegative() {
		errs = append(errs, fmt.Errorf("amount cannot be negative"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type OnCustomCurrencyOverageAccruedInput struct {
	Charge Charge         `json:"charge"`
	Run    RealizationRun `json:"run"`
}

func (i OnCustomCurrencyOverageAccruedInput) CustomCurrency() currencies.Currency {
	return i.Charge.Intent.GetEffectiveIntent().Currency
}

func (i OnCustomCurrencyOverageAccruedInput) GetFiatCurrency() (*currencyx.FiatCurrency, error) {
	return i.Charge.Intent.GetEffectiveIntent().CostBasis.GetFiatCurrency()
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

	effectiveIntent := i.Charge.Intent.GetEffectiveIntent()

	if err := effectiveIntent.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("custom currency: %w", err))
	}

	if !effectiveIntent.Currency.IsCustom() {
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
	// OnInvoiceUsageAccrued is called when invoice-settled usage-based usage is sent to the customer.
	OnInvoiceUsageAccrued(ctx context.Context, input OnInvoiceUsageAccruedInput) (ledgertransaction.GroupReference, error)

	// OnPaymentAuthorized is called when an invoice-backed usage-based run receives payment authorization.
	OnPaymentAuthorized(ctx context.Context, input OnPaymentAuthorizedInput) (ledgertransaction.GroupReference, error)

	// OnPaymentSettled is called when an invoice-backed usage-based run payment is settled.
	OnPaymentSettled(ctx context.Context, input OnPaymentSettledInput) (ledgertransaction.GroupReference, error)

	// OnCustomCurrencyOverageAccrued is called when uncovered custom-currency usage is accrued in fiat.
	// This must be modeled as a credit purchase flow from the ledger point of view.
	OnCustomCurrencyOverageAccrued(ctx context.Context, input OnCustomCurrencyOverageAccruedInput) (OnCustomCurrencyOverageAccruedResult, error)

	// OnCreditsOnlyUsageAccrued is called when a credit-only usage-based charge needs to be allocated as credits fully.
	OnCreditsOnlyUsageAccrued(ctx context.Context, input CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error)

	// OnCreditsOnlyUsageAccruedCorrection is called when a credit-only usage-based charge needs to be corrected.
	OnCreditsOnlyUsageAccruedCorrection(ctx context.Context, input CreditsOnlyUsageAccruedCorrectionInput) (creditrealization.CreateCorrectionInputs, error)
}

type UnimplementedHandler struct{}

var _ Handler = (*UnimplementedHandler)(nil)

func (h UnimplementedHandler) OnInvoiceUsageAccrued(ctx context.Context, input OnInvoiceUsageAccruedInput) (ledgertransaction.GroupReference, error) {
	return ledgertransaction.GroupReference{}, errors.New("not implemented")
}

func (h UnimplementedHandler) OnCustomCurrencyOverageAccrued(ctx context.Context, input OnCustomCurrencyOverageAccruedInput) (OnCustomCurrencyOverageAccruedResult, error) {
	return OnCustomCurrencyOverageAccruedResult{}, errors.New("not implemented")
}

func (h UnimplementedHandler) OnPaymentAuthorized(ctx context.Context, input OnPaymentAuthorizedInput) (ledgertransaction.GroupReference, error) {
	return ledgertransaction.GroupReference{}, errors.New("not implemented")
}

func (h UnimplementedHandler) OnPaymentSettled(ctx context.Context, input OnPaymentSettledInput) (ledgertransaction.GroupReference, error) {
	return ledgertransaction.GroupReference{}, errors.New("not implemented")
}

func (h UnimplementedHandler) OnCreditsOnlyUsageAccrued(ctx context.Context, input CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
	return nil, errors.New("not implemented")
}

func (h UnimplementedHandler) OnCreditsOnlyUsageAccruedCorrection(ctx context.Context, input CreditsOnlyUsageAccruedCorrectionInput) (creditrealization.CreateCorrectionInputs, error) {
	return nil, errors.New("not implemented")
}
