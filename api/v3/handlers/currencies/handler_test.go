package currencies

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCustomCurrencyMutationsAreDisabledByDefault(t *testing.T) {
	handler := New(func(context.Context) (string, error) {
		return "test", nil
	}, nil, false)

	t.Run("create currency", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/api/v3/openmeter/currencies/custom", nil)
		response := httptest.NewRecorder()

		handler.CreateCurrency().ServeHTTP(response, request)

		require.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("create cost basis", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/api/v3/openmeter/currencies/custom/currency-id/cost-bases", nil)
		response := httptest.NewRecorder()

		handler.CreateCostBasis().With("currency-id").ServeHTTP(response, request)

		require.Equal(t, http.StatusBadRequest, response.Code)
	})
}
