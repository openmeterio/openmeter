package currencies

import (
	"errors"
	"fmt"

	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/filters"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/models"
)

// FromAPICurrencyCodeFilter converts an API StringFieldFilterExact for the
// currency code into a flat []string of codes to match (positive list).
// Only eq and oeq operators are supported; neq returns an error. Each value
// is validated for length (3–24 chars), matching the custom_currencies ent
// schema constraints (and also accepting fiat ISO codes which are 3 chars).
func FromAPICurrencyCodeFilter(f *filters.FilterStringExact) ([]string, error) {
	if f == nil {
		return nil, nil
	}
	if f.Neq != nil {
		return nil, errors.New("only eq and oeq operators are supported for currency code")
	}

	var codes []string
	if f.Eq != nil {
		codes = append(codes, *f.Eq)
	}
	codes = append(codes, f.Oeq...)

	if len(codes) == 0 {
		return nil, nil
	}

	var errs []error
	for _, code := range codes {
		if len(code) < 3 {
			errs = append(errs, fmt.Errorf("currency code must be at least 3 characters, got %q", code))
		} else if len(code) > 24 {
			errs = append(errs, fmt.Errorf("currency code must be at most 24 characters, got %q", code))
		}
	}
	if len(errs) > 0 {
		return nil, models.NewNillableGenericValidationError(errors.Join(errs...))
	}

	return codes, nil
}

func FromAPIBillingCurrencyType(t v3.BillingCurrencyType) currencies.CurrencyType {
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

func ToAPIBillingCurrency(c currencies.Currency) (v3.BillingCurrency, error) {
	if c.ID != "" {
		return NewBillingCurrencyFrom(v3.BillingCurrencyCustom{
			Id:        c.ID,
			Code:      c.Code,
			Name:      c.Name,
			Symbol:    &c.Symbol,
			Type:      v3.BillingCurrencyCustomTypeCustom,
			CreatedAt: c.CreatedAt,
		})
	}
	return NewBillingCurrencyFrom(v3.BillingCurrencyFiat{
		Code:   c.Code,
		Name:   c.Name,
		Symbol: &c.Symbol,
		Type:   v3.BillingCurrencyFiatTypeFiat,
	})
}

func ToAPIBillingCostBasis(cb currencies.CostBasis) v3.BillingCostBasis {
	return v3.BillingCostBasis{
		Id:            cb.ID,
		FiatCode:      cb.FiatCode,
		Rate:          cb.Rate.String(),
		EffectiveFrom: &cb.EffectiveFrom,
		CreatedAt:     cb.CreatedAt,
	}
}
