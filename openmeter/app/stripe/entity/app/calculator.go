package appstripeentityapp

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// NewStripeCalculator creates a new StripeCalculator.
func NewStripeCalculator(currency currencyx.Code) (StripeCalculator, error) {
	calculator, err := currency.Calculator()
	if err != nil {
		return StripeCalculator{}, fmt.Errorf("failed to get stripe calculator: %w", err)
	}

	return StripeCalculator{
		calculator: calculator,
		multiplier: alpacadecimal.NewFromInt(10).Pow(alpacadecimal.NewFromInt(int64(calculator.Def.Subunits))),
	}, nil
}

// StripeCalculator provides a currency calculator object.
type StripeCalculator struct {
	calculator currencyx.Calculator
	multiplier alpacadecimal.Decimal
}

// RoundToAmount rounds the amount to the precision of the Stripe currency in Stripe amount.
func (c StripeCalculator) RoundToAmount(amount alpacadecimal.Decimal) int64 {
	return amount.Mul(c.multiplier).Round(0).IntPart()
}

// IsInteger checks if the amount is an integer in the Stripe currency.
func (c StripeCalculator) IsInteger(amount alpacadecimal.Decimal) bool {
	return amount.Mul(c.multiplier).IsInteger()
}
