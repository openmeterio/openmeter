package customersentitlement

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	testNamespace     = "test-ns"
	testCustomerID    = "01K5A4V2X8Q9Z7M3N6P1R4S8T2"
	testEntitlementID = "01K5A4V2X8Q9Z7M3N6P1R4S8T3"
	testFeatureKey    = "tokens"
)

type fakeEntitlementService struct {
	entitlement.Service
	get func(ctx context.Context, input entitlement.GetCustomerEntitlementAccessInput) (entitlement.CustomerEntitlementAccess, error)
}

func (f fakeEntitlementService) GetCustomerEntitlementAccess(ctx context.Context, input entitlement.GetCustomerEntitlementAccessInput) (entitlement.CustomerEntitlementAccess, error) {
	return f.get(ctx, input)
}

func serveGetCustomerEntitlementAccess(t *testing.T, svc fakeEntitlementService, params GetCustomerEntitlementAccessParams) *httptest.ResponseRecorder {
	t.Helper()

	h := New(func(context.Context) (string, error) { return testNamespace, nil }, svc)

	request := httptest.NewRequest(http.MethodGet, "/api/v3/openmeter/customers/"+testCustomerID+"/entitlement-access/features/"+testFeatureKey, nil)
	response := httptest.NewRecorder()

	h.GetCustomerEntitlementAccess(OperationGetCustomerEntitlementAccess).With(params).ServeHTTP(response, request)

	return response
}

func TestGetCustomerEntitlementAccessHandler(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	capture := func(received *entitlement.GetCustomerEntitlementAccessInput) fakeEntitlementService {
		return fakeEntitlementService{
			get: func(_ context.Context, input entitlement.GetCustomerEntitlementAccessInput) (entitlement.CustomerEntitlementAccess, error) {
				*received = input
				return entitlement.CustomerEntitlementAccess{FeatureKey: testFeatureKey, Value: &booleanentitlement.BooleanEntitlementValue{}}, nil
			},
		}
	}

	t.Run("passes the namespaced customer and feature key and defaults at to now", func(t *testing.T) {
		var received entitlement.GetCustomerEntitlementAccessInput

		response := serveGetCustomerEntitlementAccess(t, capture(&received), GetCustomerEntitlementAccessParams{
			CustomerID: testCustomerID,
			FeatureKey: testFeatureKey,
		})

		require.Equal(t, http.StatusOK, response.Code)
		require.Equal(t, entitlement.GetCustomerEntitlementAccessInput{
			CustomerID: received.CustomerID,
			FeatureKey: testFeatureKey,
			At:         now,
		}, received)
		require.Equal(t, testNamespace, received.CustomerID.Namespace)
		require.Equal(t, testCustomerID, received.CustomerID.ID)

		var body api.BillingEntitlementAccessResult
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.Equal(t, api.BillingEntitlementAccessResult{
			FeatureKey: testFeatureKey,
			Type:       api.BillingEntitlementTypeBoolean,
			HasAccess:  true,
		}, body)
	})

	t.Run("passes the entitlement ID and the requested at", func(t *testing.T) {
		var received entitlement.GetCustomerEntitlementAccessInput
		at := now.Add(-48 * time.Hour)

		response := serveGetCustomerEntitlementAccess(t, capture(&received), GetCustomerEntitlementAccessParams{
			CustomerID:    testCustomerID,
			EntitlementID: testEntitlementID,
			At:            &at,
		})

		require.Equal(t, http.StatusOK, response.Code)
		require.Empty(t, received.FeatureKey)
		require.Equal(t, testEntitlementID, received.EntitlementID)
		require.Equal(t, at, received.At)
	})

	metered := fakeEntitlementService{
		get: func(_ context.Context, input entitlement.GetCustomerEntitlementAccessInput) (entitlement.CustomerEntitlementAccess, error) {
			return entitlement.CustomerEntitlementAccess{
				FeatureKey: input.FeatureKey,
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
		response := serveGetCustomerEntitlementAccess(t, metered, GetCustomerEntitlementAccessParams{
			CustomerID: testCustomerID,
			FeatureKey: testFeatureKey,
		})

		require.Equal(t, http.StatusOK, response.Code)
		require.NotContains(t, response.Body.String(), `"value"`)
	})

	t.Run("includes the value with the expand", func(t *testing.T) {
		response := serveGetCustomerEntitlementAccess(t, metered, GetCustomerEntitlementAccessParams{
			CustomerID: testCustomerID,
			FeatureKey: testFeatureKey,
			Expand:     []api.BillingEntitlementAccessExpand{api.BillingEntitlementAccessExpandValue},
		})

		require.Equal(t, http.StatusOK, response.Code)

		var body api.BillingEntitlementAccessResult
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.NotNil(t, body.Value)
		require.Equal(t, api.Numeric("25"), body.Value.Balance)
		require.Equal(t, api.Numeric("75"), body.Value.Usage)
		require.Equal(t, api.Numeric("100"), body.Value.TotalAvailableGrantAmount)
		require.Equal(t, map[string]api.Numeric{"grant-1": "25"}, body.Value.GrantBalances)
	})

	failing := func(err error) fakeEntitlementService {
		return fakeEntitlementService{
			get: func(context.Context, entitlement.GetCustomerEntitlementAccessInput) (entitlement.CustomerEntitlementAccess, error) {
				return entitlement.CustomerEntitlementAccess{}, err
			},
		}
	}

	byID := GetCustomerEntitlementAccessParams{CustomerID: testCustomerID, EntitlementID: testEntitlementID}

	t.Run("maps a missing customer or entitlement to 404", func(t *testing.T) {
		response := serveGetCustomerEntitlementAccess(t, failing(models.NewGenericNotFoundError(errors.New("entitlement not found"))), byID)

		require.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("maps a deleted customer to 412", func(t *testing.T) {
		response := serveGetCustomerEntitlementAccess(t, failing(models.NewGenericPreConditionFailedError(errors.New("customer is deleted"))), byID)

		require.Equal(t, http.StatusPreconditionFailed, response.Code)
	})

	t.Run("maps a validation error to 400", func(t *testing.T) {
		response := serveGetCustomerEntitlementAccess(t, failing(models.NewGenericValidationError(errors.New("feature key is required"))), byID)

		require.Equal(t, http.StatusBadRequest, response.Code)
	})
}
