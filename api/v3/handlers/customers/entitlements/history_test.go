package customersentitlements

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
	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

const (
	testNamespace     = "test-ns"
	testCustomerID    = "01K5A4V2X8Q9Z7M3N6P1R4S8T2"
	testEntitlementID = "01K5A4V2X8Q9Z7M3N6P1R4S8T3"
)

type fakeService struct {
	history     func(ctx context.Context, input entitlement.GetCustomerEntitlementHistoryInput) (entitlement.CustomerEntitlementHistory, error)
	createGrant func(ctx context.Context, input entitlement.CreateCustomerEntitlementGrantInput) (grant.Grant, error)
}

func (f fakeService) GetCustomerEntitlementHistory(ctx context.Context, input entitlement.GetCustomerEntitlementHistoryInput) (entitlement.CustomerEntitlementHistory, error) {
	return f.history(ctx, input)
}

func (f fakeService) CreateCustomerEntitlement(context.Context, entitlement.CreateCustomerEntitlementInput) (*entitlement.Entitlement, error) {
	return nil, errors.New("not implemented")
}

func (f fakeService) GetCustomerEntitlement(context.Context, entitlement.GetCustomerEntitlementInput) (*entitlement.Entitlement, error) {
	return nil, errors.New("not implemented")
}

func (f fakeService) ListCustomerEntitlements(context.Context, entitlement.ListCustomerEntitlementsInput) (pagination.Result[entitlement.Entitlement], error) {
	return pagination.Result[entitlement.Entitlement]{}, errors.New("not implemented")
}

func (f fakeService) ListCustomerEntitlementGrants(context.Context, entitlement.ListCustomerEntitlementGrantsInput) (pagination.Result[grant.Grant], error) {
	return pagination.Result[grant.Grant]{}, errors.New("not implemented")
}

func (f fakeService) CreateCustomerEntitlementGrant(ctx context.Context, input entitlement.CreateCustomerEntitlementGrantInput) (grant.Grant, error) {
	return f.createGrant(ctx, input)
}

func serveGetCustomerEntitlementHistory(t *testing.T, svc fakeService, params api.GetCustomerEntitlementHistoryParams) *httptest.ResponseRecorder {
	t.Helper()

	h := New(func(context.Context) (string, error) { return testNamespace, nil }, svc)

	request := httptest.NewRequest(http.MethodGet, "/api/v3/openmeter/customers/"+testCustomerID+"/entitlements/"+testEntitlementID+"/history", nil)
	response := httptest.NewRecorder()

	h.GetCustomerEntitlementHistory().With(GetCustomerEntitlementHistoryParams{
		CustomerID:    testCustomerID,
		EntitlementID: testEntitlementID,
		Params:        params,
	}).ServeHTTP(response, request)

	return response
}

