package currencies

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/currencies"
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
		apiType      v3.BillingCurrencyType
		currencyType currencies.CurrencyType
	}{
		{
			name:         "custom",
			apiType:      v3.BillingCurrencyTypeCustom,
			currencyType: currencies.CurrencyTypeCustom,
		},
		{
			name:         "fiat",
			apiType:      v3.BillingCurrencyTypeFiat,
			currencyType: currencies.CurrencyTypeFiat,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// given:
			// - a list request filtered by one API currency type
			service := &listCurrenciesService{}
			handler := New(func(context.Context) (string, error) {
				return "test", nil
			}, service)

			request := httptest.NewRequest(http.MethodGet, "/api/v3/currencies", nil)
			response := httptest.NewRecorder()

			// when:
			// - the v3 list handler decodes the generated API parameters
			handler.ListCurrencies().With(v3.ListCurrenciesParams{
				Filter: &v3.ListCurrenciesParamsFilter{
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
