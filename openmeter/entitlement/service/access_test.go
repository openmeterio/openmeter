package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestGetAccess(t *testing.T) {
	t.Run("Should return empty access if no entitlements are found", func(t *testing.T) {
		conn, deps := setupDependecies(t)
		defer deps.Teardown()

		access, err := conn.GetAccess(context.Background(), "test", "test")
		require.NoError(t, err)
		require.Equal(t, access, entitlement.Access{})
	})

	t.Run("Should return access for a single entitlement", func(t *testing.T) {
		conn, deps := setupDependecies(t)
		defer deps.Teardown()

		now := testutils.GetRFC3339Time(t, "2025-01-01T00:00:00Z")

		clock.SetTime(now)
		defer clock.ResetTime()

		subjectKey := "test"
		featureKey := "test"
		ns := "ns1"

		// Lets set up a feature and an entitlement
		feat, err := deps.featureRepo.CreateFeature(context.Background(), feature.CreateFeatureInputs{
			Key:       featureKey,
			Name:      "test",
			Namespace: ns,
			MeterSlug: lo.ToPtr("meter1"),
		})
		require.NoError(t, err)
		require.NotNil(t, feat)

		// Let's create a bool entitlement
		ent, err := conn.CreateEntitlement(context.Background(), entitlement.CreateEntitlementInputs{
			Namespace:       ns,
			SubjectKey:      subjectKey,
			FeatureKey:      &featureKey,
			FeatureID:       &feat.ID,
			EntitlementType: entitlement.EntitlementTypeBoolean,
		})
		require.NoError(t, err)
		require.NotNil(t, ent)

		// Let's pass some time
		clock.SetTime(clock.Now().Add(time.Hour))

		// Let's get the access
		access, err := conn.GetAccess(context.Background(), ns, subjectKey)
		require.NoError(t, err)
		require.Len(t, access.Entitlements, 1)
		require.NotNil(t, access.Entitlements[featureKey])
		require.Equal(t, access.Entitlements[featureKey].Value.HasAccess(), true)
		require.Equal(t, access.Entitlements[featureKey].ID, ent.ID)
	})

	t.Run("Should return access for multiple entitlements (< than max concurrency)", func(t *testing.T) {
		conn, deps := setupDependecies(t)
		defer deps.Teardown()

		now := testutils.GetRFC3339Time(t, "2025-01-01T00:00:00Z")

		clock.SetTime(now)
		defer clock.ResetTime()

		subjectKey := "test"
		ns := "ns1"

		count := 5
		entIds := make([]string, count)
		for i := 0; i < count; i++ {
			feat, err := deps.featureRepo.CreateFeature(context.Background(), feature.CreateFeatureInputs{
				Key:       fmt.Sprintf("test-%d", i),
				Name:      "test",
				Namespace: ns,
				MeterSlug: lo.ToPtr("meter1"),
			})
			require.NoError(t, err)
			require.NotNil(t, feat)

			ent, err := conn.CreateEntitlement(context.Background(), entitlement.CreateEntitlementInputs{
				Namespace:       ns,
				SubjectKey:      subjectKey,
				FeatureKey:      lo.ToPtr(fmt.Sprintf("test-%d", i)),
				FeatureID:       &feat.ID,
				EntitlementType: entitlement.EntitlementTypeBoolean,
			})
			require.NoError(t, err)
			require.NotNil(t, ent)

			entIds[i] = ent.ID
		}

		// Let's pass some time
		clock.SetTime(clock.Now().Add(time.Hour))

		// Let's get the access
		access, err := conn.GetAccess(context.Background(), ns, subjectKey)
		require.NoError(t, err)
		require.Len(t, access.Entitlements, count)
		for _, ent := range access.Entitlements {
			require.Equal(t, ent.Value.HasAccess(), true)
			require.Contains(t, entIds, ent.ID)
		}
	})

	t.Run("Should return access for multiple entitlements of multiple types", func(t *testing.T) {
		conn, deps := setupDependecies(t)
		defer deps.Teardown()

		now := testutils.GetRFC3339Time(t, "2025-01-01T00:00:00Z")

		clock.SetTime(now)
		defer clock.ResetTime()
		subjectKey := "test"
		ns := "ns1"

		// Let's make a bool entitlement
		feat, err := deps.featureRepo.CreateFeature(context.Background(), feature.CreateFeatureInputs{
			Key:       "test-bool",
			Name:      "test",
			Namespace: ns,
			MeterSlug: lo.ToPtr("meter1"),
		})
		require.NoError(t, err)
		require.NotNil(t, feat)

		ent, err := conn.CreateEntitlement(context.Background(), entitlement.CreateEntitlementInputs{
			Namespace:       ns,
			SubjectKey:      subjectKey,
			FeatureKey:      lo.ToPtr("test-bool"),
			FeatureID:       &feat.ID,
			EntitlementType: entitlement.EntitlementTypeBoolean,
		})
		require.NoError(t, err)
		require.NotNil(t, ent)

		// Let's make a static entitlement
		feat, err = deps.featureRepo.CreateFeature(context.Background(), feature.CreateFeatureInputs{
			Key:       "test-static",
			Name:      "test",
			Namespace: ns,
			MeterSlug: lo.ToPtr("meter1"),
		})
		require.NoError(t, err)
		require.NotNil(t, feat)

		ent, err = conn.CreateEntitlement(context.Background(), entitlement.CreateEntitlementInputs{
			Namespace:       ns,
			SubjectKey:      subjectKey,
			FeatureKey:      lo.ToPtr("test-static"),
			FeatureID:       &feat.ID,
			EntitlementType: entitlement.EntitlementTypeStatic,
			Config:          []byte(`{"value": 10}`),
		})
		require.NoError(t, err)
		require.NotNil(t, ent)

		// Let's make a metered entitlement
		feat, err = deps.featureRepo.CreateFeature(context.Background(), feature.CreateFeatureInputs{
			Key:       "test-metered",
			Name:      "test",
			Namespace: ns,
			MeterSlug: lo.ToPtr("meter1"),
		})
		require.NoError(t, err)
		require.NotNil(t, feat)

		ent, err = conn.CreateEntitlement(context.Background(), entitlement.CreateEntitlementInputs{
			Namespace:       ns,
			SubjectKey:      subjectKey,
			FeatureKey:      lo.ToPtr("test-metered"),
			FeatureID:       &feat.ID,
			EntitlementType: entitlement.EntitlementTypeMetered,
			UsagePeriod: &entitlement.UsagePeriod{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now,
			},
			IssueAfterReset: lo.ToPtr(10.0),
		})
		require.NoError(t, err)
		require.NotNil(t, ent)

		// We need to add an event so streming mock finds the meter
		deps.streamingConnector.AddSimpleEvent("meter1", 1, now)

		// Let's pass some time
		clock.SetTime(clock.Now().Add(time.Hour))

		// Let's get the access
		access, err := conn.GetAccess(context.Background(), ns, subjectKey)
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
		conn, deps := setupDependecies(t)
		defer deps.Teardown()

		now := testutils.GetRFC3339Time(t, "2025-01-01T00:00:00Z")

		clock.SetTime(now)
		defer clock.ResetTime()

		subjectKey := "test"
		ns := "ns1"

		count := 20
		entIds := make([]string, count)
		for i := 0; i < count; i++ {
			feat, err := deps.featureRepo.CreateFeature(context.Background(), feature.CreateFeatureInputs{
				Key:       fmt.Sprintf("test-%d", i),
				Name:      "test",
				Namespace: ns,
				MeterSlug: lo.ToPtr("meter1"),
			})
			require.NoError(t, err)
			require.NotNil(t, feat)

			ent, err := conn.CreateEntitlement(context.Background(), entitlement.CreateEntitlementInputs{
				Namespace:       ns,
				SubjectKey:      subjectKey,
				FeatureKey:      lo.ToPtr(fmt.Sprintf("test-%d", i)),
				FeatureID:       &feat.ID,
				EntitlementType: entitlement.EntitlementTypeBoolean,
			})
			require.NoError(t, err)
			require.NotNil(t, ent)

			entIds[i] = ent.ID
		}

		// Let's pass some time
		clock.SetTime(clock.Now().Add(time.Hour))

		// Let's get the access
		access, err := conn.GetAccess(context.Background(), ns, subjectKey)
		require.NoError(t, err)
		require.Len(t, access.Entitlements, count)
		for _, ent := range access.Entitlements {
			require.Equal(t, ent.Value.HasAccess(), true)
			require.Contains(t, entIds, ent.ID)
		}
	})
}
