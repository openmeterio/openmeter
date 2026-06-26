package currencies

import (
	"context"

	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/currencies"
)

func FromAPICurrencySortField(ctx context.Context, field string) (currencies.OrderBy, error) {
	switch field {
	case "code":
		return currencies.OrderByCode, nil
	case "name":
		return currencies.OrderByName, nil
	default:
		return "", apierrors.NewUnsupportedSortFieldError(ctx, field, "code", "name")
	}
}

func FromAPIBillingCurrencyType(t v3.BillingCurrencyType) currencies.CurrencyType {
	switch t {
	case v3.BillingCurrencyTypeCustom:
		return currencies.CurrencyTypeCustom
	default:
		return currencies.CurrencyTypeFiat
	}
}

func ToAPIBillingCurrency(c currencies.Currency) (v3.BillingCurrency, error) {
	out := v3.BillingCurrency{}

	if c.ID != "" {
		err := out.FromBillingCurrencyCustom(v3.BillingCurrencyCustom{
			Id:        c.ID,
			Code:      c.Code,
			Name:      c.Name,
			Symbol:    &c.Symbol,
			Type:      v3.BillingCurrencyCustomTypeCustom,
			CreatedAt: c.CreatedAt,
		})
		return out, err
	}

	err := out.FromBillingCurrencyFiat(v3.BillingCurrencyFiat{
		Code:   c.Code,
		Name:   c.Name,
		Symbol: &c.Symbol,
		Type:   v3.BillingCurrencyFiatTypeFiat,
	})
	return out, err
}

func ToAPIBillingCostBasis(cb currencies.CostBasis) v3.BillingCostBasis {
	return v3.BillingCostBasis{
		Id:            cb.ID,
		FiatCode:      cb.FiatCode,
		Rate:          cb.Rate.String(),
		EffectiveFrom: &cb.EffectiveFrom,
		EffectiveTo:   cb.EffectiveTo,
		CreatedAt:     cb.CreatedAt,
	}
}
