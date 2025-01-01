package appstripeentityapp

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/num"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

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
		printer:    message.NewPrinter(language.English),
		multiplier: alpacadecimal.NewFromInt(10).Pow(alpacadecimal.NewFromInt(int64(calculator.Def.Subunits))),
	}, nil
}

// StripeCalculator provides a currency calculator object.
type StripeCalculator struct {
	calculator currencyx.Calculator
	printer    *message.Printer
	multiplier alpacadecimal.Decimal
}

// RoundToAmount rounds the amount to the precision of the Stripe currency in Stripe amount.
func (c StripeCalculator) RoundToAmount(amount alpacadecimal.Decimal) int64 {
	return amount.Mul(c.multiplier).Round(0).IntPart()
}

// FormatAmount formats the amount
func (c StripeCalculator) FormatAmount(amount alpacadecimal.Decimal) string {
	if amount.IsInteger() {
		return c.calculator.Def.FormatAmount(num.MakeAmount(amount.IntPart(), 0))
	}

	am, _ := amount.Float64()
	return c.calculator.Def.FormatAmount(num.AmountFromFloat64(am, uint32(amount.NumDigits())))
}

// FormatQuantity formats the quantity to two decimal places.
// This should be only used to display the quantity not for calculations.
func (c StripeCalculator) FormatQuantity(quantity alpacadecimal.Decimal) string {
	if quantity.IsInteger() {
		return c.printer.Sprintf("%d", quantity.IntPart())
	} else {
		f, _ := quantity.Float64()
		return c.printer.Sprintf("%.2f", f)
	}
}

// IsInteger checks if the amount is an integer in the Stripe currency.
func (c StripeCalculator) IsInteger(amount alpacadecimal.Decimal) bool {
	return amount.Mul(c.multiplier).IsInteger()
}
