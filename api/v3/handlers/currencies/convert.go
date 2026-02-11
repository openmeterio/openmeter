package currencies

import (
	"github.com/invopop/gobl/currency"
	v3 "github.com/openmeterio/openmeter/api/v3"
)

func MapCurrencyToAPI(currency *currency.Def) v3.BillingCurrency {
	return v3.BillingCurrency{
		Code:                 v3.CurrencyCode(currency.ISOCode),
		DisambiguateSymbol:   currency.DisambiguateSymbol,
		Name:                 currency.Name,
		SmallestDenomination: uint8(currency.SmallestDenomination),
		Subunits:             currency.Subunits,
		Symbol:               currency.Symbol,
	}
}
