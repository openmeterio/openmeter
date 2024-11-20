package currencyx

import (
	"errors"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
)

// Currency represents a currency code.
// Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html) currency code.
type Code currency.Code

func (c Code) Validate() error {
	if c == "" {
		return errors.New("currency code is required")
	}

	return currency.Code(c).Validate()
}

// Calculator provides a currency calculator object. This allows us to not to resolve def a lot of times, plus
// we can assume that def is always valid, thus we can avoid a lot of error handling.
func (c Code) Calculator() (Calculator, error) {
	if err := c.Validate(); err != nil {
		return Calculator{}, err
	}

	return Calculator{
		Currency: c,
		Def:      currency.Get(currency.Code(c)),
	}, nil
}

// TODO: Better name?!
type Calculator struct {
	Currency Code
	Def      *currency.Def
}

func (c Calculator) RoundToPrecision(amount alpacadecimal.Decimal) alpacadecimal.Decimal {
	// TODO: For now we are skipping the smallestDenomination, as that is a reference to the coins
	// in circulation, but should not be an issue for online payments.
	return amount.Round(int32(c.Def.Subunits))
}
