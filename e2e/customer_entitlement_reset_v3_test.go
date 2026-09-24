package e2e

import (
	"encoding/json"
	"net/http"
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

func TestV3ResetCustomerEntitlementUsage(t *testing.T) {
	c := newV3Client(t)
	v1 := initClient(t)

	feature := createMeteredFeature(t, c, "ent_reset")

	mtr, err := c.Meters.Get(t.Context(), feature.Meter.ID)
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, mtr)

	createCustomer := func(t *testing.T, keyPrefix string) *v3sdk.Customer {
		t.Helper()

		key := uniqueKey(keyPrefix)
		cust, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:  key,
			Name: "Entitlement Reset Customer " + key,
			UsageAttribution: &v3sdk.CustomerUsageAttribution{
				SubjectKeys: []string{key},
			},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, cust)

		return cust
	}

	// The v3 create endpoint is not available yet, so entitlements are seeded
	// through the legacy v2 customer entitlement endpoint.
	createEntitlement := func(t *testing.T, customerID string, body api.CreateCustomerEntitlementV2JSONRequestBody) string {
		t.Helper()

		res, err := v1.CreateCustomerEntitlementV2WithResponse(t.Context(), customerID, body)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, res.StatusCode(), "Invalid status code [response_body=%s]", string(res.Body))

		var created struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(res.Body, &created))

		return created.ID
	}

	createMeteredEntitlement := func(t *testing.T, customerID string) string {
		t.Helper()

		var interval api.RecurringPeriodInterval
		require.NoError(t, interval.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumMONTH))

		var body api.CreateCustomerEntitlementV2JSONRequestBody
		require.NoError(t, body.FromEntitlementMeteredV2CreateInputs(api.EntitlementMeteredV2CreateInputs{
			Type:      api.EntitlementMeteredV2CreateInputsTypeMetered,
			FeatureId: lo.ToPtr(feature.ID),
			UsagePeriod: api.RecurringPeriodCreateInput{
				Interval: interval,
			},
			IssueAfterReset: lo.ToPtr(100.0),
		}))

		return createEntitlement(t, customerID, body)
	}

	meteredValue := func(t *testing.T, customerID string) *v3sdk.EntitlementAccessValue {
		t.Helper()

		access, err := c.Entitlements.GetCustomerAccess(t.Context(), customerID, feature.Key, v3sdk.GetCustomerEntitlementAccessParams{
			Expand: []v3sdk.EntitlementAccessExpand{v3sdk.EntitlementAccessExpandValue},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, access)
		require.NotNil(t, access.Value)

		return access.Value
	}

	cust := createCustomer(t, "ent_reset_customer")
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

	// A reset is rejected within the minute of the previous one, which the
	// entitlement's creation counts as, so wait for the next minute to start.
	time.Sleep(time.Until(time.Now().Truncate(time.Minute).Add(time.Minute)))

	t.Run("starts a new usage period", func(t *testing.T) {
		before := meteredValue(t, cust.ID)
		require.Equal(t, v3sdk.Numeric("2"), before.Usage)
		require.Equal(t, v3sdk.Numeric("98"), before.Balance)

		c.requireStatus(http.StatusNoContent, c.Customers.Entitlements.ResetUsage(t.Context(), cust.ID, entitlementID, nil))

		after := meteredValue(t, cust.ID)
		require.Equal(t, v3sdk.Numeric("0"), after.Usage)
		require.Equal(t, v3sdk.Numeric("100"), after.Balance)
	})

	t.Run("rejects a reset in the future", func(t *testing.T) {
		err := c.Customers.Entitlements.ResetUsage(t.Context(), cust.ID, entitlementID, &v3sdk.ResetCustomerEntitlementUsageRequest{
			EffectiveAt: lo.ToPtr(time.Now().Add(time.Hour)),
		})
		requireProblem(t, err, http.StatusBadRequest)
	})

	t.Run("rejects a reset before the last reset", func(t *testing.T) {
		err := c.Customers.Entitlements.ResetUsage(t.Context(), cust.ID, entitlementID, &v3sdk.ResetCustomerEntitlementUsageRequest{
			EffectiveAt: lo.ToPtr(time.Now().Add(-time.Hour)),
		})
		requireProblem(t, err, http.StatusBadRequest)
	})

	t.Run("rejects a non-metered entitlement", func(t *testing.T) {
		boolKey := uniqueKey("ent_reset_boolean")
		boolFeature, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
			Key:  boolKey,
			Name: "Boolean Feature " + boolKey,
		})
		c.requireStatus(http.StatusCreated, err)

		var body api.CreateCustomerEntitlementV2JSONRequestBody
		require.NoError(t, body.FromEntitlementBooleanCreateInputs(api.EntitlementBooleanCreateInputs{
			Type:      api.EntitlementBooleanCreateInputsTypeBoolean,
			FeatureId: lo.ToPtr(boolFeature.ID),
		}))
		boolEntitlementID := createEntitlement(t, cust.ID, body)

		err = c.Customers.Entitlements.ResetUsage(t.Context(), cust.ID, boolEntitlementID, nil)
		requireProblem(t, err, http.StatusBadRequest)
	})

	t.Run("returns 404 for an entitlement of another customer", func(t *testing.T) {
		other := createCustomer(t, "ent_reset_other")
		otherEntitlementID := createMeteredEntitlement(t, other.ID)

		err := c.Customers.Entitlements.ResetUsage(t.Context(), cust.ID, otherEntitlementID, nil)
		requireProblem(t, err, http.StatusNotFound)

		// The other customer's usage period is left untouched.
		require.Equal(t, v3sdk.Numeric("100"), meteredValue(t, other.ID).Balance)
	})

	t.Run("returns 404 for an unknown entitlement", func(t *testing.T) {
		err := c.Customers.Entitlements.ResetUsage(t.Context(), cust.ID, ulid.Make().String(), nil)
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("returns 404 for an unknown customer", func(t *testing.T) {
		err := c.Customers.Entitlements.ResetUsage(t.Context(), ulid.Make().String(), entitlementID, nil)
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("returns 409 for a deleted customer", func(t *testing.T) {
		deleted := createCustomer(t, "ent_reset_deleted")
		deletedEntitlementID := createMeteredEntitlement(t, deleted.ID)

		// A customer can only be deleted once its entitlements are gone.
		res, err := v1.DeleteCustomerEntitlementV2WithResponse(t.Context(), deleted.ID, deletedEntitlementID)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, res.StatusCode(), "Invalid status code [response_body=%s]", string(res.Body))
		c.requireStatus(http.StatusNoContent, c.Customers.Delete(t.Context(), deleted.ID))

		err = c.Customers.Entitlements.ResetUsage(t.Context(), deleted.ID, deletedEntitlementID, nil)
		requireProblem(t, err, http.StatusConflict)
	})
}
