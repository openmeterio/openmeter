package currencies

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type listCurrenciesService struct {
	currencies.Service

	input currencies.ListCurrenciesInput
}

func (s *listCurrenciesService) ListCurrencies(_ context.Context, input currencies.ListCurrenciesInput) (pagination.Result[currencies.Currency], error) {
	s.input = input

	return pagination.Result[currencies.Currency]{
		Page: input.Page,
	}, nil
}

func TestListCurrenciesFilterByType(t *testing.T) {
	testCases := []struct {
		name         string
		apiType      apiv3models.BillingCurrencyType
		currencyType currencyx.CurrencyType
	}{
		{
			name:         "custom",
			apiType:      apiv3models.BillingCurrencyTypeCustom,
			currencyType: currencyx.CurrencyTypeCustom,
		},
		{
			name:         "fiat",
			apiType:      apiv3models.BillingCurrencyTypeFiat,
			currencyType: currencyx.CurrencyTypeFiat,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// given:
			// - a list request filtered by one API currency type
			service := &listCurrenciesService{}
			handler := New(func(context.Context) (string, error) {
				return "test", nil
			}, service, true)

			request := httptest.NewRequest(http.MethodGet, "/api/v3/currencies", nil)
			response := httptest.NewRecorder()

			// when:
			// - the v3 list handler decodes the generated API parameters
			handler.ListCurrencies().With(apiv3models.ListCurrenciesParams{
				Filter: &apiv3models.ListCurrenciesParamsFilter{
					Type: &testCase.apiType,
				},
			}).ServeHTTP(response, request)

			// then:
			// - the service receives the corresponding domain currency type filter
			require.Equal(t, http.StatusOK, response.Code)
			require.NotNil(t, service.input.CurrencyType)
			assert.Equal(t, testCase.currencyType, *service.input.CurrencyType)
		})
	}
}

func TestListCurrenciesExpandCostBasis(t *testing.T) {
	testCases := []struct {
		name              string
		expand            *[]apiv3models.BillingCurrencyExpand
		expectedCostBasis bool
	}{
		{
			name:              "cost_basis requested",
			expand:            &[]apiv3models.BillingCurrencyExpand{apiv3models.BillingCurrencyExpandCostBasis},
			expectedCostBasis: true,
		},
		{
			name:              "expand omitted",
			expand:            nil,
			expectedCostBasis: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// given:
			// - a list request with an optional cost_basis expand
			service := &listCurrenciesService{}
			handler := New(func(context.Context) (string, error) {
				return "test", nil
			}, service, false)

			request := httptest.NewRequest(http.MethodGet, "/api/v3/currencies", nil)
			response := httptest.NewRecorder()

			// when:
			// - the v3 list handler decodes the generated API parameters
			handler.ListCurrencies().With(apiv3models.ListCurrenciesParams{
				Expand: testCase.expand,
			}).ServeHTTP(response, request)

			// then:
			// - the decoded expand parameter toggles the domain cost basis flag
			require.Equal(t, http.StatusOK, response.Code)
			assert.Equal(t, testCase.expectedCostBasis, service.input.CostBasis)
		})
	}
}
