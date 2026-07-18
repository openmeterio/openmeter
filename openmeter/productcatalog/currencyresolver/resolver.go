package currencyresolver

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type resolver struct {
	service currencies.Service
}

func New(service currencies.Service) (productcatalog.CurrencyResolver, error) {
	if service == nil {
		return nil, errors.New("currency service is required")
	}

	return &resolver{service: service}, nil
}

func (r *resolver) Resolve(ctx context.Context, namespace string, code currencyx.Code) (currencyx.CurrencyIdentity, error) {
	if namespace == "" {
		return nil, errors.New("namespace is required")
	}

	if err := code.Validate(); err != nil {
		return nil, models.NewGenericValidationError(fmt.Errorf("invalid currency code: %w", err))
	}

	if code.IsFiat() {
		return currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
			WithCode(code).
			Build()
	}

	customType := currencies.CurrencyTypeCustom
	result, err := r.service.ListCurrencies(ctx, currencies.ListCurrenciesInput{
		Page:       pagination.NewPage(1, 2),
		Namespace:  namespace,
		FilterType: &customType,
		Code: &filter.FilterString{
			Eq: lo.ToPtr(code.String()),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("listing custom currencies: %w", err)
	}

	activeCurrencies := lo.Filter(result.Items, func(item currencies.Currency, _ int) bool {
		return item.DeletedAt == nil
	})

	if len(activeCurrencies) == 0 {
		return nil, models.NewGenericNotFoundError(fmt.Errorf("currency %q", code))
	}

	if len(activeCurrencies) > 1 {
		return nil, fmt.Errorf("multiple custom currencies found for code %q", code)
	}

	return activeCurrencies[0], nil
}

func (r *resolver) HasCostBasis(ctx context.Context, namespace string, customCurrency currencyx.ManagedCurrency, fiatCurrency currencyx.CurrencyIdentity) (bool, error) {
	if err := customCurrency.Validate(); err != nil {
		return false, fmt.Errorf("invalid custom currency: %w", err)
	}

	if !customCurrency.IsCustom() || customCurrency.GetID() == "" {
		return false, errors.New("custom currency with managed resource ID is required")
	}

	if fiatCurrency == nil || !fiatCurrency.IsFiat() {
		return false, errors.New("fiat currency is required")
	}

	fiatCode := fiatCurrency.GetCode().String()
	result, err := r.service.ListCostBases(ctx, currencies.ListCostBasesInput{
		Page:           pagination.NewPage(1, 1),
		Namespace:      namespace,
		CurrencyID:     customCurrency.GetID(),
		FilterFiatCode: &fiatCode,
	})
	if err != nil {
		return false, fmt.Errorf("listing cost bases: %w", err)
	}

	return len(result.Items) > 0, nil
}

var _ productcatalog.CurrencyResolver = (*resolver)(nil)
