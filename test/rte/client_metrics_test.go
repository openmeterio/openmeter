package rte_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/adapter"
	"github.com/openmeterio/openmeter/openmeter/rte/client"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *RTEIntegrationSuite) TestClientReadMetrics() {
	t := s.T()

	// Given a database-backed client with its own telemetry provider, without an engine.
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { require.NoError(t, provider.Shutdown(t.Context())) })
	logger := testutils.NewDiscardLogger(t)
	cfg := client.Config{
		DB: s.DBClient, Publisher: eventbus.NewMock(t), Logger: logger,
		Meter: provider.Meter("rte-client-test"),
	}
	balanceClient, err := client.New(cfg)
	require.NoError(t, err)

	storeCfg := adapter.Config{Client: s.DBClient, Logger: logger}
	creditStore, err := adapter.NewCreditBalanceStore(storeCfg)
	require.NoError(t, err)
	meteredStore, err := adapter.NewMeteredEntitlementBalanceStore(storeCfg)
	require.NoError(t, err)
	ns := s.GetUniqueNamespace("rte-client-metrics")

	// When each balance kind is missing, stored as zero, and read with a cancelled context.
	exerciseClientReads(t, balanceClient.CreditBalances(), creditStore, creditbalance.Key{
		CustomerID: customer.CustomerID{Namespace: ns, ID: "customer"},
		Currency:   currencies.NewCurrencyReference("USD"),
	})
	exerciseClientReads(t, balanceClient.MeteredEntitlements(), meteredStore, meteredentitlement.Key{
		EntitlementID: models.NamespacedID{Namespace: ns, ID: "entitlement"},
	})

	// Then every database read contributes exactly once to both instruments.
	var collected metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &collected))
	counts := map[string]map[string]uint64{}
	for _, scope := range collected.ScopeMetrics {
		for _, instrument := range scope.Metrics {
			points := map[string]uint64{}
			switch data := instrument.Data.(type) {
			case metricdata.Sum[int64]:
				for _, point := range data.DataPoints {
					points[point.Attributes.Encoded(attribute.DefaultEncoder())] = uint64(point.Value)
				}
			case metricdata.Histogram[float64]:
				for _, point := range data.DataPoints {
					points[point.Attributes.Encoded(attribute.DefaultEncoder())] = point.Count
				}
			}
			counts[instrument.Name] = points
		}
	}
	require.Len(t, counts, 2, "client reads must not construct calculation instruments")
	require.Equal(t, counts["rte.state.reads"], counts["rte.state.read.duration"])
	require.Len(t, counts["rte.state.reads"], 6, "three outcomes for each balance kind")
	for _, count := range counts["rte.state.reads"] {
		require.Equal(t, uint64(1), count)
	}
}

// Exercise real storage so zero balances, missing state and storage failures retain
// their API semantics while telemetry observes each outcome.
func exerciseClientReads[K rte.Key, V any](t *testing.T, client rte.BalanceClient[K, V], store rte.Store[K, V], key K) {
	t.Helper()
	ctx := t.Context()

	_, err := client.Get(ctx, key)
	require.True(t, models.IsGenericNotFoundError(err), "%v", err)

	stored, err := store.Store(ctx, rte.StoreInput[K, V]{Key: key, State: rte.State[V]{CalculatedAt: time.Now().UTC().Truncate(time.Microsecond)}})
	require.NoError(t, err)
	result, err := client.Get(ctx, key)
	require.NoError(t, err)
	require.Equal(t, stored.State, result)

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	_, err = client.Get(cancelled, key)
	require.ErrorIs(t, err, context.Canceled)
}
