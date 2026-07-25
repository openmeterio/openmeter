package currencies

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
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

	var err error

	switch c.Type() {
	case currencyx.CurrencyTypeCustom:
		err = out.FromBillingCurrencyCustom(v3.BillingCurrencyCustom{
			Code: v3.CurrencyCode(c.Details().Code),
			CostBasis: func() *[]v3.BillingCostBasis {
				if c.CostBasis != nil {
					return lo.ToPtr(lo.Map(*c.CostBasis, func(item currencies.CostBasis, index int) v3.BillingCostBasis {
						return ToAPIBillingCostBasis(item)
					}))
				}

				return nil
			}(),
			CreatedAt:         c.CreatedAt,
			DecimalMark:       c.Details().DecimalMark,
			Id:                c.ID,
			Name:              c.Details().Name,
			Precision:         c.Details().Precision,
			Symbol:            lo.EmptyableToPtr(c.Details().Symbol),
			ThousandSeparator: c.Details().ThousandsSeparator,
			Type:              v3.BillingCurrencyCustomTypeCustom,
		})

		return out, err
	case currencyx.CurrencyTypeFiat:
		err = out.FromBillingCurrencyFiat(v3.BillingCurrencyFiat{
			Code:              v3.CurrencyCode(c.Details().Code),
			DecimalMark:       c.Details().DecimalMark,
			Name:              c.Details().Name,
			Precision:         c.Details().Precision,
			Symbol:            lo.EmptyableToPtr(c.Details().Symbol),
			ThousandSeparator: c.Details().ThousandsSeparator,
			Type:              v3.BillingCurrencyFiatTypeFiat,
		})

		return out, err
	default:
		return v3.BillingCurrency{}, fmt.Errorf("unsupported currency type: %s", c.Type())
	}
}

func ToAPIBillingCostBasis(cb currencies.CostBasis) v3.BillingCostBasis {
	return v3.BillingCostBasis{
		Id:            cb.ID,
		FiatCode:      v3.CurrencyCode(cb.FiatCode),
		Rate:          cb.Rate.String(),
		EffectiveFrom: &cb.EffectiveFrom,
		EffectiveTo:   cb.EffectiveTo,
		CreatedAt:     cb.CreatedAt,
	}
}