func TestGetCustomerEntitlementHistoryHandler(t *testing.T) {
	from := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(2 * time.Hour)

	t.Run("passes the query to the service and maps the history", func(t *testing.T) {
		var received entitlement.GetCustomerEntitlementHistoryInput

		burndown, err := engine.NewGrantBurnDownHistory([]engine.GrantBurnDownHistorySegment{{
			ClosedPeriod:   timeutil.ClosedPeriod{From: from, To: to},
			BalanceAtStart: balance.Map{"grant-1": 10},
			TotalUsage:     3,
			Overage:        0,
			GrantUsages:    engine.GrantUsages{{GrantID: "grant-1", Usage: 3}},
		}}, balance.Snapshot{At: from, UsageSnapshot: &balance.UsageSnapshot{}})
		require.NoError(t, err)

		response := serveGetCustomerEntitlementHistory(t, fakeService{
			history: func(_ context.Context, input entitlement.GetCustomerEntitlementHistoryInput) (entitlement.CustomerEntitlementHistory, error) {
				received = input
				return entitlement.CustomerEntitlementHistory{
					Windows: []entitlement.BalanceHistoryWindow{
						{From: from, To: from.Add(time.Hour), UsageInPeriod: 1, BalanceAtStart: 10},
						{From: from.Add(time.Hour), To: to, UsageInPeriod: 2, BalanceAtStart: 9},
					},
					Burndown: burndown,
				}, nil
			},
		}, api.GetCustomerEntitlementHistoryParams{
			From:       &from,
			To:         &to,
			WindowSize: api.BillingEntitlementHistoryWindowSizeP1D,
			TimeZone:   lo.ToPtr("Europe/Budapest"),
		})

		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		require.Equal(t, testNamespace, received.CustomerID.Namespace)
		require.Equal(t, testCustomerID, received.CustomerID.ID)
		require.Equal(t, testEntitlementID, received.EntitlementID)
		require.Equal(t, &from, received.From)
		require.Equal(t, &to, received.To)
		require.Equal(t, meter.WindowSizeDay, received.WindowSize)
		require.Equal(t, "Europe/Budapest", received.TimeZone.String())

		var body api.BillingEntitlementHistory
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.Equal(t, api.BillingEntitlementHistory{
			WindowedHistory: []api.BillingEntitlementHistoryWindow{
				{Period: api.ClosedPeriod{From: from, To: from.Add(time.Hour)}, Usage: "1", BalanceAtStart: "10"},
				{Period: api.ClosedPeriod{From: from.Add(time.Hour), To: to}, Usage: "2", BalanceAtStart: "9"},
			},
			BurndownHistory: []api.BillingEntitlementBurndownSegment{{
				Period:  api.ClosedPeriod{From: from, To: to},
				Usage:   "3",
				Overage: "0",
				Balance: api.BillingEntitlementBurndownBalance{Start: "10", End: "7"},
				GrantBalances: api.BillingEntitlementBurndownGrantBalances{
					Start: map[string]api.Numeric{"grant-1": "10"},
					End:   map[string]api.Numeric{"grant-1": "7"},
				},
				GrantUsages: []api.BillingEntitlementGrantUsage{{GrantId: "grant-1", Usage: "3"}},
			}},
		}, body)
	})

	t.Run("defaults the time zone to UTC and returns empty lists", func(t *testing.T) {
		var received entitlement.GetCustomerEntitlementHistoryInput

		response := serveGetCustomerEntitlementHistory(t, fakeService{
			history: func(_ context.Context, input entitlement.GetCustomerEntitlementHistoryInput) (entitlement.CustomerEntitlementHistory, error) {
				received = input
				return entitlement.CustomerEntitlementHistory{}, nil
			},
		}, api.GetCustomerEntitlementHistoryParams{WindowSize: api.BillingEntitlementHistoryWindowSizePT1H})

		require.Equal(t, http.StatusOK, response.Code)
		require.Equal(t, meter.WindowSizeHour, received.WindowSize)
		require.Equal(t, time.UTC, received.TimeZone)
		require.JSONEq(t, `{"windowed_history": [], "burndown_history": []}`, response.Body.String())
	})

	t.Run("rejects an unknown time zone", func(t *testing.T) {
		response := serveGetCustomerEntitlementHistory(t, fakeService{
			history: func(context.Context, entitlement.GetCustomerEntitlementHistoryInput) (entitlement.CustomerEntitlementHistory, error) {
				t.Fatal("service must not be called")
				return entitlement.CustomerEntitlementHistory{}, nil
			},
		}, api.GetCustomerEntitlementHistoryParams{
			WindowSize: api.BillingEntitlementHistoryWindowSizePT1H,
			TimeZone:   lo.ToPtr("Mars/Olympus"),
		})

		require.Equal(t, http.StatusBadRequest, response.Code)
		require.Contains(t, response.Body.String(), "time_zone")
	})

	t.Run("rejects an unknown window size", func(t *testing.T) {
		response := serveGetCustomerEntitlementHistory(t, fakeService{}, api.GetCustomerEntitlementHistoryParams{
			WindowSize: api.BillingEntitlementHistoryWindowSize("PT1M"),
		})

		require.Equal(t, http.StatusBadRequest, response.Code)
		require.Contains(t, response.Body.String(), "window_size")
	})

	failing := func(err error) fakeService {
		return fakeService{
			history: func(context.Context, entitlement.GetCustomerEntitlementHistoryInput) (entitlement.CustomerEntitlementHistory, error) {
				return entitlement.CustomerEntitlementHistory{}, err
			},
		}
	}

	params := api.GetCustomerEntitlementHistoryParams{WindowSize: api.BillingEntitlementHistoryWindowSizePT1H}

	t.Run("maps a missing customer to 404", func(t *testing.T) {
		response := serveGetCustomerEntitlementHistory(t, failing(models.NewGenericNotFoundError(errors.New("customer not found"))), params)

		require.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("maps a missing entitlement to 404", func(t *testing.T) {
		response := serveGetCustomerEntitlementHistory(t, failing(&entitlement.NotFoundError{EntitlementID: models.NamespacedID{Namespace: testNamespace, ID: testEntitlementID}}), params)

		require.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("maps a deleted customer to 409", func(t *testing.T) {
		response := serveGetCustomerEntitlementHistory(t, failing(models.NewGenericConflictError(errors.New("customer is deleted"))), params)

		require.Equal(t, http.StatusConflict, response.Code)
	})

	t.Run("maps a non-metered entitlement to 400", func(t *testing.T) {
		response := serveGetCustomerEntitlementHistory(t, failing(&entitlement.WrongTypeError{Expected: entitlement.EntitlementTypeMetered, Actual: entitlement.EntitlementTypeBoolean}), params)

		require.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("maps a validation error to 400", func(t *testing.T) {
		response := serveGetCustomerEntitlementHistory(t, failing(models.NewGenericValidationError(errors.New("from cannot be before measurement start"))), params)

		require.Equal(t, http.StatusBadRequest, response.Code)
	})
}
