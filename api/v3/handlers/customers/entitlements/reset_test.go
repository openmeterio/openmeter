package customersentitlements

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/models"
)

func serveResetCustomerEntitlementUsage(t *testing.T, svc fakeService, body string) *httptest.ResponseRecorder {
	t.Helper()

	h := New(func(context.Context) (string, error) { return testNamespace, nil }, svc)

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v3/openmeter/customers/"+testCustomerID+"/entitlements/"+testEntitlementID+"/reset", reader)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	h.ResetCustomerEntitlementUsage().With(ResetCustomerEntitlementUsageParams{
		CustomerID:    testCustomerID,
		EntitlementID: testEntitlementID,
	}).ServeHTTP(response, request)

	return response
}

func TestResetCustomerEntitlementUsageHandler(t *testing.T) {
	t.Run("passes the body to the service", func(t *testing.T) {
		var received entitlement.ResetCustomerEntitlementUsageInput

		response := serveResetCustomerEntitlementUsage(t, fakeService{
			reset: func(_ context.Context, input entitlement.ResetCustomerEntitlementUsageInput) error {
				received = input
				return nil
			},
		}, `{"effective_at": "2025-01-01T00:00:00Z", "retain_anchor": true, "preserve_overage": false}`)

		require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
		require.Equal(t, testNamespace, received.CustomerID.Namespace)
		require.Equal(t, testCustomerID, received.CustomerID.ID)
		require.Equal(t, testEntitlementID, received.EntitlementID)
		require.NotNil(t, received.EffectiveAt)
		require.True(t, time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC).Equal(*received.EffectiveAt))
		require.True(t, received.RetainAnchor)
		require.Equal(t, lo.ToPtr(false), received.PreserveOverage)
	})

	t.Run("defaults an empty body", func(t *testing.T) {
		var received entitlement.ResetCustomerEntitlementUsageInput

		response := serveResetCustomerEntitlementUsage(t, fakeService{
			reset: func(_ context.Context, input entitlement.ResetCustomerEntitlementUsageInput) error {
				received = input
				return nil
			},
		}, "")

		require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
		require.Nil(t, received.EffectiveAt)
		require.False(t, received.RetainAnchor)
		require.Nil(t, received.PreserveOverage)
	})

	t.Run("rejects a malformed body", func(t *testing.T) {
		response := serveResetCustomerEntitlementUsage(t, fakeService{
			reset: func(context.Context, entitlement.ResetCustomerEntitlementUsageInput) error {
				t.Fatal("service must not be called")
				return nil
			},
		}, `{"effective_at": "yesterday"}`)

		require.Equal(t, http.StatusBadRequest, response.Code)
	})

	failing := func(err error) fakeService {
		return fakeService{
			reset: func(context.Context, entitlement.ResetCustomerEntitlementUsageInput) error {
				return err
			},
		}
	}

	t.Run("maps a missing customer to 404", func(t *testing.T) {
		response := serveResetCustomerEntitlementUsage(t, failing(models.NewGenericNotFoundError(errors.New("customer not found"))), "{}")

		require.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("maps a missing entitlement to 404", func(t *testing.T) {
		response := serveResetCustomerEntitlementUsage(t, failing(&entitlement.NotFoundError{EntitlementID: models.NamespacedID{Namespace: testNamespace, ID: testEntitlementID}}), "{}")

		require.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("maps a deleted customer to 409", func(t *testing.T) {
		response := serveResetCustomerEntitlementUsage(t, failing(models.NewGenericConflictError(errors.New("customer is deleted"))), "{}")

		require.Equal(t, http.StatusConflict, response.Code)
	})

	t.Run("maps a non-metered entitlement to 400", func(t *testing.T) {
		response := serveResetCustomerEntitlementUsage(t, failing(&entitlement.WrongTypeError{Expected: entitlement.EntitlementTypeMetered, Actual: entitlement.EntitlementTypeBoolean}), "{}")

		require.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("maps a validation error to 400", func(t *testing.T) {
		response := serveResetCustomerEntitlementUsage(t, failing(models.NewGenericValidationError(errors.New("reset is before the last reset"))), "{}")

		require.Equal(t, http.StatusBadRequest, response.Code)
	})
}
