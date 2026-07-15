package currencyresolver

import (
	"context"
	"errors"
	"fmt"

	"github.com/invopop/gobl/currency"
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

func (r *resolver) Resolve(ctx context.Context, namespace string, code currency.Code) (productcatalog.ResolvedCurrency, error) {
	if namespace == "" {
		return productcatalog.ResolvedCurrency{}, errors.New("namespace is required")
	}

	if err := currencyx.Code(code).Validate(); err != nil {
		return productcatalog.ResolvedCurrency{}, models.NewGenericValidationError(fmt.Errorf("invalid currency code: %w", err))
	}

	if currencyx.Code(code).IsFiat() {
		return productcatalog.ResolvedCurrency{
			Code: code,
			Type: currencyx.CurrencyTypeFiat,
		}, nil
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
		return productcatalog.ResolvedCurrency{}, fmt.Errorf("listing custom currencies: %w", err)
	}

	if len(result.Items) == 0 {
		return productcatalog.ResolvedCurrency{}, models.NewGenericNotFoundError(fmt.Errorf("currency %q", code))
	}

	if len(result.Items) > 1 {
		return productcatalog.ResolvedCurrency{}, fmt.Errorf("multiple custom currencies found for code %q", code)
	}

	return productcatalog.ResolvedCurrency{
		ID:   result.Items[0].ID,
		Code: code,
		Type: currencyx.CurrencyTypeCustom,
	}, nil
}

func (r *resolver) HasCostBasis(ctx context.Context, namespace string, customCurrency productcatalog.ResolvedCurrency, fiatCurrency currency.Code) (bool, error) {
	if customCurrency.Type != currencyx.CurrencyTypeCustom || customCurrency.ID == "" {
		return false, errors.New("custom currency with managed resource ID is required")
	}

	if !currencyx.Code(fiatCurrency).IsFiat() {
		return false, errors.New("fiat currency is required")
	}

	fiatCode := fiatCurrency.String()
	result, err := r.service.ListCostBases(ctx, currencies.ListCostBasesInput{
		Page:           pagination.NewPage(1, 1),
		Namespace:      namespace,
		CurrencyID:     customCurrency.ID,
		FilterFiatCode: &fiatCode,
	})
	if err != nil {
		return false, fmt.Errorf("listing cost bases: %w", err)
	}

	return len(result.Items) > 0, nil
}

var _ productcatalog.CurrencyResolver = (*resolver)(nil)
