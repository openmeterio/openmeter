package testutils

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type CurrencyResolverStub struct {
	Resolved map[currencyx.Code]currencyx.CurrencyIdentity
}

func (r *CurrencyResolverStub) ResolveCurrency(_ context.Context, _ string, ref currencies.CurrencyRef) (*currencies.Currency, error) {
	identity, ok := r.Resolved[ref.Code]
	if !ok {
		return nil, fmt.Errorf("unexpected currency resolution: %s", ref.Code)
	}

	return currencyFromIdentity(identity)
}

func (r *CurrencyResolverStub) BatchResolveCurrencies(ctx context.Context, namespace string, refs ...currencies.CurrencyRef) (map[currencies.CurrencyRef]*currencies.Currency, error) {
	resolved := make(map[currencies.CurrencyRef]*currencies.Currency, len(refs))
	for _, ref := range refs {
		currency, err := r.ResolveCurrency(ctx, namespace, ref)
		if err != nil {
			return nil, err
		}

		resolved[ref] = currency
	}

	return resolved, nil
}

func (r *CurrencyResolverStub) WithNamespace(namespace string) currencies.NamespacedCurrencyResolver {
	return &namespacedCurrencyResolverStub{resolver: r, namespace: namespace}
}

type namespacedCurrencyResolverStub struct {
	resolver  *CurrencyResolverStub
	namespace string
}

func (r *namespacedCurrencyResolverStub) ResolveCurrency(ctx context.Context, ref currencies.CurrencyRef) (*currencies.Currency, error) {
	return r.resolver.ResolveCurrency(ctx, r.namespace, ref)
}

func (r *namespacedCurrencyResolverStub) BatchResolveCurrencies(ctx context.Context, refs ...currencies.CurrencyRef) (map[currencies.CurrencyRef]*currencies.Currency, error) {
	return r.resolver.BatchResolveCurrencies(ctx, r.namespace, refs...)
}

func (r *namespacedCurrencyResolverStub) Namespace() string {
	return r.namespace
}

func currencyFromIdentity(identity currencyx.CurrencyIdentity) (*currencies.Currency, error) {
	switch value := identity.(type) {
	case currencies.Currency:
		return &value, nil
	case *currencies.Currency:
		return value, nil
	case currencyx.Currency:
		return &currencies.Currency{Currency: value}, nil
	default:
		currencyType := currencyx.CurrencyTypeCustom
		if identity.IsFiat() {
			currencyType = currencyx.CurrencyTypeFiat
		}

		currency, err := currencyx.NewCurrencyBuilder(currencyType).
			WithCode(identity.GetCode()).
			WithName(identity.GetCode().String()).
			Build()
		if err != nil {
			return nil, err
		}

		return &currencies.Currency{Currency: currency}, nil
	}
}

var _ currencies.CurrencyResolver = (*CurrencyResolverStub)(nil)
