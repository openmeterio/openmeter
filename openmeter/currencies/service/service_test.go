package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

// noopDriver implements transaction.Driver as a no-op for unit tests.
type noopDriver struct{}

func (noopDriver) Commit() error    { return nil }
func (noopDriver) Rollback() error  { return nil }
func (noopDriver) SavePoint() error { return nil }

// fakeAdapter implements currencies.Adapter for unit testing the service layer.
// ListCustomCurrencies applies the Code filter from params to simulate DB-level filtering.
type fakeAdapter struct {
	custom          []currencies.Currency
	createCostBasis func(context.Context, currencies.CreateCostBasisInput) (currencies.CostBasis, error)
}

func (f *fakeAdapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	return ctx, noopDriver{}, nil
}

func (f *fakeAdapter) ListCustomCurrencies(_ context.Context, params currencies.ListCurrenciesInput) (pagination.Result[currencies.Currency], error) {
	items := make([]currencies.Currency, 0, len(f.custom))
	for _, c := range f.custom {
		if ok, _ := params.Code.Match(c.Code); ok {
			items = append(items, c)
		}
	}
	return pagination.Result[currencies.Currency]{
		Items:      items,
		TotalCount: len(items),
		Page:       params.Page,
	}, nil
}

func (f *fakeAdapter) CreateCurrency(_ context.Context, _ currencies.CreateCurrencyInput) (currencies.Currency, error) {
	return currencies.Currency{}, errors.New("fakeAdapter.CreateCurrency is not implemented")
}

func (f *fakeAdapter) CreateCostBasis(ctx context.Context, input currencies.CreateCostBasisInput) (currencies.CostBasis, error) {
	if f.createCostBasis != nil {
		return f.createCostBasis(ctx, input)
	}

	return currencies.CostBasis{}, errors.New("fakeAdapter.CreateCostBasis is not implemented")
}

func (f *fakeAdapter) ListCostBases(_ context.Context, _ currencies.ListCostBasesInput) (pagination.Result[currencies.CostBasis], error) {
	return pagination.Result[currencies.CostBasis]{}, errors.New("fakeAdapter.ListCostBases is not implemented")
}

// newTestService creates a Service backed by a fake adapter seeded with custom currencies.
func newTestService(custom []currencies.Currency) *Service {
	return New(&fakeAdapter{custom: custom})
}

