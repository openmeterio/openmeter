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

	fiatCurrency := input.CostBasisIntent.GetFiatCurrency()
	if fiatCurrency == nil {
		return FiatOverage{}, fmt.Errorf("cost basis fiat currency is required")
	}

	if input.ResolvedCostBasis == nil {
		return FiatOverage{}, fmt.Errorf("resolved cost basis is required")
	}

	if err := input.ResolvedCostBasis.Validate(); err != nil {
		return FiatOverage{}, fmt.Errorf("resolved cost basis: %w", err)
	}

	return FiatOverage{
		Currency: fiatCurrency,
		Amount: fiatCurrency.RoundToPrecision(
			input.Totals.Total.Mul(input.ResolvedCostBasis.CostBasis),
		),
	}, nil
}
