package creditreconciliation

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ReconcileCustomCurrencyWithFiatOverageInput struct {
	UnallocatedTotals totals.Totals
	Currency          currencies.Currency
	CostBasisIntent   *costbasis.Intent
	ResolvedCostBasis *costbasis.State

	ChargeCurrencyHandler Handler
	FiatOverageHandler    Handler
}

func (i ReconcileCustomCurrencyWithFiatOverageInput) Validate() error {
	var errs []error
	currencyValid := true
	costBasisIntentValid := true

	if err := i.UnallocatedTotals.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("unallocated totals: %w", err))
	}

	if !i.UnallocatedTotals.CreditsTotal.IsZero() {
		errs = append(errs, errors.New("unallocated totals must not include credits"))
	}

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
		currencyValid = false
	} else if !i.Currency.IsCustom() {
		errs = append(errs, errors.New("currency must be custom"))
		currencyValid = false
	}

	if i.CostBasisIntent == nil {
		errs = append(errs, errors.New("cost basis intent is required"))
		costBasisIntentValid = false
	} else if err := i.CostBasisIntent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("cost basis intent: %w", err))
		costBasisIntentValid = false
	}

	if i.ResolvedCostBasis == nil {
		errs = append(errs, errors.New("resolved cost basis is required"))
	} else if err := i.ResolvedCostBasis.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("resolved cost basis: %w", err))
	}

	if i.ChargeCurrencyHandler == nil {
		errs = append(errs, errors.New("charge currency handler is required"))
	} else {
		var expectedCurrency currencyx.Currency
		if currencyValid {
			expectedCurrency = i.Currency
		}
		if err := validateHandlerCurrency(i.ChargeCurrencyHandler, expectedCurrency); err != nil {
			errs = append(errs, fmt.Errorf("charge currency handler: %w", err))
		}
	}

	if i.FiatOverageHandler == nil {
		errs = append(errs, errors.New("fiat overage handler is required"))
	} else if costBasisIntentValid {
		fiatCurrency, err := i.CostBasisIntent.GetFiatCurrency()
		if err != nil {
			errs = append(errs, fmt.Errorf("get cost basis fiat currency: %w", err))
		} else if err := validateHandlerCurrency(i.FiatOverageHandler, fiatCurrency); err != nil {
			errs = append(errs, fmt.Errorf("fiat overage handler: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func validateHandlerCurrency(handler Handler, expected currencyx.Currency) error {
	var errs []error

	if err := handler.Validate(); err != nil {
		errs = append(errs, err)
	}

	currencyCalculator := handler.CurrencyCalculator()
	if currencyCalculator == nil {
		errs = append(errs, errors.New("currency calculator is required"))
	} else if err := currencyCalculator.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency calculator: %w", err))
	} else if expected != nil && !currencyCalculator.Details().Code.Equal(expected.Details().Code) {
		errs = append(errs, fmt.Errorf(
			"currency calculator must match monetary domain: %s != %s",
			currencyCalculator.Details().Code,
			expected.Details().Code,
		))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CustomCurrencyWithFiatOverageResult struct {
	Totals totals.Totals

	ChargeCurrency ReconcileResult

	ConvertedFiatOverage meta.FiatOverage
	FiatOverageCredits   ReconcileResult
	RemainingFiatOverage alpacadecimal.Decimal
}

// ReconcileCustomCurrencyWithFiatOverage allocates charge-currency credits
// before converting the remaining overage to fiat and reconciling fiat credits.
// Both monetary domains retain their own handler and realization lineage.
func ReconcileCustomCurrencyWithFiatOverage(
	ctx context.Context,
	in ReconcileCustomCurrencyWithFiatOverageInput,
) (CustomCurrencyWithFiatOverageResult, error) {
	if err := in.Validate(); err != nil {
		return CustomCurrencyWithFiatOverageResult{}, err
	}

	chargeCurrencyResult, err := Reconcile(ctx, ReconcileInput{
		TargetAmount: in.UnallocatedTotals.Total,
		Handler:      in.ChargeCurrencyHandler,
	})
	if err != nil {
		return CustomCurrencyWithFiatOverageResult{}, fmt.Errorf("reconcile charge currency credits: %w", err)
	}

	chargeCurrencyRealizations := append(
		slices.Clone(in.ChargeCurrencyHandler.Realizations()),
		chargeCurrencyResult.Realizations...,
	)
	chargeCurrencyCreditsTotal := in.Currency.RoundToPrecision(chargeCurrencyRealizations.Sum())
	if chargeCurrencyCreditsTotal.GreaterThan(in.UnallocatedTotals.Total) {
		return CustomCurrencyWithFiatOverageResult{}, fmt.Errorf(
			"charge currency credit allocations exceed total [total=%s allocated=%s]",
			in.UnallocatedTotals.Total,
			chargeCurrencyCreditsTotal,
		)
	}

	reconciledTotals := in.UnallocatedTotals
	reconciledTotals.CreditsTotal = chargeCurrencyCreditsTotal
	reconciledTotals.Total = in.Currency.RoundToPrecision(
		in.UnallocatedTotals.Total.Sub(chargeCurrencyCreditsTotal),
	)

	fiatOverage, err := meta.ConvertCustomCurrencyOverageToFiat(meta.ConvertCustomCurrencyOverageToFiatInput{
		Currency:          in.Currency,
		CostBasisIntent:   in.CostBasisIntent,
		ResolvedCostBasis: in.ResolvedCostBasis,
		Totals:            reconciledTotals,
	})
	if err != nil {
		return CustomCurrencyWithFiatOverageResult{}, fmt.Errorf("convert custom currency overage to fiat: %w", err)
	}

	fiatOverageResult, err := Reconcile(ctx, ReconcileInput{
		TargetAmount: fiatOverage.Amount,
		Handler:      in.FiatOverageHandler,
	})
	if err != nil {
		return CustomCurrencyWithFiatOverageResult{}, fmt.Errorf("reconcile fiat overage credits: %w", err)
	}

	fiatOverageRealizations := append(
		slices.Clone(in.FiatOverageHandler.Realizations()),
		fiatOverageResult.Realizations...,
	)
	fiatCreditsTotal := fiatOverage.Currency.RoundToPrecision(fiatOverageRealizations.Sum())
	if fiatCreditsTotal.GreaterThan(fiatOverage.Amount) {
		return CustomCurrencyWithFiatOverageResult{}, fmt.Errorf(
			"fiat overage credit allocations exceed converted overage [overage=%s allocated=%s]",
			fiatOverage.Amount,
			fiatCreditsTotal,
		)
	}

	return CustomCurrencyWithFiatOverageResult{
		Totals:               reconciledTotals,
		ChargeCurrency:       chargeCurrencyResult,
		ConvertedFiatOverage: fiatOverage,
		FiatOverageCredits:   fiatOverageResult,
		RemainingFiatOverage: fiatOverage.Currency.RoundToPrecision(fiatOverage.Amount.Sub(fiatCreditsTotal)),
	}, nil
}