func TestListCurrencies_CombinedPath(t *testing.T) {
	customCurrency := currencies.Currency{
		Code:   "MYCUSTOM",
		Name:   "My Custom Currency",
		Symbol: lo.ToPtr("MC"),
	}

	svc := newTestService([]currencies.Currency{customCurrency})

	tests := []struct {
		name          string
		input         currencies.ListCurrenciesInput
		wantErr       bool
		assertResults func(t *testing.T, result pagination.Result[currencies.Currency])
	}{
		{
			name: "no filter no sort returns combined list sorted by code asc",
			input: currencies.ListCurrenciesInput{
				Namespace: "test",
				Page:      pagination.NewPage(1, 5),
			},
			assertResults: func(t *testing.T, result pagination.Result[currencies.Currency]) {
				t.Helper()
				require.Equal(t, 5, len(result.Items))
				for i := 1; i < len(result.Items); i++ {
					assert.LessOrEqual(t, result.Items[i-1].Code, result.Items[i].Code, "items should be sorted by code asc")
				}
			},
		},
		{
			name: "filter by single fiat code returns only that currency",
			input: currencies.ListCurrenciesInput{
				Namespace: "test",
				Code:      &filter.FilterString{Eq: lo.ToPtr("USD")},
			},
			assertResults: func(t *testing.T, result pagination.Result[currencies.Currency]) {
				t.Helper()
				require.Equal(t, 1, result.TotalCount)
				assert.Equal(t, "USD", result.Items[0].Code)
			},
		},
		{
			name: "filter by multiple fiat codes using In returns only those currencies",
			input: currencies.ListCurrenciesInput{
				Namespace: "test",
				Code:      &filter.FilterString{In: lo.ToPtr([]string{"USD", "EUR"})},
			},
			assertResults: func(t *testing.T, result pagination.Result[currencies.Currency]) {
				t.Helper()
				require.Equal(t, 2, result.TotalCount)
				codes := []string{result.Items[0].Code, result.Items[1].Code}
				assert.ElementsMatch(t, []string{"USD", "EUR"}, codes)
			},
		},
		{
			name: "filter by custom currency code returns only that custom currency",
			input: currencies.ListCurrenciesInput{
				Namespace: "test",
				Code:      &filter.FilterString{Eq: lo.ToPtr("MYCUSTOM")},
			},
			assertResults: func(t *testing.T, result pagination.Result[currencies.Currency]) {
				t.Helper()
				require.Equal(t, 1, result.TotalCount)
				assert.Equal(t, "MYCUSTOM", result.Items[0].Code)
			},
		},
		{
			name: "sort by name returns items sorted by name asc",
			input: currencies.ListCurrenciesInput{
				Namespace: "test",
				Code:      &filter.FilterString{In: lo.ToPtr([]string{"USD", "EUR", "GBP"})},
				OrderBy:   currencies.OrderByName,
			},
			assertResults: func(t *testing.T, result pagination.Result[currencies.Currency]) {
				t.Helper()
				require.Equal(t, 3, result.TotalCount)
				for i := 1; i < len(result.Items); i++ {
					assert.LessOrEqual(t, result.Items[i-1].Name, result.Items[i].Name, "items should be sorted by name asc")
				}
			},
		},
		{
			name: "sort by code desc returns items sorted by code descending",
			input: currencies.ListCurrenciesInput{
				Namespace: "test",
				Code:      &filter.FilterString{In: lo.ToPtr([]string{"USD", "EUR", "GBP"})},
				Order:     sortx.OrderDesc,
			},
			assertResults: func(t *testing.T, result pagination.Result[currencies.Currency]) {
				t.Helper()
				require.Equal(t, 3, result.TotalCount)
				for i := 1; i < len(result.Items); i++ {
					assert.GreaterOrEqual(t, result.Items[i-1].Code, result.Items[i].Code, "items should be sorted by code desc")
				}
			},
		},
		{
			name: "filter by Ne excludes a single code from combined results",
			input: currencies.ListCurrenciesInput{
				Namespace: "test",
				Code:      &filter.FilterString{Ne: lo.ToPtr("USD")},
				// Limit to known codes plus our custom one to make the assertion easy
				Page: pagination.NewPage(1, 5),
			},
			assertResults: func(t *testing.T, result pagination.Result[currencies.Currency]) {
				t.Helper()
				for _, item := range result.Items {
					assert.NotEqual(t, "USD", item.Code, "USD should be excluded")
				}
			},
		},
		{
			name: "invalid order by returns validation error",
			input: currencies.ListCurrenciesInput{
				Namespace: "test",
				OrderBy:   currencies.OrderBy("invalid"),
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.input.Namespace = "test"
			result, err := svc.ListCurrencies(t.Context(), tc.input)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tc.assertResults(t, result)
		})
	}
}

func TestListCurrencies_CustomOnlyPath(t *testing.T) {
	customCurrency := currencies.Currency{
		Code:   "MYCUSTOM",
		Name:   "My Custom Currency",
		Symbol: lo.ToPtr("MC"),
	}
	svc := newTestService([]currencies.Currency{customCurrency})

	t.Run("filter by type custom with code filter uses custom-only fast path", func(t *testing.T) {
		ft := currencies.CurrencyTypeCustom
		result, err := svc.ListCurrencies(t.Context(), currencies.ListCurrenciesInput{
			Namespace:  "test",
			FilterType: &ft,
			Code:       &filter.FilterString{Eq: lo.ToPtr("MYCUSTOM")},
		})
		require.NoError(t, err)
		require.Equal(t, 1, result.TotalCount)
		assert.Equal(t, "MYCUSTOM", result.Items[0].Code)
	})

	t.Run("filter by type custom returns no fiat currencies", func(t *testing.T) {
		ft := currencies.CurrencyTypeCustom
		result, err := svc.ListCurrencies(t.Context(), currencies.ListCurrenciesInput{
			Namespace:  "test",
			FilterType: &ft,
		})
		require.NoError(t, err)
		require.Equal(t, 1, result.TotalCount)
		assert.Equal(t, "MYCUSTOM", result.Items[0].Code)
	})
}

