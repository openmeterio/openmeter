package meteredentitlement_test

import (
	"context"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestEntitlementGrantOwnerAdapter(t *testing.T) {
	createFeatureAndCustomer := func(t *testing.T, deps *dependencies) (feature.Feature, *customer.Customer) {
		t.Helper()

		f, err := deps.featureRepo.CreateFeature(context.Background(), feature.CreateFeatureInputs{
			Name:      "f1",
			Key:       "f1",
			MeterSlug: lo.ToPtr(meterSlug),
			Namespace: namespace,
		})
		require.NoError(t, err)

		randName := testutils.NameGenerator.Generate()

		// create customer and subject
		cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

		return f, cust
	}

	t.Run("Should return the last reset time for the full period if there are no resets", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		now := testutils.GetRFC3339Time(t, "2025-02-01T00:00:00Z")
		clock.SetTime(now)

		_, deps := setupConnector(t)
		defer deps.Teardown()
		f, c := createFeatureAndCustomer(t, deps)

		// Let's create an entitlement
		ent, err := deps.entitlementRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
			Namespace:        namespace,
			FeatureID:        f.ID,
			FeatureKey:       f.Key,
			UsageAttribution: c.GetUsageAttribution(),
			EntitlementType:  entitlement.EntitlementTypeMetered,
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   now,
			})),
		})
		require.NoError(t, err)

		// We do no resets...

		owner := models.NamespacedID{
			Namespace: namespace,
			ID:        ent.ID,
		}

		t.Run("Should return reset for period start if before the period", func(t *testing.T) {
			// We query for 4 days without reset
			timeline, err := deps.ownerConnector.GetResetTimelineInclusive(ctx, owner, timeutil.ClosedPeriod{
				From: now.AddDate(0, 0, 1),
				To:   now.AddDate(0, 0, 5),
			})
			require.NoError(t, err)

			require.Len(t, timeline.GetTimes(), 1)

			periods := timeline.GetClosedPeriods()
			require.Len(t, periods, 1)

			assert.Equal(t, now, periods[0].From)
			assert.Equal(t, now, periods[0].To)
		})

		t.Run("Should return reset for period start when coincides with period start", func(t *testing.T) {
			// We query for 4 days without reset
			timeline, err := deps.ownerConnector.GetResetTimelineInclusive(ctx, owner, timeutil.ClosedPeriod{
				From: now,
				To:   now.AddDate(0, 0, 5),
			})
			require.NoError(t, err)

			require.Len(t, timeline.GetTimes(), 1)

			periods := timeline.GetClosedPeriods()
			require.Len(t, periods, 1)

			assert.Equal(t, now, periods[0].From)
			assert.Equal(t, now, periods[0].To)
		})
	})

	t.Run("Should return manual reset time on the 3rd day", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		now := testutils.GetRFC3339Time(t, "2025-02-01T00:00:00Z")
		clock.SetTime(now)

		_, deps := setupConnector(t)
		defer deps.Teardown()
		f, c := createFeatureAndCustomer(t, deps)

		// Let's create an entitlement
		ent, err := deps.entitlementRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
			Namespace:        namespace,
			FeatureID:        f.ID,
			FeatureKey:       f.Key,
			UsageAttribution: c.GetUsageAttribution(),
			EntitlementType:  entitlement.EntitlementTypeMetered,
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   now,
			})),
		})
		require.NoError(t, err)

		// We do a single reset on the 3rd day
		resetTime := now.AddDate(0, 0, 3)
		err = deps.usageResetRepo.Save(ctx, meteredentitlement.UsageResetUpdate{
			NamespacedModel: models.NamespacedModel{
				Namespace: namespace,
			},
			ResetTime:           resetTime,
			Anchor:              now,
			EntitlementID:       ent.ID,
			UsagePeriodInterval: timeutil.RecurrencePeriodMonth.ISOString(),
		})
		require.NoError(t, err)

		owner := models.NamespacedID{
			Namespace: namespace,
			ID:        ent.ID,
		}

		// We query for 4 days without the reset included
		timeline, err := deps.ownerConnector.GetResetTimelineInclusive(ctx, owner, timeutil.ClosedPeriod{
			From: now.AddDate(0, 0, 1),
			To:   now.AddDate(0, 0, 5),
		})
		require.NoError(t, err)

		times := timeline.GetTimes()

		require.Len(t, times, 2)

		assert.Equal(t, now, times[0])
		assert.Equal(t, resetTime, times[1])
	})

	t.Run("Should find programmatic reset time in the period", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		now := testutils.GetRFC3339Time(t, "2025-02-01T00:00:00Z")
		clock.SetTime(now)

		_, deps := setupConnector(t)
		defer deps.Teardown()
		f, c := createFeatureAndCustomer(t, deps)

		// Let's create an entitlement
		ent, err := deps.entitlementRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
			Namespace:        namespace,
			FeatureID:        f.ID,
			FeatureKey:       f.Key,
			UsageAttribution: c.GetUsageAttribution(),
			EntitlementType:  entitlement.EntitlementTypeMetered,
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   now,
			})),
		})
		require.NoError(t, err)

		// We do no resets...

		owner := models.NamespacedID{
			Namespace: namespace,
			ID:        ent.ID,
		}

		// We query for 4 days without the reset included
		timeline, err := deps.ownerConnector.GetResetTimelineInclusive(ctx, owner, timeutil.ClosedPeriod{
			From: now.AddDate(0, 0, 1),
			To:   now.AddDate(0, 1, 5), // We query for more than usage period
		})
		require.NoError(t, err)

		times := timeline.GetTimes()

		require.Len(t, times, 2)

		assert.Equal(t, now, times[0])
		assert.Equal(t, now.AddDate(0, 1, 0), times[1])
	})

	t.Run("Should find programmatic reset time between manual resets", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		now := testutils.GetRFC3339Time(t, "2025-02-01T00:00:00Z")
		clock.SetTime(now)

		_, deps := setupConnector(t)
		defer deps.Teardown()
		f, c := createFeatureAndCustomer(t, deps)

		// Let's create an entitlement
		ent, err := deps.entitlementRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
			Namespace:        namespace,
			FeatureID:        f.ID,
			FeatureKey:       f.Key,
			UsageAttribution: c.GetUsageAttribution(),
			EntitlementType:  entitlement.EntitlementTypeMetered,
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   now,
			})),
		})
		require.NoError(t, err)

		// Let's do two resets, one before and one after the programmatic reset
		resetTime1 := now.AddDate(0, 0, 15)
		err = deps.usageResetRepo.Save(ctx, meteredentitlement.UsageResetUpdate{
			NamespacedModel: models.NamespacedModel{
				Namespace: namespace,
			},
			ResetTime:           resetTime1,
			Anchor:              now,
			EntitlementID:       ent.ID,
			UsagePeriodInterval: timeutil.RecurrencePeriodMonth.ISOString(),
		})
		require.NoError(t, err)

		resetTime2 := now.AddDate(0, 1, 3)
		err = deps.usageResetRepo.Save(ctx, meteredentitlement.UsageResetUpdate{
			NamespacedModel: models.NamespacedModel{
				Namespace: namespace,
			},
			ResetTime:           resetTime2,
			Anchor:              now,
			EntitlementID:       ent.ID,
			UsagePeriodInterval: timeutil.RecurrencePeriodMonth.ISOString(),
		})
		require.NoError(t, err)

		owner := models.NamespacedID{
			Namespace: namespace,
			ID:        ent.ID,
		}

		// We query for 4 days without the reset included
		timeline, err := deps.ownerConnector.GetResetTimelineInclusive(ctx, owner, timeutil.ClosedPeriod{
			From: now.AddDate(0, 0, 1),
			To:   now.AddDate(0, 1, 5), // We query for more than usage period
		})
		require.NoError(t, err)

		times := timeline.GetTimes()

		require.Len(t, times, 4)

		assert.Equal(t, now, times[0])
		assert.Equal(t, resetTime1, times[1])
		assert.Equal(t, now.AddDate(0, 1, 0), times[2])
		assert.Equal(t, resetTime2, times[3])
	})

	t.Run("Should respect if anchor has been changed during a reset - no programmatic reset", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		now := testutils.GetRFC3339Time(t, "2025-02-01T00:00:00Z")
		clock.SetTime(now)

		_, deps := setupConnector(t)
		defer deps.Teardown()
		f, c := createFeatureAndCustomer(t, deps)

		// Let's create an entitlement
		ent, err := deps.entitlementRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
			Namespace:        namespace,
			FeatureID:        f.ID,
			FeatureKey:       f.Key,
			UsageAttribution: c.GetUsageAttribution(),
			EntitlementType:  entitlement.EntitlementTypeMetered,
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   now,
			})),
		})
		require.NoError(t, err)

		// We do a single reset on the 10th day resetting the anchor
		resetTime := now.AddDate(0, 0, 10)
		err = deps.usageResetRepo.Save(ctx, meteredentitlement.UsageResetUpdate{
			NamespacedModel: models.NamespacedModel{
				Namespace: namespace,
			},
			ResetTime:           resetTime,
			Anchor:              resetTime,
			EntitlementID:       ent.ID,
			UsagePeriodInterval: timeutil.RecurrencePeriodMonth.ISOString(),
		})
		require.NoError(t, err)

		owner := models.NamespacedID{
			Namespace: namespace,
			ID:        ent.ID,
		}

		timeline, err := deps.ownerConnector.GetResetTimelineInclusive(ctx, owner, timeutil.ClosedPeriod{
			From: now.AddDate(0, 0, 1),
			To:   now.AddDate(0, 1, 5),
		})
		require.NoError(t, err)

		times := timeline.GetTimes()

		require.Len(t, times, 2)

		assert.Equal(t, now, times[0])
		assert.Equal(t, resetTime, times[1])
	})

	t.Run("Should return end of period if it coincides with a programmatic reset", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		now := testutils.GetRFC3339Time(t, "2025-02-01T00:00:00Z")
		clock.SetTime(now)

		_, deps := setupConnector(t)
		defer deps.Teardown()
		f, c := createFeatureAndCustomer(t, deps)

		// Let's create an entitlement
		ent, err := deps.entitlementRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
			Namespace:        namespace,
			FeatureID:        f.ID,
			FeatureKey:       f.Key,
			UsageAttribution: c.GetUsageAttribution(),
			EntitlementType:  entitlement.EntitlementTypeMetered,
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   now,
			})),
		})
		require.NoError(t, err)

		owner := models.NamespacedID{
			Namespace: namespace,
			ID:        ent.ID,
		}

		timeline, err := deps.ownerConnector.GetResetTimelineInclusive(ctx, owner, timeutil.ClosedPeriod{
			From: now.AddDate(0, 0, 1),
			To:   now.AddDate(0, 1, 0),
		})
		require.NoError(t, err)

		times := timeline.GetTimes()

		require.Len(t, times, 2)

		assert.Equal(t, now, times[0])
		assert.Equal(t, now.AddDate(0, 1, 0), times[1])
	})

	t.Run("Should return end of period if it coincides with a manual reset", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		now := testutils.GetRFC3339Time(t, "2025-02-01T00:00:00Z")
		clock.SetTime(now)

		_, deps := setupConnector(t)
		defer deps.Teardown()
		f, c := createFeatureAndCustomer(t, deps)

		// Let's create an entitlement
		ent, err := deps.entitlementRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
			Namespace:        namespace,
			FeatureID:        f.ID,
			FeatureKey:       f.Key,
			UsageAttribution: c.GetUsageAttribution(),
			EntitlementType:  entitlement.EntitlementTypeMetered,
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   now,
			})),
		})
		require.NoError(t, err)

		// We do a single reset on the 10th day resetting the anchor
		resetTime := now.AddDate(0, 0, 10)
		err = deps.usageResetRepo.Save(ctx, meteredentitlement.UsageResetUpdate{
			NamespacedModel: models.NamespacedModel{
				Namespace: namespace,
			},
			ResetTime:           resetTime,
			Anchor:              resetTime,
			EntitlementID:       ent.ID,
			UsagePeriodInterval: timeutil.RecurrencePeriodMonth.ISOString(),
		})
		require.NoError(t, err)

		owner := models.NamespacedID{
			Namespace: namespace,
			ID:        ent.ID,
		}

		timeline, err := deps.ownerConnector.GetResetTimelineInclusive(ctx, owner, timeutil.ClosedPeriod{
			From: now,
			To:   resetTime,
		})
		require.NoError(t, err)

		times := timeline.GetTimes()

		require.Len(t, times, 2)

		assert.Equal(t, now, times[0])
		assert.Equal(t, resetTime, times[1])
	})

	t.Run("Should return a single reset time if manual reset coincides with programmatic reset", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		now := testutils.GetRFC3339Time(t, "2025-02-01T00:00:00Z")
		clock.SetTime(now)

		_, deps := setupConnector(t)
		defer deps.Teardown()
		f, c := createFeatureAndCustomer(t, deps)

		// Let's create an entitlement
		ent, err := deps.entitlementRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
			Namespace:        namespace,
			FeatureID:        f.ID,
			FeatureKey:       f.Key,
			UsageAttribution: c.GetUsageAttribution(),
			EntitlementType:  entitlement.EntitlementTypeMetered,
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   now,
			})),
		})
		require.NoError(t, err)

		// We do a single reset on the 10th day resetting the anchor
		resetTime := now.AddDate(0, 1, 0)
		err = deps.usageResetRepo.Save(ctx, meteredentitlement.UsageResetUpdate{
			NamespacedModel: models.NamespacedModel{
				Namespace: namespace,
			},
			ResetTime:           resetTime,
			Anchor:              resetTime,
			EntitlementID:       ent.ID,
			UsagePeriodInterval: timeutil.RecurrencePeriodMonth.ISOString(),
		})
		require.NoError(t, err)

		owner := models.NamespacedID{
			Namespace: namespace,
			ID:        ent.ID,
		}

		timeline, err := deps.ownerConnector.GetResetTimelineInclusive(ctx, owner, timeutil.ClosedPeriod{
			From: now,
			To:   now.AddDate(0, 1, 1),
		})
		require.NoError(t, err)

		times := timeline.GetTimes()

		require.Len(t, times, 2)

		assert.Equal(t, now, times[0])
		assert.Equal(t, resetTime, times[1])
	})

	t.Run("Should handle changing usage period interval through resets", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		now := testutils.GetRFC3339Time(t, "2025-02-01T00:00:00Z")
		clock.SetTime(now)

		_, deps := setupConnector(t)
		defer deps.Teardown()
		f, c := createFeatureAndCustomer(t, deps)

		// Create an entitlement with monthly usage period
		ent, err := deps.entitlementRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
			Namespace:        namespace,
			FeatureID:        f.ID,
			FeatureKey:       f.Key,
			UsageAttribution: c.GetUsageAttribution(),
			EntitlementType:  entitlement.EntitlementTypeMetered,
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   now,
			})),
		})
		require.NoError(t, err)

		// Time travel to 1 month 15 days later
		timeAfterMonthAnd15Days := now.AddDate(0, 1, 15)
		clock.SetTime(timeAfterMonthAnd15Days)

		// Perform a reset that changes the usage period to weekly
		err = deps.usageResetRepo.Save(ctx, meteredentitlement.UsageResetUpdate{
			NamespacedModel: models.NamespacedModel{
				Namespace: namespace,
			},
			ResetTime:           timeAfterMonthAnd15Days,
			Anchor:              timeAfterMonthAnd15Days,
			EntitlementID:       ent.ID,
			UsagePeriodInterval: datetime.ISODurationString("P1W"),
		})
		require.NoError(t, err)

		// Time travel to 10 days later
		timeAfter25Days := timeAfterMonthAnd15Days.AddDate(0, 0, 10)
		clock.SetTime(timeAfter25Days)

		// Perform another reset that changes the usage period to daily
		err = deps.usageResetRepo.Save(ctx, meteredentitlement.UsageResetUpdate{
			NamespacedModel: models.NamespacedModel{
				Namespace: namespace,
			},
			ResetTime:           timeAfter25Days,
			Anchor:              timeAfter25Days,
			EntitlementID:       ent.ID,
			UsagePeriodInterval: datetime.ISODurationString("P1D"),
		})
		require.NoError(t, err)

		// Now let's fetch the entitlement again and check the usage period
		ent, err = deps.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{
			Namespace: namespace,
			ID:        ent.ID,
		})
		require.NoError(t, err)

		// Now let's assert all 3 usageperiods.
		// First lets assert the 3 vlaues
		inp1, _, err := ent.UsagePeriod.GetUsagePeriodInputAt(now)
		require.NoError(t, err)
		assert.Equal(t, datetime.ISODurationString("P1M"), inp1.GetValue().Interval.ISOString())
		assert.Equal(t, now, inp1.GetValue().Anchor)
		inp2, _, err := ent.UsagePeriod.GetUsagePeriodInputAt(timeAfterMonthAnd15Days)
		require.NoError(t, err)
		assert.Equal(t, datetime.ISODurationString("P1W"), inp2.GetValue().Interval.ISOString())
		assert.Equal(t, timeAfterMonthAnd15Days, inp2.GetValue().Anchor)
		inp3, _, err := ent.UsagePeriod.GetUsagePeriodInputAt(timeAfter25Days)
		require.NoError(t, err)
		assert.Equal(t, datetime.ISODurationString("P1D"), inp3.GetValue().Interval.ISOString())
		assert.Equal(t, timeAfter25Days, inp3.GetValue().Anchor)

		// Second, lets assert that the period resolution is correct (we'll query the period one minute after the resets)
		period1, err := ent.UsagePeriod.GetCurrentPeriodAt(now.Add(time.Minute))
		require.NoError(t, err)
		assert.Equal(t, timeutil.ClosedPeriod{
			From: now,
			To:   now.AddDate(0, 1, 0),
		}, period1)

		period2, err := ent.UsagePeriod.GetCurrentPeriodAt(timeAfterMonthAnd15Days.Add(time.Minute))
		require.NoError(t, err)
		assert.Equal(t, timeutil.ClosedPeriod{
			From: timeAfterMonthAnd15Days,
			To:   timeAfterMonthAnd15Days.AddDate(0, 0, 7),
		}, period2)

		period3, err := ent.UsagePeriod.GetCurrentPeriodAt(timeAfter25Days.Add(time.Minute))
		require.NoError(t, err)
		assert.Equal(t, timeutil.ClosedPeriod{
			From: timeAfter25Days,
			To:   timeAfter25Days.AddDate(0, 0, 1),
		}, period3)
	})

	t.Run("Should not create snapshots if underlying meter uses LATEST aggregation type", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		now := testutils.GetRFC3339Time(t, "2025-02-01T00:00:00Z")
		clock.SetTime(now)
		defer clock.ResetTime()

		conn, deps := setupConnector(t)
		defer deps.Teardown()

		latestMeterSlug := "latest_meter"

		// Create a meter with LATEST aggregation type
		require.NoError(t, deps.meterAdapter.ReplaceMeters(ctx, []meter.Meter{{
			ManagedResource: models.ManagedResource{
				ID: ulid.Make().String(),
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name: "Latest Meter",
			},
			Key:           latestMeterSlug,
			Aggregation:   meter.MeterAggregationLatest,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		}}))

		// Create feature with the LATEST meter
		f, err := deps.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
			Name:      "latest_feature",
			Key:       "latest_feature",
			MeterSlug: lo.ToPtr(latestMeterSlug),
			Namespace: namespace,
		})
		require.NoError(t, err)

		randName := testutils.NameGenerator.Generate()

		// create customer and subject
		c := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

		// Create entitlement

		ent, err := deps.entitlementRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
			Namespace:        namespace,
			FeatureID:        f.ID,
			FeatureKey:       f.Key,
			UsageAttribution: c.GetUsageAttribution(),
			EntitlementType:  entitlement.EntitlementTypeMetered,
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   now,
			})),
			IsSoftLimit:      lo.ToPtr(false),
			MeasureUsageFrom: &now,
		})
		require.NoError(t, err)

		owner := models.NamespacedID{
			Namespace: namespace,
			ID:        ent.ID,
		}

		// Create a grant
		_, err = deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
			OwnerID:     ent.ID,
			Namespace:   namespace,
			Amount:      1000,
			Priority:    1,
			EffectiveAt: now,
			ExpiresAt:   lo.ToPtr(now.AddDate(0, 0, 10)),
		})
		require.NoError(t, err)

		// Add usage events
		deps.streamingConnector.AddSimpleEvent(latestMeterSlug, 100, now.Add(time.Minute))
		deps.streamingConnector.AddSimpleEvent(latestMeterSlug, 200, now.Add(time.Minute*2))

		// Move time forward beyond grace period to trigger snapshot creation for regular meters
		queryTime := now.AddDate(0, 0, 8) // 8 days later, beyond grace period
		clock.SetTime(queryTime)

		// Get snapshots count before calling GetEntitlementBalance
		snapshotsBefore, err := deps.balanceSnapshotService.GetLatestValidAt(ctx, owner, queryTime)
		// For LATEST aggregation, we expect no snapshots to exist, so this should return an error
		require.Error(t, err)
		require.IsType(t, &balance.NoSavedBalanceForOwnerError{}, err)

		// Trigger balance calculation which would normally create snapshots
		entBalance, err := conn.GetEntitlementBalance(ctx, owner, queryTime)
		require.NoError(t, err)

		// Verify the balance calculation works
		require.Equal(t, 800.0, entBalance.Balance)       // 1000 - 200 (latest value)
		require.Equal(t, 200.0, entBalance.UsageInPeriod) // latest value in current period

		// Verify that no snapshots were created after the calculation
		snapshotsAfter, err := deps.balanceSnapshotService.GetLatestValidAt(ctx, owner, queryTime)
		// Should still return error indicating no snapshots exist
		require.Error(t, err)
		require.IsType(t, &balance.NoSavedBalanceForOwnerError{}, err)

		// Ensure we didn't accidentally create any snapshots
		require.Equal(t, snapshotsBefore, snapshotsAfter)
	})
}
