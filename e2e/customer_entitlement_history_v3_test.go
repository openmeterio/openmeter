package e2e

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

func TestV3GetCustomerEntitlementHistory(t *testing.T) {
	c := newV3Client(t)
	v1 := initClient(t)

	feature := createMeteredFeature(t, c, "ent_history")

	mtr, err := c.Meters.Get(t.Context(), feature.Meter.ID)
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, mtr)

	createCustomer := func(t *testing.T, keyPrefix string) *v3sdk.Customer {
		t.Helper()

		key := uniqueKey(keyPrefix)
		cust, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:  key,
			Name: "Entitlement History Customer " + key,
			UsageAttribution: &v3sdk.CustomerUsageAttribution{
				SubjectKeys: []string{key},
			},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, cust)

		return cust
	}

	// The v3 create endpoint is not available yet, so entitlements are seeded
	// through the legacy v2 customer entitlement endpoint. Measurement starts at
	// the top of the hour so the hourly windows are not cut off by it.
	createMeteredEntitlement := func(t *testing.T, customerID string) string {
		t.Helper()

		var interval api.RecurringPeriodInterval
		require.NoError(t, interval.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumMONTH))

		var measureUsageFrom api.MeasureUsageFrom
		require.NoError(t, measureUsageFrom.FromMeasureUsageFromTime(time.Now().Truncate(time.Hour)))

		var body api.CreateCustomerEntitlementV2JSONRequestBody
		require.NoError(t, body.FromEntitlementMeteredV2CreateInputs(api.EntitlementMeteredV2CreateInputs{
			Type:      api.EntitlementMeteredV2CreateInputsTypeMetered,
			FeatureId: lo.ToPtr(feature.ID),
			UsagePeriod: api.RecurringPeriodCreateInput{
				Interval: interval,
			},
			MeasureUsageFrom: &measureUsageFrom,
			IssueAfterReset:  lo.ToPtr(100.0),
		}))

		res, err := v1.CreateCustomerEntitlementV2WithResponse(t.Context(), customerID, body)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, res.StatusCode(), "Invalid status code [response_body=%s]", string(res.Body))

		created, err := res.JSON201.AsEntitlementMeteredV2()
		require.NoError(t, err)

		return created.Id
	}

	cust := createCustomer(t, "ent_history_customer")
	entitlementID := createMeteredEntitlement(t, cust.ID)

	// given usage recorded against the customer's subject
	for range 2 {
		ev := event.New()
		ev.SetID(ulid.Make().String())
		ev.SetSource("e2e")
		ev.SetType(mtr.EventType)
		ev.SetSubject(cust.Key)
		ev.SetTime(time.Now())
		require.NoError(t, ev.SetData("application/json", map[string]string{"value": "1"}))

		res, err := v1.IngestEventWithResponse(t.Context(), ev)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, res.StatusCode(), "Invalid status code [response_body=%s]", string(res.Body))
	}

	ctx := t.Context()
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		access, err := c.Entitlements.GetCustomerAccess(ctx, cust.ID, feature.Key, v3sdk.GetCustomerEntitlementAccessParams{
			Expand: []v3sdk.EntitlementAccessExpand{v3sdk.EntitlementAccessExpandValue},
		})
		require.NoError(t, err)
		require.NotNil(t, access.Value)
		assert.Equal(t, v3sdk.Numeric("2"), access.Value.Usage)
	}, time.Minute, time.Second)

	hourly := v3sdk.GetCustomerEntitlementHistoryParams{WindowSize: v3sdk.EntitlementHistoryWindowSizeHour}

	t.Run("returns the windowed and burndown history", func(t *testing.T) {
		history, err := c.Customers.Entitlements.GetHistory(t.Context(), cust.ID, entitlementID, hourly)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, history)

		require.NotEmpty(t, history.WindowedHistory)
		usage := 0.0
		for _, window := range history.WindowedHistory {
			require.True(t, window.Period.From.Before(window.Period.To))
			value, err := strconv.ParseFloat(window.Usage, 64)
			require.NoError(t, err)
			usage += value
		}
		require.Equal(t, 2.0, usage)

		// The default grant is consumed by the usage in the last segment.
		require.NotEmpty(t, history.BurndownHistory)
		last := history.BurndownHistory[len(history.BurndownHistory)-1]
		require.Equal(t, v3sdk.Numeric("100"), last.Balance.Start)
		require.Equal(t, v3sdk.Numeric("98"), last.Balance.End)
		require.Equal(t, time.UTC, last.Period.To.Location())
		require.Len(t, last.GrantBalances.End, 1)
		require.Len(t, last.GrantUsages, 1)
	})

	t.Run("rejects an unknown time zone", func(t *testing.T) {
		params := hourly
		params.TimeZone = lo.ToPtr("Mars/Olympus")

		_, err := c.Customers.Entitlements.GetHistory(t.Context(), cust.ID, entitlementID, params)
		requireProblem(t, err, http.StatusBadRequest)
	})

	t.Run("rejects a range starting before measurement", func(t *testing.T) {
		params := hourly
		params.From = lo.ToPtr(time.Now().Add(-48 * time.Hour))

		_, err := c.Customers.Entitlements.GetHistory(t.Context(), cust.ID, entitlementID, params)
		requireProblem(t, err, http.StatusBadRequest)
	})

	t.Run("returns 404 for an entitlement of another customer", func(t *testing.T) {
		other := createCustomer(t, "ent_history_other")
		otherEntitlementID := createMeteredEntitlement(t, other.ID)

		_, err := c.Customers.Entitlements.GetHistory(t.Context(), cust.ID, otherEntitlementID, hourly)
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("returns 404 for an unknown entitlement", func(t *testing.T) {
		_, err := c.Customers.Entitlements.GetHistory(t.Context(), cust.ID, ulid.Make().String(), hourly)
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("returns 404 for an unknown customer", func(t *testing.T) {
		_, err := c.Customers.Entitlements.GetHistory(t.Context(), ulid.Make().String(), entitlementID, hourly)
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("returns 409 for a deleted customer", func(t *testing.T) {
		deleted := createCustomer(t, "ent_history_deleted")
		deletedEntitlementID := createMeteredEntitlement(t, deleted.ID)

		// A customer can only be deleted once its entitlements are gone.
		res, err := v1.DeleteCustomerEntitlementV2WithResponse(t.Context(), deleted.ID, deletedEntitlementID)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, res.StatusCode(), "Invalid status code [response_body=%s]", string(res.Body))
		c.requireStatus(http.StatusNoContent, c.Customers.Delete(t.Context(), deleted.ID))

		_, err = c.Customers.Entitlements.GetHistory(t.Context(), deleted.ID, deletedEntitlementID, hourly)
		requireProblem(t, err, http.StatusConflict)
	})
}