func TestCreateCostBasis_EffectiveTo(t *testing.T) {
	effectiveFrom := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	effectiveTo := effectiveFrom.Add(24 * time.Hour)

	var gotInput currencies.CreateCostBasisInput
	svc := New(&fakeAdapter{
		createCostBasis: func(_ context.Context, input currencies.CreateCostBasisInput) (currencies.CostBasis, error) {
			gotInput = input

			return currencies.CostBasis{
				NamespacedID: models.NamespacedID{
					ID:        "01K00000000000000000000000",
					Namespace: input.Namespace,
				},
				CurrencyID:    input.CurrencyID,
				FiatCode:      input.FiatCode,
				Rate:          input.Rate,
				EffectiveFrom: *input.EffectiveFrom,
				EffectiveTo:   input.EffectiveTo,
			}, nil
		},
	})

	result, err := svc.CreateCostBasis(t.Context(), currencies.CreateCostBasisInput{
		Namespace:     "test",
		CurrencyID:    "01J00000000000000000000000",
		FiatCode:      "USD",
		Rate:          alpacadecimal.RequireFromString("0.5"),
		EffectiveFrom: &effectiveFrom,
		EffectiveTo:   &effectiveTo,
	})
	require.NoError(t, err)

	require.NotNil(t, gotInput.EffectiveFrom)
	require.NotNil(t, gotInput.EffectiveTo)
	require.Equal(t, effectiveFrom, *gotInput.EffectiveFrom)
	require.Equal(t, effectiveTo, *gotInput.EffectiveTo)
	require.Equal(t, "01K00000000000000000000000", result.ID)
	require.NotNil(t, result.EffectiveTo)
	require.Equal(t, effectiveTo, *result.EffectiveTo)
}

func TestCreateCostBasis_DefaultEffectiveFromAllowsOpenEndedCostBasis(t *testing.T) {
	var gotInput currencies.CreateCostBasisInput
	svc := New(&fakeAdapter{
		createCostBasis: func(_ context.Context, input currencies.CreateCostBasisInput) (currencies.CostBasis, error) {
			gotInput = input

			return currencies.CostBasis{
				CurrencyID:    input.CurrencyID,
				FiatCode:      input.FiatCode,
				Rate:          input.Rate,
				EffectiveFrom: *input.EffectiveFrom,
				EffectiveTo:   input.EffectiveTo,
			}, nil
		},
	})

	_, err := svc.CreateCostBasis(t.Context(), currencies.CreateCostBasisInput{
		Namespace:  "test",
		CurrencyID: "01J00000000000000000000000",
		FiatCode:   "USD",
		Rate:       alpacadecimal.RequireFromString("0.5"),
	})
	require.NoError(t, err)

	require.NotNil(t, gotInput.EffectiveFrom)
	require.Nil(t, gotInput.EffectiveTo)
}

func TestCreateCostBasis_RejectsInvalidEffectiveTo(t *testing.T) {
	effectiveFrom := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	effectiveTo := effectiveFrom

	svc := New(&fakeAdapter{})

	_, err := svc.CreateCostBasis(t.Context(), currencies.CreateCostBasisInput{
		Namespace:     "test",
		CurrencyID:    "01J00000000000000000000000",
		FiatCode:      "USD",
		Rate:          alpacadecimal.RequireFromString("0.5"),
		EffectiveFrom: &effectiveFrom,
		EffectiveTo:   &effectiveTo,
	})
	require.Error(t, err)
	require.True(t, models.IsGenericValidationError(err), "error must be a validation error")
	require.Contains(t, err.Error(), "effective_to")
	require.Contains(t, err.Error(), "must be after effective_from")
}
