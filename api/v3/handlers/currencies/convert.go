package currencies

import (
	"fmt"

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

func NewBillingCurrencyFrom[T v3.BillingCurrencyCustom | v3.BillingCurrencyFiat](v T) (v3.BillingCurrency, error) {
	c := v3.BillingCurrency{}
	switch any(v).(type) {
	case v3.BillingCurrencyCustom:
		custom := any(v).(v3.BillingCurrencyCustom)
		if err := c.FromBillingCurrencyCustom(custom); err != nil {
			return c, fmt.Errorf("failed to construct BillingCurrencyCustom: %w", err)
		}
	case v3.BillingCurrencyFiat:
		fiat := any(v).(v3.BillingCurrencyFiat)
		if err := c.FromBillingCurrencyFiat(fiat); err != nil {
			return c, fmt.Errorf("failed to construct BillingCurrencyFiat: %w", err)
		}
	}
	return c, nil
}

func CurrencyToAPI(c currencies.Currency) (v3.BillingCurrency, error) {
	if c.ID != "" {
		return NewBillingCurrencyFrom(v3.BillingCurrencyCustom{
			Id:        c.ID,
			Code:      c.Code,
			Name:      c.Name,
			Symbol:    &c.Symbol,
			Type:      v3.BillingCurrencyCustomTypeCustom,
			CreatedAt: &c.CreatedAt,
		})
	}
	return NewBillingCurrencyFrom(v3.BillingCurrencyFiat{
		Code:   c.Code,
		Name:   c.Name,
		Symbol: &c.Symbol,
		Type:   v3.BillingCurrencyFiatTypeFiat,
	})
}

func CostBasisToAPI(cb currencies.CostBasis) v3.BillingCostBasis {
	return v3.BillingCostBasis{
		Id:            cb.ID,
		FiatCode:      cb.FiatCode,
		Rate:          cb.Rate.String(),
		EffectiveFrom: &cb.EffectiveFrom,
		CreatedAt:     &cb.CreatedAt,
	}
}
