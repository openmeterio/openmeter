package service_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func TestGetEntitlementOfCustomerAt(t *testing.T) {
	conn, deps := setupDependecies(t)
	defer deps.Teardown()

	namespace := "ns-get-entitlement-of-customer-at"
	now := testutils.GetRFC3339Time(t, "2025-01-01T00:00:00Z")

	clock.SetTime(now)
	defer clock.ResetTime()

	mtr, err := deps.meterService.CreateMeter(t.Context(), meter.CreateMeterInput{
		Namespace:     namespace,
		Name:          "Meter 1",
		Key:           "meter1",
		Description:   nil,
		Aggregation:   meter.MeterAggregationSum,
		EventType:     "test",
		EventFrom:     nil,
		ValueProperty: lo.ToPtr("$.value"),
		GroupBy:       nil,
	})
	require.NoError(t, err)
	require.NotNil(t, mtr)
	createMeterInPG(t, deps.dbClient, mtr)

	cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, "cust-1", "Customer 1")

	feat, err := deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
		Key:       "free_plan_usage",
		Name:      "Free plan usage",
		Namespace: namespace,
		MeterID:   &mtr.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, feat)

	ent, err := conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
		Namespace:        namespace,
		UsageAttribution: cust.GetUsageAttribution(),
		FeatureKey:       &feat.Key,
		FeatureID:        &feat.ID,
		EntitlementType:  entitlement.EntitlementTypeBoolean,
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, ent)

	t.Run("Should resolve entitlement by feature key", func(t *testing.T) {
		res, err := conn.GetEntitlementOfCustomerAt(t.Context(), namespace, cust.ID, feat.Key, clock.Now().Add(time.Hour))
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, ent.ID, res.ID)
		require.Equal(t, feat.Key, res.FeatureKey)
		require.Equal(t, cust.ID, res.CustomerID)
	})

	t.Run("Should resolve entitlement by entitlement ID", func(t *testing.T) {
		res, err := conn.GetEntitlementOfCustomerAt(t.Context(), namespace, cust.ID, ent.ID, clock.Now().Add(time.Hour))
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, ent.ID, res.ID)
		require.Equal(t, feat.Key, res.FeatureKey)
		require.Equal(t, cust.ID, res.CustomerID)
	})
}
