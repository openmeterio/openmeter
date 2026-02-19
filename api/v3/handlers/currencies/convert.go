package currencies

import (
	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/currencies"
)

func MapCurrencyToAPI(currency currencies.Currency) v3.BillingCurrency {
	return v3.BillingCurrency{
		Id:       currency.ID,
		Code:     currency.Code,
		Name:     currency.Name,
		Symbol:   &currency.Symbol,
		IsCustom: currency.IsCustom,
	}
}
