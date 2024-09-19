package currency

import "github.com/alpacahq/alpacadecimal"

// Currency represents a currency code.
type Currency string

func (c Currency) RoundToPrecision(amount alpacadecimal.Decimal) alpacadecimal.Decimal {
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
