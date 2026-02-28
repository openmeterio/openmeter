package currencies

import (
	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/currencies"
)

func MapCurrencyTypeFromAPI(t v3.BillingCurrencyType) currencies.CurrencyType {
	switch t {
	case v3.BillingCurrencyTypeCustom:
		return currencies.CurrencyTypeCustom
	default:
		return currencies.CurrencyTypeFiat
	}
}


