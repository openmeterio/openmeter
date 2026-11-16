package currencies

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
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

func FromAPIBillingCurrencyType(t apiv3models.BillingCurrencyType) currencyx.CurrencyType {
	switch t {
	case apiv3models.BillingCurrencyTypeCustom:
		return currencyx.CurrencyTypeCustom
	default:
		return currencyx.CurrencyTypeFiat
	}
}

func ToAPIBillingCurrency(c currencies.Currency) (apiv3models.BillingCurrency, error) {
	out := apiv3models.BillingCurrency{}

	var err error

	switch c.Type() {
	case currencyx.CurrencyTypeCustom:
		err = out.FromBillingCurrencyCustom(apiv3models.BillingCurrencyCustom{
			Code: apiv3models.CurrencyCode(c.Details().Code),
			CostBasis: func() *[]apiv3models.BillingCostBasis {
				if c.CostBasis != nil {
					return lo.ToPtr(lo.Map(*c.CostBasis, func(item currencies.CostBasis, index int) apiv3models.BillingCostBasis {
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
			Type:              apiv3models.BillingCurrencyCustomTypeCustom,
		})

		return out, err
	case currencyx.CurrencyTypeFiat:
		err = out.FromBillingCurrencyFiat(apiv3models.BillingCurrencyFiat{
			Code:              apiv3models.CurrencyCode(c.Details().Code),
			DecimalMark:       c.Details().DecimalMark,
			Name:              c.Details().Name,
			Precision:         c.Details().Precision,
			Symbol:            lo.EmptyableToPtr(c.Details().Symbol),
			ThousandSeparator: c.Details().ThousandsSeparator,
			Type:              apiv3models.BillingCurrencyFiatTypeFiat,
		})

		return out, err
	default:
		return apiv3models.BillingCurrency{}, fmt.Errorf("unsupported currency type: %s", c.Type())
	}
}

func ToAPIBillingCostBasis(cb currencies.CostBasis) apiv3models.BillingCostBasis {
	return apiv3models.BillingCostBasis{
		Id:            cb.ID,
		FiatCode:      apiv3models.CurrencyCode(cb.FiatCode),
		Rate:          cb.Rate.String(),
		EffectiveFrom: &cb.EffectiveFrom,
		EffectiveTo:   cb.EffectiveTo,
		CreatedAt:     cb.CreatedAt,
	}
}
