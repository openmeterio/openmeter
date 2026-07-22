package currencyresolver

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func New(service currencies.Service) (currencies.CurrencyResolver, error) {
	if service == nil {
		return nil, errors.New("currency service is required")
	}

	return &resolver{
		service: service,
	}, nil
}

var _ currencies.NamespacedCurrencyResolver = (*namespacedResolver)(nil)

type namespacedResolver struct {
	resolver  *resolver
	namespace string
}

func (n *namespacedResolver) Namespace() string {
	return n.namespace
}

func (n *namespacedResolver) ResolveCurrency(ctx context.Context, ref currencies.CurrencyRef) (*currencies.Currency, error) {
	return n.resolver.ResolveCurrency(ctx, n.namespace, ref)
}

func (n *namespacedResolver) BatchResolveCurrencies(ctx context.Context, refs ...currencies.CurrencyRef) (map[currencies.CurrencyRef]*currencies.Currency, error) {
	return n.resolver.BatchResolveCurrencies(ctx, n.namespace, refs...)
}

var _ currencies.CurrencyResolver = (*resolver)(nil)

type resolver struct {
	service currencies.Service
}

func (r *resolver) WithNamespace(namespace string) currencies.NamespacedCurrencyResolver {
	return &namespacedResolver{
		resolver:  r,
		namespace: namespace,
	}
}

func (r *resolver) ResolveCurrency(ctx context.Context, namespace string, ref currencies.CurrencyRef) (*currencies.Currency, error) {
	if err := ref.Validate(); err != nil {
		return nil, err
	}

	resolved, err := r.BatchResolveCurrencies(ctx, namespace, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch currency: %w", err)
	}

	currency := resolved[ref]
	if currency != nil {
		return currency, nil
	}

	if ref.ID != "" {
		return nil, models.NewGenericNotFoundError(fmt.Errorf("currency [currency.id=%s]", ref.ID))
	}

	return nil, models.NewGenericNotFoundError(fmt.Errorf("currency [currency.code=%s]", ref.Code))
}

func (r *resolver) BatchResolveCurrencies(ctx context.Context, namespace string, refs ...currencies.CurrencyRef) (map[currencies.CurrencyRef]*currencies.Currency, error) {
	if namespace == "" {
		return nil, errors.New("namespace is not set")
	}

	if len(refs) == 0 {
		return nil, nil
	}

	ids := make([]string, 0, len(refs))
	codes := make([]string, 0, len(refs))
	for idx, ref := range refs {
		if err := ref.Validate(); err != nil {
			return nil, fmt.Errorf("invalid currency reference at index %d: %w", idx, err)
		}

		if ref.ID != "" {
			ids = append(ids, ref.ID)
		} else {
			codes = append(codes, ref.Code.String())
		}
	}
	ids = lo.Uniq(ids)
	codes = lo.Uniq(codes)

	var idFilter *filter.FilterString
	if len(ids) > 0 {
		idFilter = &filter.FilterString{In: &ids}
	}

	var codeFilter *filter.FilterString
	if len(codes) > 0 {
		codeFilter = &filter.FilterString{In: &codes}
	}

	items, err := pagination.CollectAll(ctx, pagination.NewPaginator(func(ctx context.Context, page pagination.Page) (pagination.Result[currencies.Currency], error) {
		return r.service.ListCurrencies(ctx, currencies.ListCurrenciesInput{
			Page: page,
			FilteringOptions: currencies.FilteringOptions{
				Union: true,
			},
			CurrencyExpandOptions: currencies.CurrencyExpandOptions{
				CostBasis: true,
			},
			Namespace: namespace,
			ID:        idFilter,
			Code:      codeFilter,
		})
	}), min(len(ids)+len(codes), 100))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch currencies: %w", err)
	}

	byID := make(map[string]*currencies.Currency, len(items))
	byCode := make(map[string]*currencies.Currency, len(items))

	for idx := range items {
		currency := &items[idx]

		if currency.ID != "" {
			byID[currency.ID] = currency
		}

		if currency.DeletedAt == nil {
			code := currency.Details().Code.String()
			if _, ok := byCode[code]; ok {
				return nil, fmt.Errorf("multiple active currencies found for code %q", code)
			}

			byCode[code] = currency
		}
	}

	result := make(map[currencies.CurrencyRef]*currencies.Currency, len(refs))
	for _, ref := range refs {
		if ref.ID != "" {
			result[ref] = byID[ref.ID]
		} else {
			result[ref] = byCode[ref.Code.String()]
		}
	}

	return result, nil
}
