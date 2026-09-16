package customersentitlement

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	testNamespace  = "test-ns"
	testCustomerID = "01K5A4V2X8Q9Z7M3N6P1R4S8T2"
	testFeatureKey = "tokens"
)

type fakeEntitlementService struct {
	entitlement.Service
	get      func(ctx context.Context, input entitlement.GetCustomerEntitlementAccessInput) (entitlement.CustomerEntitlementAccess, error)
	getValue func(ctx context.Context, input entitlement.GetCustomerEntitlementValueInput) (entitlement.CustomerEntitlementAccess, error)
}

func (f fakeEntitlementService) GetCustomerEntitlementAccess(ctx context.Context, input entitlement.GetCustomerEntitlementAccessInput) (entitlement.CustomerEntitlementAccess, error) {
	return f.get(ctx, input)
}

func (f fakeEntitlementService) GetCustomerEntitlementValue(ctx context.Context, input entitlement.GetCustomerEntitlementValueInput) (entitlement.CustomerEntitlementAccess, error) {
	return f.getValue(ctx, input)
}

func serveGetCustomerEntitlementAccess(t *testing.T, svc fakeEntitlementService, expand ...api.BillingEntitlementAccessExpand) *httptest.ResponseRecorder {
	t.Helper()

	h := New(func(context.Context) (string, error) { return testNamespace, nil }, svc)

	params := GetCustomerEntitlementAccessParams{
		CustomerID: testCustomerID,
		FeatureKey: testFeatureKey,
	}
	if len(expand) > 0 {
		params.Params.Expand = &expand
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v3/openmeter/customers/"+testCustomerID+"/entitlement-access/features/"+testFeatureKey, nil)
	response := httptest.NewRecorder()

	h.GetCustomerEntitlementAccess().With(params).ServeHTTP(response, request)

	return response
}

func TestGetCustomerEntitlementAccessHandler(t *testing.T) {
	t.Run("passes the namespaced customer and feature key to the service", func(t *testing.T) {
		var received entitlement.GetCustomerEntitlementAccessInput

		response := serveGetCustomerEntitlementAccess(t, fakeEntitlementService{
			get: func(_ context.Context, input entitlement.GetCustomerEntitlementAccessInput) (entitlement.CustomerEntitlementAccess, error) {
				received = input
				return entitlement.CustomerEntitlementAccess{FeatureKey: input.FeatureKey, Value: &booleanentitlement.BooleanEntitlementValue{}}, nil
			},
		})

		require.Equal(t, http.StatusOK, response.Code)
		require.Equal(t, testNamespace, received.CustomerID.Namespace)
		require.Equal(t, testCustomerID, received.CustomerID.ID)
		require.Equal(t, testFeatureKey, received.FeatureKey)

		var body api.BillingEntitlementAccessResult
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.Equal(t, api.BillingEntitlementAccessResult{
			FeatureKey: testFeatureKey,
			Type:       api.BillingEntitlementTypeBoolean,
			HasAccess:  true,
		}, body)
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
		response := serveGetCustomerEntitlementAccess(t, metered)

		require.Equal(t, http.StatusOK, response.Code)
		require.NotContains(t, response.Body.String(), `"value"`)
	})

	t.Run("includes the value with the expand", func(t *testing.T) {
		response := serveGetCustomerEntitlementAccess(t, metered, api.BillingEntitlementAccessExpandValue)

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

	t.Run("maps a missing customer to 404", func(t *testing.T) {
		response := serveGetCustomerEntitlementAccess(t, failing(models.NewGenericNotFoundError(errors.New("customer not found"))))

		require.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("maps a deleted customer to 412", func(t *testing.T) {
		response := serveGetCustomerEntitlementAccess(t, failing(models.NewGenericPreConditionFailedError(errors.New("customer is deleted"))))

		require.Equal(t, http.StatusPreconditionFailed, response.Code)
	})

	t.Run("maps a validation error to 400", func(t *testing.T) {
		response := serveGetCustomerEntitlementAccess(t, failing(models.NewGenericValidationError(errors.New("feature key is required"))))

		require.Equal(t, http.StatusBadRequest, response.Code)
	})
}
