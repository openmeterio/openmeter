package service_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestGetAccess(t *testing.T) {
	conn, deps := setupDependecies(t)
	defer deps.Teardown()

	t.Run("Should return empty access if no entitlements are found", func(t *testing.T) {
		access, err := conn.GetAccess(t.Context(), "test", "test")
		require.NoError(t, err)
		require.Equal(t, access, entitlement.Access{})
	})

	t.Run("Should return access for a single entitlement", func(t *testing.T) {
		namespace := "ns1"

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

		// First, create the subject and the customer
		randName := testutils.NameGenerator.Generate()

		featureKey := randName.Key

		// create customer and subject
		cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

		// Then set up a feature and an entitlement
		feat, err := deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
			Key:       randName.Key,
			Name:      randName.Name,
			Namespace: namespace,
			MeterSlug: lo.ToPtr(mtr.Key),
		})
		require.NoError(t, err)
		require.NotNil(t, feat)

		// Let's create a bool entitlement
		ent, err := conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
			Namespace:        namespace,
			UsageAttribution: cust.GetUsageAttribution(),
			FeatureKey:       &featureKey,
			FeatureID:        &feat.ID,
			EntitlementType:  entitlement.EntitlementTypeBoolean,
		}, nil)
		require.NoError(t, err)
		require.NotNil(t, ent)

		// Let's pass some time
		clock.SetTime(clock.Now().Add(time.Hour))

		// Let's get the access
		access, err := conn.GetAccess(t.Context(), namespace, cust.ID)
		require.NoError(t, err)
		require.Len(t, access.Entitlements, 1)
		require.NotNil(t, access.Entitlements[featureKey])
		require.Equal(t, access.Entitlements[featureKey].Value.HasAccess(), true)
		require.Equal(t, access.Entitlements[featureKey].ID, ent.ID)
	})

	t.Run("Should return access for multiple entitlements (< than max concurrency)", func(t *testing.T) {
		namespace := "ns2"

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

		// First, create the subject and the customer
		randName := testutils.NameGenerator.Generate()

		// create customer and subject
		cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

		count := 5
		entIds := make([]string, count)
		for i := 0; i < count; i++ {
			feat, err := deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
				Key:       fmt.Sprintf("test-%d", i),
				Name:      "test",
				Namespace: namespace,
				MeterSlug: lo.ToPtr(mtr.Key),
			})
			require.NoError(t, err)
			require.NotNil(t, feat)

			ent, err := conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
				Namespace:        namespace,
				UsageAttribution: cust.GetUsageAttribution(),
				FeatureKey:       lo.ToPtr(fmt.Sprintf("test-%d", i)),
				FeatureID:        &feat.ID,
				EntitlementType:  entitlement.EntitlementTypeBoolean,
			}, nil)
			require.NoError(t, err)
			require.NotNil(t, ent)

			entIds[i] = ent.ID
		}

		// Let's pass some time
		clock.SetTime(clock.Now().Add(time.Hour))

		// Let's get the access
		access, err := conn.GetAccess(t.Context(), namespace, cust.ID)
		require.NoError(t, err)
		require.Len(t, access.Entitlements, count)
		for _, ent := range access.Entitlements {
			require.Equal(t, ent.Value.HasAccess(), true)
			require.Contains(t, entIds, ent.ID)
		}
	})

	t.Run("Should return access for multiple entitlements of multiple types", func(t *testing.T) {
		namespace := "ns3"

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

		// First, create the subject
		randName := testutils.NameGenerator.Generate()

		// create customer and subject
		cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

		// Let's make a bool entitlement
		feat, err := deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
			Key:       "test-bool",
			Name:      "test",
			Namespace: namespace,
			MeterSlug: lo.ToPtr(mtr.Key),
		})
		require.NoError(t, err)
		require.NotNil(t, feat)

		ent, err := conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
			Namespace:        namespace,
			UsageAttribution: cust.GetUsageAttribution(),
			FeatureKey:       lo.ToPtr("test-bool"),
			FeatureID:        &feat.ID,
			EntitlementType:  entitlement.EntitlementTypeBoolean,
		}, nil)
		require.NoError(t, err)
		require.NotNil(t, ent)

		// Let's make a static entitlement
		feat, err = deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
			Key:       "test-static",
			Name:      "test",
			Namespace: namespace,
			MeterSlug: lo.ToPtr(mtr.Key),
		})
		require.NoError(t, err)
		require.NotNil(t, feat)

		ent, err = conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
			Namespace:        namespace,
			UsageAttribution: cust.GetUsageAttribution(),
			FeatureKey:       lo.ToPtr("test-static"),
			FeatureID:        &feat.ID,
			EntitlementType:  entitlement.EntitlementTypeStatic,
			Config:           lo.ToPtr(`{"value": 10}`),
		}, nil)
		require.NoError(t, err)
		require.NotNil(t, ent)

		// Let's make a metered entitlement
		feat, err = deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
			Key:       "test-metered",
			Name:      "test",
			Namespace: namespace,
			MeterSlug: lo.ToPtr(mtr.Key),
		})
		require.NoError(t, err)
		require.NotNil(t, feat)

		ent, err = conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
			Namespace:        namespace,
			UsageAttribution: cust.GetUsageAttribution(),
			FeatureKey:       lo.ToPtr("test-metered"),
			FeatureID:        &feat.ID,
			EntitlementType:  entitlement.EntitlementTypeMetered,
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now,
			})),
			IssueAfterReset: lo.ToPtr(10.0),
		}, nil)
		require.NoError(t, err)
		require.NotNil(t, ent)

		// We need to add an event so streming mock finds the meter
		deps.streamingConnector.AddSimpleEvent(mtr.Key, 1, now)

		// Let's pass some time
		clock.SetTime(clock.Now().Add(time.Hour))

		// Let's get the access
		access, err := conn.GetAccess(t.Context(), namespace, cust.ID)
		require.NoError(t, err)

		require.Len(t, access.Entitlements, 3)
		require.NotNil(t, access.Entitlements["test-bool"])
		require.NotNil(t, access.Entitlements["test-static"])
		require.NotNil(t, access.Entitlements["test-metered"])

		require.Equal(t, access.Entitlements["test-bool"].Value.HasAccess(), true)
		require.Equal(t, access.Entitlements["test-static"].Value.HasAccess(), true)
		require.Equal(t, access.Entitlements["test-metered"].Value.HasAccess(), true)
	})

	t.Run("Should return access for multiple entitlements (> than max concurrency)", func(t *testing.T) {
		namespace := "ns4"

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

		randName := testutils.NameGenerator.Generate()

		// create customer and subject
		cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

		count := 20
		entIds := make([]string, count)
		for i := 0; i < count; i++ {
			feat, err := deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
				Key:       fmt.Sprintf("test-%d", i),
				Name:      "test",
				Namespace: namespace,
				MeterSlug: lo.ToPtr(mtr.Key),
			})
			require.NoError(t, err)
			require.NotNil(t, feat)

			ent, err := conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
				Namespace:        namespace,
				UsageAttribution: cust.GetUsageAttribution(),
				FeatureKey:       lo.ToPtr(fmt.Sprintf("test-%d", i)),
				FeatureID:        &feat.ID,
				EntitlementType:  entitlement.EntitlementTypeBoolean,
			}, nil)
			require.NoError(t, err)
			require.NotNil(t, ent)

			entIds[i] = ent.ID
		}

		// Let's pass some time
		clock.SetTime(clock.Now().Add(time.Hour))

		// Let's get the access
		access, err := conn.GetAccess(t.Context(), namespace, cust.ID)
		require.NoError(t, err)
		require.Len(t, access.Entitlements, count)
		for _, ent := range access.Entitlements {
			require.Equal(t, ent.Value.HasAccess(), true)
			require.Contains(t, entIds, ent.ID)
		}
	})
}
