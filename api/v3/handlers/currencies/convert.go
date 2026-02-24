package currencies

import (
	"fmt"

	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/samber/lo"
)

func MapCostBasisToAPI(cb currencies.CostBasis) v3.BillingCostBasis {
	effectiveFrom := v3.DateTime(cb.EffectiveFrom)
	return v3.BillingCostBasis{
		Id:            cb.ID,
		FiatCode:      cb.FiatCode,
		Rate:          cb.Rate.String(),
		EffectiveFrom: lo.ToPtr(effectiveFrom),
	}
}

func MapCurrencyTypeFromAPI(t v3.BillingCurrencyType) currencies.CurrencyType {
	switch t {
	case v3.BillingCurrencyTypeCustom:
		return currencies.CurrencyTypeCustom
	default:
		return currencies.CurrencyTypeFiat
	}
}

func MapCurrencyToAPI(currency currencies.Currency) v3.BillingCurrency {
	var result v3.BillingCurrency

	if currency.IsCustom {
		if err := result.FromBillingCurrencyCustom(v3.BillingCurrencyCustom{
			Id:     currency.ID,
			Code:   currency.Code,
			Name:   currency.Name,
			Symbol: &currency.Symbol,
			Type:   v3.BillingCurrencyCustomTypeCustom,
		}); err != nil {
			panic(fmt.Sprintf("failed to construct BillingCurrencyCustom: %v", err))
		}
	} else {
		if err := result.FromBillingCurrencyFiat(v3.BillingCurrencyFiat{
			Id:     currency.ID,
			Code:   currency.Code,
			Name:   currency.Name,
			Symbol: &currency.Symbol,
			Type:   v3.BillingCurrencyFiatTypeFiat,
		}); err != nil {
			panic(fmt.Sprintf("failed to construct BillingCurrencyFiat: %v", err))
		}
	}

	return result
}
