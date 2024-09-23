package currencyx

import (
	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
)

// Currency represents a currency code.
// Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html) currency code.
type Code currency.Code

func (c Code) RoundToPrecision(amount alpacadecimal.Decimal) alpacadecimal.Decimal {
	// TODO[OM-907]: find a library to handle currency codes and precisions. (e.g. JPY has a precision of 0)
	return amount.Round(2)
}

/**
package main

import (
	"fmt"

	"github.com/invopop/gobl/currency"
)

func main() {
	currencies := currency.Definitions()

	fmt.Println(currency.Get("USD"))

	fmt.Println(currencies)
*/
