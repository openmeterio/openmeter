package meta

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type ConvertCustomCurrencyOverageToFiatInput struct {
	Currency          currencies.Currency
	CostBasisIntent   *costbasis.Intent
	ResolvedCostBasis *costbasis.State
	Totals            totals.Totals
}

type FiatOverage struct {
	Currency *currencyx.FiatCurrency
	Amount   alpacadecimal.Decimal
}

// CalculateFiatAmount converts an amount using a resolved cost basis and
// rounds the result to the settlement currency's precision.
func CalculateFiatAmount(amount, resolvedCostBasis alpacadecimal.Decimal, fiatCurrency *currencyx.FiatCurrency) (alpacadecimal.Decimal, error) {
	if amount.IsNegative() {
		return alpacadecimal.Zero, fmt.Errorf("amount cannot be negative")
	}

	if resolvedCostBasis.IsZero() {
		return alpacadecimal.Zero, fmt.Errorf("resolved cost basis cannot be zero")
	}

	return fiatCurrency.RoundToPrecision(amount.Mul(resolvedCostBasis)), nil
}

// ConvertCustomCurrencyOverageToFiat converts the post-allocation total of a
// custom-currency realization into its invoice currency using the persisted
// cost basis.
func ConvertCustomCurrencyOverageToFiat(input ConvertCustomCurrencyOverageToFiatInput) (FiatOverage, error) {
	if err := input.Currency.Validate(); err != nil {
		return FiatOverage{}, fmt.Errorf("currency: %w", err)
	}

	if !input.Currency.IsCustom() {
		return FiatOverage{}, fmt.Errorf("currency must be custom")
	}

	if err := input.Totals.ValidateTotalNonNegative(); err != nil {
		return FiatOverage{}, fmt.Errorf("totals: %w", err)
	}

	if !input.Currency.IsRoundedToPrecision(input.Totals.Total) {
		return FiatOverage{}, fmt.Errorf("totals total must be rounded to custom currency precision")
	}

	if input.CostBasisIntent == nil {
		return FiatOverage{}, fmt.Errorf("cost basis intent is required")
	}

	if err := input.CostBasisIntent.Validate(); err != nil {
		return FiatOverage{}, fmt.Errorf("cost basis intent: %w", err)
	}

	fiatCurrency, err := input.CostBasisIntent.GetFiatCurrency()
	if err != nil {
		return FiatOverage{}, fmt.Errorf("get cost basis fiat currency: %w", err)
	}

	if input.ResolvedCostBasis == nil {
		return FiatOverage{}, fmt.Errorf("resolved cost basis is required")
	}

	if err := input.ResolvedCostBasis.Validate(); err != nil {
		return FiatOverage{}, fmt.Errorf("resolved cost basis: %w", err)
	}

	fiatAmount, err := CalculateFiatAmount(
		input.Totals.Total,
		input.ResolvedCostBasis.CostBasis,
		fiatCurrency,
	)
	if err != nil {
		return FiatOverage{}, fmt.Errorf("calculate fiat amount: %w", err)
	}

	return FiatOverage{
		Currency: fiatCurrency,
		Amount:   fiatAmount,
	}, nil
}
