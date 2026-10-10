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

	h.GetCustomerEntitlementAccess().With(params).ServeHTTP(response, request)

	return response
}

func serveGetCustomerEntitlementValue(t *testing.T, svc fakeEntitlementService, params GetCustomerEntitlementValueParams) *httptest.ResponseRecorder {
	t.Helper()

	h := New(func(context.Context) (string, error) { return testNamespace, nil }, svc)
	request := httptest.NewRequest(http.MethodGet, "/api/v3/openmeter/customers/"+testCustomerID+"/entitlements/"+testEntitlementID+"/value", nil)
	response := httptest.NewRecorder()
	h.GetCustomerEntitlementValue().With(params).ServeHTTP(response, request)

	return response
}

func serveGetCustomerEntitlementValueByFeatureKey(t *testing.T, svc fakeEntitlementService, params GetCustomerEntitlementValueByFeatureKeyParams) *httptest.ResponseRecorder {
	t.Helper()

	h := New(func(context.Context) (string, error) { return testNamespace, nil }, svc)
	request := httptest.NewRequest(http.MethodGet, "/api/v3/openmeter/customers/"+testCustomerID+"/entitlement-access/features/"+testFeatureKey+"/value", nil)
	response := httptest.NewRecorder()
	h.GetCustomerEntitlementValueByFeatureKey().With(params).ServeHTTP(response, request)

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
				return entitlement.CustomerEntitlementAccess{FeatureKey: testFeatureKey, Type: entitlement.EntitlementTypeBoolean, Value: &booleanentitlement.BooleanEntitlementValue{}}, nil
			},
		}
	}

	t.Run("passes the namespaced customer and feature key and defaults at to now", func(t *testing.T) {
		// given a feature with a boolean entitlement for the customer
		// when checked without an explicit evaluation time
		// then the service receives the namespaced customer and current time
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

		var body api.BillingEntitlementAccessCheckResult
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.Equal(t, api.BillingEntitlementAccessCheckResult{
			Type:      lo.ToPtr(api.BillingEntitlementTypeBoolean),
			HasAccess: true,
		}, body)
	})

	t.Run("feature without an entitlement has no type", func(t *testing.T) {
		// given a feature with no entitlement
		// when the customer access is checked
		// then the response denies access without inventing an entitlement type
		response := serveGetCustomerEntitlementAccess(t, fakeEntitlementService{get: func(context.Context, entitlement.GetCustomerEntitlementAccessInput) (entitlement.CustomerEntitlementAccess, error) {
			return entitlement.CustomerEntitlementAccess{Value: &entitlement.NoAccessValue{}}, nil
		}}, GetCustomerEntitlementAccessParams{CustomerID: testCustomerID, FeatureKey: testFeatureKey})

		require.Equal(t, http.StatusOK, response.Code)
		require.JSONEq(t, `{"has_access":false}`, response.Body.String())
	})

	t.Run("passes the entitlement ID and the requested at", func(t *testing.T) {
		// given a customer entitlement and a historical evaluation time
		// when its value is requested by ID
		// then the service receives the entitlement ID and requested time
		var received entitlement.GetCustomerEntitlementAccessInput
		at := now.Add(-48 * time.Hour)

		response := serveGetCustomerEntitlementValue(t, capture(&received), GetCustomerEntitlementValueParams{
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
				Type:       entitlement.EntitlementTypeMetered,
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
		response := serveGetCustomerEntitlementValue(t, metered, GetCustomerEntitlementValueParams{
			CustomerID:    testCustomerID,
			EntitlementID: testEntitlementID,
		})

		require.Equal(t, http.StatusOK, response.Code)
		require.NotContains(t, response.Body.String(), `"value"`)
	})

	t.Run("includes the value with the expand", func(t *testing.T) {
		response := serveGetCustomerEntitlementValue(t, metered, GetCustomerEntitlementValueParams{
			CustomerID:    testCustomerID,
			EntitlementID: testEntitlementID,
			Expand:        []api.BillingEntitlementAccessExpand{api.BillingEntitlementAccessExpandValue},
		})

		require.Equal(t, http.StatusOK, response.Code)

		var body api.BillingEntitlementValueResult
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

	byID := GetCustomerEntitlementValueParams{CustomerID: testCustomerID, EntitlementID: testEntitlementID}

	t.Run("maps a missing customer or entitlement to 404", func(t *testing.T) {
		response := serveGetCustomerEntitlementValue(t, failing(models.NewGenericNotFoundError(errors.New("entitlement not found"))), byID)

		require.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("maps a deleted customer to 409", func(t *testing.T) {
		response := serveGetCustomerEntitlementValue(t, failing(models.NewGenericConflictError(errors.New("customer is deleted"))), byID)

		require.Equal(t, http.StatusConflict, response.Code)
	})

	t.Run("maps a validation error to 400", func(t *testing.T) {
		response := serveGetCustomerEntitlementValue(t, failing(models.NewGenericValidationError(errors.New("feature key is required"))), byID)

		require.Equal(t, http.StatusBadRequest, response.Code)
	})
}

func TestGetCustomerEntitlementValueByFeatureKeyHandler(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	metered := fakeEntitlementService{
		get: func(_ context.Context, input entitlement.GetCustomerEntitlementAccessInput) (entitlement.CustomerEntitlementAccess, error) {
			return entitlement.CustomerEntitlementAccess{
				FeatureKey: input.FeatureKey,
				Type:       entitlement.EntitlementTypeMetered,
				Value: &meteredentitlement.MeteredEntitlementValue{
					Balance:                   25,
					UsageInPeriod:             75,
					TotalAvailableGrantAmount: 100,
					GrantBalances:             map[string]float64{"grant-1": 25},
				},
			}, nil
		},
	}

	t.Run("defaults evaluation time and omits balance without expand", func(t *testing.T) {
		// given a metered entitlement for the requested feature
		// when its value is read without an evaluation time or expansion
		// then the service receives the feature key and current time, and the balance is omitted
		var received entitlement.GetCustomerEntitlementAccessInput
		svc := fakeEntitlementService{get: func(ctx context.Context, input entitlement.GetCustomerEntitlementAccessInput) (entitlement.CustomerEntitlementAccess, error) {
			received = input
			return metered.get(ctx, input)
		}}

		response := serveGetCustomerEntitlementValueByFeatureKey(t, svc, GetCustomerEntitlementValueByFeatureKeyParams{
			CustomerID: testCustomerID,
			FeatureKey: testFeatureKey,
		})

		require.Equal(t, http.StatusOK, response.Code)
		require.Equal(t, testNamespace, received.CustomerID.Namespace)
		require.Equal(t, testCustomerID, received.CustomerID.ID)
		require.Equal(t, testFeatureKey, received.FeatureKey)
		require.Empty(t, received.EntitlementID)
		require.Equal(t, now, received.At)

		var body api.BillingEntitlementFeatureValueResult
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.Equal(t, testFeatureKey, body.FeatureKey)
		require.Equal(t, lo.ToPtr(api.BillingEntitlementTypeMetered), body.Type)
		require.True(t, body.HasAccess)
		require.Nil(t, body.Value)
	})

	t.Run("evaluates at the requested time and expands the balance", func(t *testing.T) {
		// given a metered entitlement and a requested evaluation time
		// when the value expansion is requested
		// then the service receives that time and the response includes balance details
		at := now.Add(-time.Hour)
		var received entitlement.GetCustomerEntitlementAccessInput
		svc := fakeEntitlementService{get: func(ctx context.Context, input entitlement.GetCustomerEntitlementAccessInput) (entitlement.CustomerEntitlementAccess, error) {
			received = input
			return metered.get(ctx, input)
		}}

		response := serveGetCustomerEntitlementValueByFeatureKey(t, svc, GetCustomerEntitlementValueByFeatureKeyParams{
			CustomerID: testCustomerID,
			FeatureKey: testFeatureKey,
			At:         &at,
			Expand:     []api.BillingEntitlementAccessExpand{api.BillingEntitlementAccessExpandValue},
		})

		require.Equal(t, http.StatusOK, response.Code)
		require.Equal(t, at, received.At)

		var body api.BillingEntitlementFeatureValueResult
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.NotNil(t, body.Value)
		require.Equal(t, api.Numeric("25"), body.Value.Balance)
		require.Equal(t, api.Numeric("75"), body.Value.Usage)
		require.Equal(t, map[string]api.Numeric{"grant-1": "25"}, body.Value.GrantBalances)
	})

	t.Run("missing entitlement omits type and value", func(t *testing.T) {
		// given a feature without an active entitlement
		// when its value is requested with expansion
		// then access is denied without inventing a type or balance
		response := serveGetCustomerEntitlementValueByFeatureKey(t, fakeEntitlementService{get: func(_ context.Context, input entitlement.GetCustomerEntitlementAccessInput) (entitlement.CustomerEntitlementAccess, error) {
			return entitlement.CustomerEntitlementAccess{FeatureKey: input.FeatureKey, Value: &entitlement.NoAccessValue{}}, nil
		}}, GetCustomerEntitlementValueByFeatureKeyParams{
			CustomerID: testCustomerID,
			FeatureKey: testFeatureKey,
			Expand:     []api.BillingEntitlementAccessExpand{api.BillingEntitlementAccessExpandValue},
		})

		require.Equal(t, http.StatusOK, response.Code)
		require.JSONEq(t, `{"feature_key":"tokens","has_access":false}`, response.Body.String())
	})

	t.Run("known entitlement with no access retains its type", func(t *testing.T) {
		response := serveGetCustomerEntitlementValueByFeatureKey(t, fakeEntitlementService{get: func(_ context.Context, input entitlement.GetCustomerEntitlementAccessInput) (entitlement.CustomerEntitlementAccess, error) {
			return entitlement.CustomerEntitlementAccess{FeatureKey: input.FeatureKey, Type: entitlement.EntitlementTypeMetered, Value: &entitlement.NoAccessValue{}}, nil
		}}, GetCustomerEntitlementValueByFeatureKeyParams{CustomerID: testCustomerID, FeatureKey: testFeatureKey})

		require.Equal(t, http.StatusOK, response.Code)
		require.JSONEq(t, `{"feature_key":"tokens","type":"metered","has_access":false}`, response.Body.String())
	})

	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{
		{name: "missing customer", err: models.NewGenericNotFoundError(errors.New("customer not found")), status: http.StatusNotFound},
		{name: "deleted customer", err: models.NewGenericConflictError(errors.New("customer is deleted")), status: http.StatusConflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := serveGetCustomerEntitlementValueByFeatureKey(t, fakeEntitlementService{get: func(context.Context, entitlement.GetCustomerEntitlementAccessInput) (entitlement.CustomerEntitlementAccess, error) {
				return entitlement.CustomerEntitlementAccess{}, tc.err
			}}, GetCustomerEntitlementValueByFeatureKeyParams{CustomerID: testCustomerID, FeatureKey: testFeatureKey})

			require.Equal(t, tc.status, response.Code)
		})
	}
}
