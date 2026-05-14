package service

import (
	"context"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
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
	custom []currencies.Currency
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
	panic("not implemented")
}

func (f *fakeAdapter) CreateCostBasis(_ context.Context, _ currencies.CreateCostBasisInput) (currencies.CostBasis, error) {
	panic("not implemented")
}

func (f *fakeAdapter) ListCostBases(_ context.Context, _ currencies.ListCostBasesInput) (pagination.Result[currencies.CostBasis], error) {
	panic("not implemented")
}

// newTestService creates a Service backed by a fake adapter seeded with custom currencies.
func newTestService(custom []currencies.Currency) *Service {
	return New(&fakeAdapter{custom: custom})
}

func TestListCurrencies_CombinedPath(t *testing.T) {
	customCurrency := currencies.Currency{Code: "MYCUSTOM", Name: "My Custom Currency", Symbol: "MC"}

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
	customCurrency := currencies.Currency{Code: "MYCUSTOM", Name: "My Custom Currency", Symbol: "MC"}
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
