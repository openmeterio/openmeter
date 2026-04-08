package invoicesync

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/num"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

var printer = message.NewPrinter(language.English)

// RoundToAmount converts a decimal amount to Stripe's integer representation for the given currency.
func RoundToAmount(amount alpacadecimal.Decimal, currency string) (int64, error) {
	calc, err := currencyx.Code(currency).Calculator()
	if err != nil {
		return 0, fmt.Errorf("invalid currency %q: %w", currency, err)
	}

	multiplier := alpacadecimal.NewFromInt(10).Pow(alpacadecimal.NewFromInt(int64(calc.Def.Subunits)))
	return amount.Mul(multiplier).Round(0).IntPart(), nil
}

// FormatAmount formats a decimal amount for display in the given currency.
// It rounds to the currency's minor-unit precision first so the displayed value
// matches what RoundToAmount (and therefore Stripe) will actually charge.
func FormatAmount(amount alpacadecimal.Decimal, currency string) (string, error) {
	calc, err := currencyx.Code(currency).Calculator()
	if err != nil {
		return "", fmt.Errorf("invalid currency %q: %w", currency, err)
	}

	multiplier := alpacadecimal.NewFromInt(10).Pow(alpacadecimal.NewFromInt(int64(calc.Def.Subunits)))
	minorUnitAmount := amount.Mul(multiplier).Round(0).IntPart()

	return calc.Def.FormatAmount(num.MakeAmount(minorUnitAmount, calc.Def.Subunits)), nil
}

// FormatQuantity formats a quantity for display.
func FormatQuantity(quantity alpacadecimal.Decimal, _ string) string {
	if quantity.IsInteger() {
		return printer.Sprintf("%d", quantity.IntPart())
	}

	f, _ := quantity.Float64()
	return printer.Sprintf("%.2f", f)
}
