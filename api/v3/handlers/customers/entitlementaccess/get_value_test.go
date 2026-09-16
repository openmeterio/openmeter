package customersentitlement

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

const testEntitlementID = "01K5A4V2X8Q9Z7M3N6P1R4S8T3"

func serveGetCustomerEntitlementValue(t *testing.T, svc fakeEntitlementService, params api.GetCustomerEntitlementValueParams) *httptest.ResponseRecorder {
	t.Helper()

	h := New(func(context.Context) (string, error) { return testNamespace, nil }, svc)

	request := httptest.NewRequest(http.MethodGet, "/api/v3/openmeter/customers/"+testCustomerID+"/entitlements/"+testEntitlementID+"/value", nil)
	response := httptest.NewRecorder()

	h.GetCustomerEntitlementValue().With(GetCustomerEntitlementValueParams{
		CustomerID:    testCustomerID,
		EntitlementID: testEntitlementID,
		Params:        params,
	}).ServeHTTP(response, request)

	return response
}

func TestGetCustomerEntitlementValueHandler(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	capture := func(received *entitlement.GetCustomerEntitlementValueInput) fakeEntitlementService {
		return fakeEntitlementService{
			getValue: func(_ context.Context, input entitlement.GetCustomerEntitlementValueInput) (entitlement.CustomerEntitlementAccess, error) {
				*received = input
				return entitlement.CustomerEntitlementAccess{FeatureKey: testFeatureKey, Value: &booleanentitlement.BooleanEntitlementValue{}}, nil
			},
		}
	}

	t.Run("passes the namespaced customer and entitlement ID and defaults at to now", func(t *testing.T) {
		var received entitlement.GetCustomerEntitlementValueInput

		response := serveGetCustomerEntitlementValue(t, capture(&received), api.GetCustomerEntitlementValueParams{})

		require.Equal(t, http.StatusOK, response.Code)
		require.Equal(t, testNamespace, received.CustomerID.Namespace)
		require.Equal(t, testCustomerID, received.CustomerID.ID)
		require.Equal(t, testEntitlementID, received.EntitlementID)
		require.Equal(t, now, received.At)

		var body api.BillingEntitlementAccessResult
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.Equal(t, api.BillingEntitlementAccessResult{
			FeatureKey: testFeatureKey,
			Type:       api.BillingEntitlementTypeBoolean,
			HasAccess:  true,
		}, body)
	})

	t.Run("passes the requested at", func(t *testing.T) {
		var received entitlement.GetCustomerEntitlementValueInput
		at := now.Add(-48 * time.Hour)

		response := serveGetCustomerEntitlementValue(t, capture(&received), api.GetCustomerEntitlementValueParams{At: &at})

		require.Equal(t, http.StatusOK, response.Code)
		require.Equal(t, at, received.At)
	})

	metered := fakeEntitlementService{
		getValue: func(context.Context, entitlement.GetCustomerEntitlementValueInput) (entitlement.CustomerEntitlementAccess, error) {
			return entitlement.CustomerEntitlementAccess{
				FeatureKey: testFeatureKey,
				Value: &meteredentitlement.MeteredEntitlementValue{
					Balance:                   25,
					UsageInPeriod:             75,
					TotalAvailableGrantAmount: 100,
					GrantBalances:             map[string]float64{"grant-1": 25},
				},
			}, nil
		},
	}

	t.Run("omits the value without the expand", func(t *testing.T) {
		response := serveGetCustomerEntitlementValue(t, metered, api.GetCustomerEntitlementValueParams{})

		require.Equal(t, http.StatusOK, response.Code)
		require.NotContains(t, response.Body.String(), `"value"`)
	})

	t.Run("includes the value with the expand", func(t *testing.T) {
		response := serveGetCustomerEntitlementValue(t, metered, api.GetCustomerEntitlementValueParams{
			Expand: lo.ToPtr([]api.BillingEntitlementAccessExpand{api.BillingEntitlementAccessExpandValue}),
		})

		require.Equal(t, http.StatusOK, response.Code)

		var body api.BillingEntitlementAccessResult
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.NotNil(t, body.Value)
		require.Equal(t, api.Numeric("25"), body.Value.Balance)
		require.Equal(t, api.Numeric("75"), body.Value.Usage)
		require.Equal(t, map[string]api.Numeric{"grant-1": "25"}, body.Value.GrantBalances)
	})

	failing := func(err error) fakeEntitlementService {
		return fakeEntitlementService{
			getValue: func(context.Context, entitlement.GetCustomerEntitlementValueInput) (entitlement.CustomerEntitlementAccess, error) {
				return entitlement.CustomerEntitlementAccess{}, err
			},
		}
	}

	t.Run("maps a missing entitlement to 404", func(t *testing.T) {
		response := serveGetCustomerEntitlementValue(t, failing(models.NewGenericNotFoundError(errors.New("entitlement not found"))), api.GetCustomerEntitlementValueParams{})

		require.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("maps a deleted customer to 412", func(t *testing.T) {
		response := serveGetCustomerEntitlementValue(t, failing(models.NewGenericPreConditionFailedError(errors.New("customer is deleted"))), api.GetCustomerEntitlementValueParams{})

		require.Equal(t, http.StatusPreconditionFailed, response.Code)
	})

	t.Run("maps a validation error to 400", func(t *testing.T) {
		response := serveGetCustomerEntitlementValue(t, failing(models.NewGenericValidationError(errors.New("entitlement ID is required"))), api.GetCustomerEntitlementValueParams{})

		require.Equal(t, http.StatusBadRequest, response.Code)
	})
}
