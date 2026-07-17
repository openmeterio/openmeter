package currencyresolver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type mockCurrencyService struct {
	currencies []currencies.Currency
}

func (m mockCurrencyService) ListCurrencies(_ context.Context, params currencies.ListCurrenciesInput) (pagination.Result[currencies.Currency], error) {
	return pagination.Result[currencies.Currency]{
		Page:       params.Page,
		TotalCount: len(m.currencies),
		Items:      m.currencies,
	}, nil
}

func (m mockCurrencyService) CreateCurrency(context.Context, currencies.CreateCurrencyInput) (currencies.Currency, error) {
	return currencies.Currency{}, errors.New("not implemented")
}

func (m mockCurrencyService) CreateCostBasis(context.Context, currencies.CreateCostBasisInput) (currencies.CostBasis, error) {
	return currencies.CostBasis{}, errors.New("not implemented")
}

func (m mockCurrencyService) ListCostBases(context.Context, currencies.ListCostBasesInput) (pagination.Result[currencies.CostBasis], error) {
	return pagination.Result[currencies.CostBasis]{}, errors.New("not implemented")
}

func (m mockCurrencyService) GetCostBasisAt(context.Context, currencies.GetCostBasisAtInput) (currencies.CostBasis, error) {
	return currencies.CostBasis{}, errors.New("not implemented")
}

func TestResolverResolveCustomCurrencyIgnoresArchivedResources(t *testing.T) {
	deletedAt := time.Now()
	service := mockCurrencyService{
		currencies: []currencies.Currency{
			{
				ManagedModel: models.ManagedModel{DeletedAt: &deletedAt},
				NamespacedID: models.NamespacedID{Namespace: "ns", ID: "archived"},
				Code:         "CREDITS",
			},
			{
				NamespacedID: models.NamespacedID{Namespace: "ns", ID: "active"},
				Code:         "CREDITS",
			},
		},
	}

	resolver, err := New(service)
	require.NoError(t, err)

	identity, err := resolver.Resolve(t.Context(), "ns", currencyx.Code("CREDITS"))
	require.NoError(t, err)

	managed, ok := identity.(currencyx.ManagedCurrency)
	assert.True(t, ok)
	assert.Equal(t, "active", managed.GetID())
}

func TestResolverResolveCustomCurrencyRejectsArchivedResource(t *testing.T) {
	deletedAt := time.Now()
	service := mockCurrencyService{
		currencies: []currencies.Currency{
			{
				ManagedModel: models.ManagedModel{DeletedAt: &deletedAt},
				NamespacedID: models.NamespacedID{Namespace: "ns", ID: "archived"},
				Code:         "CREDITS",
			},
		},
	}

	resolver, err := New(service)
	require.NoError(t, err)

	_, err = resolver.Resolve(t.Context(), "ns", currencyx.Code("CREDITS"))
	require.Error(t, err)
	assert.True(t, models.IsGenericNotFoundError(err))
}
