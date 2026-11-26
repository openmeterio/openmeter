package rte_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	creditstest "github.com/openmeterio/openmeter/test/credits"
)

func (s *RTEIntegrationSuite) TestIncrementalCreditBalanceAcrossReplicas() {
	t := s.T()
	ctx := t.Context()
	at := time.Date(2026, time.January, 2, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(at)
	defer clock.UnFreeze()

	// Given a funded customer and replicas that share PostgreSQL and publish through the outbox.
	pipeline := s.newEventPipeline()
	ns := s.GetUniqueNamespace("rte-checkpoint")
	customer := s.CreateLedgerBackedCustomer(ns, "funded")
	key := creditbalance.Key{CustomerID: customer.GetID(), Currency: currencies.NewCurrencyReference(creditstest.USD)}
	s.postFundedCredits(ctx, key, at.Add(-time.Minute), 100)

	target, err := rte.NewCalculationTarget(key)
	require.NoError(t, err)
	pull := rte.EnsureFreshInput{Target: target, PullTTL: time.Hour, MaxAge: 30 * time.Second}

	metrics := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(metrics))
	t.Cleanup(func() { require.NoError(t, provider.Shutdown(ctx)) })
	meter := provider.Meter("rte-checkpoint-test")

	checkBalance := func(replica creditTestReplica, wantLive float64) {
		t.Helper()
		require.NoError(t, replica.demand.EnsureFresh(ctx, pull))
		stored, err := replica.engine.CreditBalances().Get(ctx, key)
		require.NoError(t, err)

		full, err := s.CustomerBalanceSvc.GetBalance(ctx, customerbalance.GetBalanceServiceInput{CustomerID: key.CustomerID, Currency: key.Currency})
		require.NoError(t, err)
		require.Equal(t, wantLive, stored.Value.Live.InexactFloat64())
		require.True(t, full.Live().Equal(stored.Value.Live))
		require.True(t, full.Booked().Equal(stored.Value.Booked))
		require.True(t, full.Spendable().Equal(stored.Value.Spendable))
		require.True(t, full.Pending().Equal(stored.Value.Pending))

		// The event carries exactly the public stored state. Checkpoint payloads stay in PostgreSQL.
		update := pipeline.awaitUpdate(t, target)
		publicState, err := json.Marshal(stored)
		require.NoError(t, err)
		require.JSONEq(t, string(publicState), string(update.State))
	}

	first := s.newCreditTestReplica(pipeline.publisher, meter, true)
	checkBalance(first, 100)
	require.Equal(t, map[string]int64{"checkpoint_miss": 1}, creditCalculationMethods(t, metrics))

	// When a second instance refreshes the same key, it resumes from the first instance's checkpoint.
	clock.FreezeTime(at.Add(time.Minute))
	defer clock.UnFreeze()
	second := s.newCreditTestReplica(pipeline.publisher, meter, true)
	checkBalance(second, 100)
	require.Equal(t, map[string]int64{"checkpoint_miss": 1, "checkpoint_hit": 1}, creditCalculationMethods(t, metrics))

	// Then even a backdated posting without an RTE invalidation is visible on the next age-based refresh.
	s.postFundedCredits(ctx, key, at.Add(-2*time.Minute), 25)
	clock.FreezeTime(at.Add(2 * time.Minute))
	defer clock.UnFreeze()
	checkBalance(second, 125)
	require.Equal(t, map[string]int64{"checkpoint_miss": 2, "checkpoint_hit": 1}, creditCalculationMethods(t, metrics))

	// A later funding can advance the 12:02 checkpoint to 12:03 through the new postings.
	s.postFundedCredits(ctx, key, at.Add(150*time.Second), 15)
	clock.FreezeTime(at.Add(3 * time.Minute))
	defer clock.UnFreeze()
	checkBalance(second, 140)

	// This 12:02:45 posting invalidates the 12:03 checkpoint. The 12:02 opening still covers its prefix.
	s.postFundedCredits(ctx, key, at.Add(165*time.Second), 10)
	clock.FreezeTime(at.Add(4 * time.Minute))
	defer clock.UnFreeze()
	checkBalance(first, 150)
	require.Equal(t, map[string]int64{"checkpoint_miss": 2, "checkpoint_hit": 1, "checkpoint_advanced": 2}, creditCalculationMethods(t, metrics))
}

type creditTestReplica struct {
	engine rte.Service
	demand rte.DemandService
}

func (s *RTEIntegrationSuite) newCreditTestReplica(publisher eventbus.Publisher, meter metric.Meter, useCheckpoints bool) creditTestReplica {
	t := s.T()
	t.Helper()
	logger := testutils.NewDiscardLogger(t)
	engine, err := common.NewRTEService(s.DBClient, s.CustomerBalanceSvc, s.Entitlements, logger,
		config.CreditsConfiguration{Enabled: true}, publisher, meter)
	require.NoError(t, err)

	demand, err := common.NewRTEDemandService(s.DBClient, engine, logger, meter, config.RTEConfiguration{
		IncrementalCalculations: useCheckpoints,
		CalculationTimeout:      30 * time.Second,
	})
	require.NoError(t, err)

	return creditTestReplica{engine: engine, demand: demand}
}

func creditCalculationMethods(t *testing.T, reader *sdkmetric.ManualReader) map[string]int64 {
	t.Helper()
	var collected metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &collected))

	counts := make(map[string]int64)
	for _, scope := range collected.ScopeMetrics {
		for _, instrument := range scope.Metrics {
			if instrument.Name != "rte.calculations" {
				continue
			}

			for _, point := range instrument.Data.(metricdata.Sum[int64]).DataPoints {
				kind, _ := point.Attributes.Value(attribute.Key("kind"))
				if kind.AsString() == string(rte.CalculationKindCreditBalance) {
					method, ok := point.Attributes.Value(attribute.Key("method"))
					require.True(t, ok)
					counts[method.AsString()] += point.Value
				}
			}
		}
	}

	return counts
}
