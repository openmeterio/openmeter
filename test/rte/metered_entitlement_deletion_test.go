package rte_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/rte"
	rteadapter "github.com/openmeterio/openmeter/openmeter/rte/adapter"
	rtemetered "github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	"github.com/openmeterio/openmeter/openmeter/rte/upstream"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *RTEIntegrationSuite) TestDeletedMeteredEntitlementDemand() {
	for _, tc := range []struct {
		name      string
		scheduled bool
		push      bool
	}{
		{name: "immediate push", push: true},
		{name: "scheduled push", scheduled: true, push: true},
		{name: "scheduled pull", scheduled: true},
	} {
		s.T().Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
			clock.FreezeTime(start)
			defer clock.UnFreeze()

			// Given a tracked entitlement with a successfully stored balance.
			ns := s.GetUniqueNamespace("rte-delete")
			cust := s.CreateTestCustomer(ns, "customer")
			feature := s.SetupApiRequestsTotalFeature(ctx, ns)
			defer feature.Cleanup()

			ent, err := s.Entitlements.Entitlement.CreateCustomerEntitlement(ctx, entitlement.CreateCustomerEntitlementInput{
				CustomerID: cust.GetID(),
				Entitlement: entitlement.CreateEntitlementInputs{
					FeatureID:       &feature.Feature.ID,
					EntitlementType: entitlement.EntitlementTypeMetered,
					IssueAfterReset: lo.ToPtr(float64(100)),
					UsagePeriod:     lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{Anchor: start, Interval: timeutil.RecurrencePeriodMonth})),
				},
			})
			require.NoError(t, err)

			logger := testutils.NewDiscardLogger(t)
			source, err := upstream.New(upstream.Config{Client: s.DBClient, Logger: logger})
			require.NoError(t, err)

			engine := s.newRTE()
			demand, err := common.NewRTEDemandService(s.DBClient, engine, logger, noop.NewMeterProvider().Meter("test"), config.RTEConfiguration{CalculationTimeout: time.Minute})
			require.NoError(t, err)

			key := rtemetered.Key{EntitlementID: models.NamespacedID{Namespace: ns, ID: ent.ID}}
			event, err := rte.NewCalculationRequestedEvent(key)
			require.NoError(t, err)

			pull := rte.EnsureFreshInput{Target: event.CalculationTarget, PullTTL: 2 * time.Hour}
			if tc.push {
				require.NoError(t, demand.Activate(ctx, pull.Target))
			}

			require.NoError(t, demand.EnsureFresh(ctx, pull))

			// When the deletion arrives, a future effective time still allows a refresh.
			now := start.Add(time.Minute)
			clock.FreezeTime(now)
			defer clock.UnFreeze()

			effective := now
			if tc.scheduled {
				effective = now.Add(time.Hour)
			}

			require.NoError(t, s.Entitlements.Entitlement.DeleteEntitlement(ctx, ns, ent.ID, effective))

			fact := entitlement.NewEntitlementDeletedEventPayloadV2(*ent, cust)
			selectors, err := source.Resolve(ctx, fact)
			require.NoError(t, err)

			invalid := rte.InvalidateInput{Source: fact.EventMetadata().Source, EventID: "deletion", Selectors: selectors}
			require.NoError(t, demand.Invalidate(ctx, invalid))
			if tc.scheduled {
				require.NoError(t, demand.EnsureFresh(ctx, pull))
				if tc.push {
					// Registering the key again preserves its deletion deadline.
					require.NoError(t, demand.Activate(ctx, pull.Target))
				}

				processed, err := demand.Process(ctx)
				require.NoError(t, err)
				require.False(t, processed)
			}

			last, err := engine.MeteredEntitlements().Get(ctx, key)
			require.NoError(t, err)
			if tc.scheduled {
				require.True(t, last.CalculatedAt.Equal(now))
			}

			if tc.scheduled && tc.push {
				// A transient calculator failure just before deletion must not
				// postpone retirement until the retry window expires.
				clock.FreezeTime(effective.Add(-time.Second))
				defer clock.UnFreeze()

				require.NoError(t, demand.Invalidate(ctx, rte.InvalidateInput{Source: invalid.Source, EventID: "new-fact", Selectors: selectors}))

				repo, err := rteadapter.NewDemandRepository(rteadapter.Config{Client: s.DBClient, Logger: logger})
				require.NoError(t, err)

				transient := errors.New("storage temporarily unavailable")
				work, err := repo.Process(ctx, rte.ProcessDemandInput{CalculationTimeout: time.Minute, RetryMinDelay: time.Minute, RetryMaxDelay: time.Minute},
					func(context.Context, rte.CalculationTarget) error { return transient })
				require.NoError(t, err)
				require.ErrorIs(t, work.CalculationError, transient)
				processed, err := demand.Process(ctx)
				require.NoError(t, err)
				require.False(t, processed)
			}

			// Then the deadline alone retires push work. Explicit pulls see the
			// missing source and stored reads keep returning the last successful value.
			clock.FreezeTime(effective)
			defer clock.UnFreeze()

			processed, err := demand.Process(ctx)
			require.NoError(t, err)
			require.Equal(t, tc.push, processed)
			require.NoError(t, demand.Invalidate(ctx, invalid))

			for range 2 {
				var missing *entitlement.NotFoundError
				require.ErrorAs(t, demand.EnsureFresh(ctx, pull), &missing)
				require.Equal(t, key.EntitlementID, missing.EntitlementID)
			}

			clock.FreezeTime(effective.Add(24 * time.Hour))
			defer clock.UnFreeze()

			processed, err = demand.Process(ctx)
			require.NoError(t, err)
			require.False(t, processed)
			stored, err := engine.MeteredEntitlements().Get(ctx, key)
			require.NoError(t, err)
			require.Equal(t, last, stored)
		})
	}
}
