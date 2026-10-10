package customersentitlements

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
)

func serveCreateCustomerEntitlementGrant(t *testing.T, svc fakeService, body string) *httptest.ResponseRecorder {
	t.Helper()

	h := New(func(context.Context) (string, error) { return testNamespace, nil }, svc)

	request := httptest.NewRequest(http.MethodPost, "/api/v3/openmeter/customers/"+testCustomerID+"/entitlements/"+testEntitlementID+"/grants", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	h.CreateCustomerEntitlementGrant().With(CreateCustomerEntitlementGrantParams{
		CustomerID:    testCustomerID,
		EntitlementID: testEntitlementID,
	}).ServeHTTP(response, request)

	return response
}

func TestCreateCustomerEntitlementGrantHandler(t *testing.T) {
	effectiveAt := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

	t.Run("passes the grant to the service and maps the created grant", func(t *testing.T) {
		var received entitlement.CreateCustomerEntitlementGrantInput

		response := serveCreateCustomerEntitlementGrant(t, fakeService{
			createGrant: func(_ context.Context, input entitlement.CreateCustomerEntitlementGrantInput) (grant.Grant, error) {
				received = input

				return grant.Grant{
					ID:               "01K4WAQ0J99ZZ0MD75HXR112HA",
					OwnerID:          input.EntitlementID,
					Amount:           input.Grant.Amount,
					Priority:         input.Grant.Priority,
					EffectiveAt:      input.Grant.EffectiveAt,
					Expiration:       input.Grant.Expiration,
					ResetMaxRollover: input.Grant.ResetMaxRollover,
					ResetMinRollover: input.Grant.ResetMinRollover,
				}, nil
			},
		}, `{"amount": "100.5", "priority": 2, "effective_at": "2025-01-01T00:00:00Z", "expires_after": "P1W"}`)

		require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
		require.Equal(t, testNamespace, received.CustomerID.Namespace)
		require.Equal(t, testCustomerID, received.CustomerID.ID)
		require.Equal(t, testEntitlementID, received.EntitlementID)
		require.Equal(t, 100.5, received.Grant.Amount)
		require.Equal(t, uint8(2), received.Grant.Priority)
		require.Equal(t, effectiveAt, received.Grant.EffectiveAt)
		require.Equal(t, &grant.ExpirationPeriod{Count: 1, Duration: grant.ExpirationPeriodDurationWeek}, received.Grant.Expiration)
		require.Equal(t, 100.5, received.Grant.ResetMaxRollover)

		var body api.BillingEntitlementGrant
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.Equal(t, "01K4WAQ0J99ZZ0MD75HXR112HA", body.Id)
		require.Equal(t, testEntitlementID, body.EntitlementId)
		require.Equal(t, "100.5", body.Amount)
		require.Equal(t, "P1W", lo.FromPtr(body.ExpiresAfter))
	})

	t.Run("rejects an expiration that is not a single unit", func(t *testing.T) {
		response := serveCreateCustomerEntitlementGrant(t, fakeService{
			createGrant: func(context.Context, entitlement.CreateCustomerEntitlementGrantInput) (grant.Grant, error) {
				t.Fatal("service must not be called")
				return grant.Grant{}, nil
			},
		}, `{"amount": "100", "effective_at": "2025-01-01T00:00:00Z", "expires_after": "P1DT12H"}`)

		require.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("maps a non-metered entitlement to 400", func(t *testing.T) {
		response := serveCreateCustomerEntitlementGrant(t, fakeService{
			createGrant: func(context.Context, entitlement.CreateCustomerEntitlementGrantInput) (grant.Grant, error) {
				return grant.Grant{}, &entitlement.WrongTypeError{Expected: entitlement.EntitlementTypeMetered, Actual: entitlement.EntitlementTypeBoolean}
			},
		}, `{"amount": "100", "effective_at": "2025-01-01T00:00:00Z"}`)

		require.Equal(t, http.StatusBadRequest, response.Code)
	})
}
