package rte_test

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/unitconfig"
	"github.com/openmeterio/openmeter/openmeter/rte"
	rtemetered "github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *RTEIntegrationSuite) TestMeteredUsageCheckpoint() {
	for _, aggregation := range []meter.MeterAggregation{meter.MeterAggregationCount, meter.MeterAggregationSum} {
		s.Run(string(aggregation), func() { s.checkMeteredUsageCheckpoint(aggregation) })
	}
}

func (s *RTEIntegrationSuite) checkMeteredUsageCheckpoint(aggregation meter.MeterAggregation) {
	t := s.T()
	ctx := t.Context()
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// Given a metered entitlement with grants activating in four different minutes.
	ns := s.GetUniqueNamespace("rte-count")
	customer := s.CreateTestCustomer(ns, "rte-count-customer")
	m := meter.Meter{
		ManagedResource: models.ManagedResource{
			ID: ulid.Make().String(), Name: "Requests", NamespacedModel: models.NamespacedModel{Namespace: ns},
			ManagedModel: models.ManagedModel{CreatedAt: start, UpdatedAt: start},
		},
		Key: "requests", EventType: "request", Aggregation: aggregation,
	}
	var conversion *unitconfig.UnitConfig
	if aggregation == meter.MeterAggregationSum {
		m.ValueProperty = lo.ToPtr("$.value")
		conversion = &unitconfig.UnitConfig{
			Operation: unitconfig.UnitConfigOperationMultiply, ConversionFactor: alpacadecimal.NewFromInt(2),
		}
	}

	s.Require().NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{m}))
	s.MockStreamingConnector.SetSimpleEvents(m.Key, func(_ []streamingtestutils.SimpleEvent) []streamingtestutils.SimpleEvent { return nil })
	t.Cleanup(s.MockStreamingConnector.Reset)

	f, err := s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{Namespace: ns, Name: m.Name, Key: m.Key, MeterID: &m.ID})
	s.Require().NoError(err)
	ent, err := s.Entitlements.Entitlement.CreateCustomerEntitlement(ctx, entitlement.CreateCustomerEntitlementInput{
		CustomerID: customer.GetID(),
		Entitlement: entitlement.CreateEntitlementInputs{
			FeatureID: &f.ID, EntitlementType: entitlement.EntitlementTypeMetered, IssueAfterReset: lo.ToPtr(100.0),
			UnitConfig: conversion,
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Anchor: start, Interval: timeutil.RecurrencePeriodMonth,
			})),
		},
	})
	s.Require().NoError(err)
	key := rtemetered.Key{EntitlementID: models.NamespacedID{Namespace: ns, ID: ent.ID}}
	for minute := 1; minute <= 3; minute++ {
		_, err := s.Entitlements.Grant.CreateGrant(ctx, key.EntitlementID, credit.CreateGrantInput{
			Amount: 10, EffectiveAt: start.Add(time.Duration(minute) * time.Minute),
		})
		s.Require().NoError(err)
	}

	target, err := rte.NewCalculationTarget(key)
	s.Require().NoError(err)
	refresh := func(at time.Time) (rtemetered.Balance, int) {
		t.Helper()
		clock.FreezeTime(at)
		defer clock.UnFreeze()

		// Each worker starts with only PostgreSQL state from the last refresh.
		worker := s.newRTE()
		before := len(s.MockStreamingConnector.MeterQueries())
		s.Require().NoError(worker.RefreshDemand(ctx, rte.RefreshDemandInput{Target: target, UseCheckpoints: true}))
		state, err := worker.MeteredEntitlements().Get(ctx, key)
		s.Require().NoError(err)

		return state.Value, len(s.MockStreamingConnector.MeterQueries()) - before
	}

	// When a later phase has one event, a single source probe covers all four
	// earlier empty phases. The last partial minute is still queried for usage.
	seed, seedQueries := refresh(start.Add(5 * time.Minute))
	s.Require().Equal(4, seedQueries)
	s.Require().Equal(130.0, seed.Balance)
	s.MockStreamingConnector.AddSimpleEvent(m.Key, 1, start.Add(5*time.Minute))

	now := start.Add(5*time.Minute + 30*time.Second)
	warm, warmQueries := refresh(now)
	expected, err := s.Entitlements.MeteredEntitlement.GetEntitlementBalance(credit.WithoutBalanceSnapshotWrites(ctx), key.EntitlementID, now)
	s.Require().NoError(err)
	s.Require().Equal(*expected, warm)
	if conversion == nil {
		s.Require().Equal(1.0, warm.UsageInPeriod)
	} else {
		s.Require().Equal(2.0, warm.UsageInPeriod)
	}

	s.Require().Equal(2, warmQueries)

	// Then a backdated event with no invalidation makes the probe fail. The
	// original calculation assigns the usage to its correct grants and period.
	s.MockStreamingConnector.AddSimpleEvent(m.Key, 1, start)
	if aggregation == meter.MeterAggregationSum {
		// These changes sum to zero over the old prefix but hit different
		// grants. A SUM probe would incorrectly certify that prefix as empty.
		s.MockStreamingConnector.AddSimpleEvent(m.Key, -2, start.Add(time.Minute))
		s.MockStreamingConnector.AddSimpleEvent(m.Key, 1, start.Add(2*time.Minute))
	} else {
		s.MockStreamingConnector.AddSimpleEvent(m.Key, 1, start.Add(time.Minute))
		s.MockStreamingConnector.AddSimpleEvent(m.Key, 1, start.Add(2*time.Minute))
	}

	now = now.Add(time.Minute)
	late, lateQueries := refresh(now)
	expected, err = s.Entitlements.MeteredEntitlement.GetEntitlementBalance(credit.WithoutBalanceSnapshotWrites(ctx), key.EntitlementID, now)
	s.Require().NoError(err)
	s.Require().Equal(*expected, late)
	if conversion == nil {
		s.Require().Equal(4.0, late.UsageInPeriod)
	} else {
		s.Require().Equal(2.0, late.UsageInPeriod)
	}

	s.Require().NotEqual(warm.GrantBalances, late.GrantBalances)
	s.Require().Equal(5, lateQueries)

	// Disproved ranges were removed, and all four phases have usage now.
	dense, denseQueries := refresh(now.Add(time.Minute))
	s.Require().Equal(late, dense)
	s.Require().Equal(4, denseQueries)
}
