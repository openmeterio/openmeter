package currencies

import (
	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/currencies"
)

func MapCurrencyToAPI(currency currencies.Currency) v3.BillingCurrency {
	return v3.BillingCurrency{
		Id:                   currency.ID,
		Code:                 v3.CurrencyCode(currency.Code),
		DisambiguateSymbol:   currency.DisambiguateSymbol,
		Name:                 currency.Name,
		SmallestDenomination: uint8(currency.SmallestDenomination),
		Subunits:             currency.Subunits,
		Symbol:               currency.Symbol,
		IsCustom:             currency.IsCustom,
	}
}
