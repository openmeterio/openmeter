package worker

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	rtemetrics "github.com/openmeterio/openmeter/openmeter/rte/metrics"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type comparisonDemand struct {
	rte.DemandService
	calls []rte.EnsureFreshInput
}

func (d *comparisonDemand) EnsureFresh(_ context.Context, input rte.EnsureFreshInput) error {
	time.Sleep(10 * time.Millisecond)
	d.calls = append(d.calls, input)

	return nil
}

type comparisonEngine struct {
	rte.Service
	gets, calculations int
	readError          error
}

func (s *comparisonEngine) MeteredEntitlements() rte.Calculation[meteredentitlement.Key, meteredentitlement.Balance] {
	return s
}

func (s *comparisonEngine) Get(context.Context, meteredentitlement.Key) (rte.State[meteredentitlement.Balance], error) {
	time.Sleep(2 * time.Millisecond)
	s.gets++

	return rte.State[meteredentitlement.Balance]{}, s.readError
}

func (s *comparisonEngine) Calculate(context.Context, meteredentitlement.Key) (rte.State[meteredentitlement.Balance], error) {
	time.Sleep(7 * time.Millisecond)
	s.calculations++

	return rte.State[meteredentitlement.Balance]{}, nil
}

func TestComparisonReplaysAndMeasuresMatchedOperations(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Given a read observed 5s ago, queue lag is separate from both durations.
		reader := sdkmetric.NewManualReader()
		provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		metrics, err := rtemetrics.NewComparison(provider.Meter("test"))
		require.NoError(t, err)

		target, err := rte.NewCalculationTarget(meteredentitlement.Key{EntitlementID: models.NamespacedID{Namespace: "ns", ID: "id"}})
		require.NoError(t, err)

		e := &rte.BalanceComparisonEvent{
			CalculationTarget: target,
			Operation:         rte.ComparisonOperationRead,
			DirectDuration:    1500 * time.Microsecond,
			ObservedAt:        time.Now().Add(-5 * time.Second),
		}
		demand, engine := &comparisonDemand{}, &comparisonEngine{}
		w := worker{service: engine, demand: demand, comparison: metrics, logger: testutils.NewDiscardLogger(t)}

		// When a production read and worker calculation are replayed.
		require.NoError(t, w.handleComparison(t.Context(), e))
		require.Equal(t, []rte.EnsureFreshInput{{Target: target, PullTTL: time.Minute, MaxAge: 30 * time.Second}}, demand.calls)
		e.Operation = rte.ComparisonOperationCalculate
		require.NoError(t, w.handleComparison(t.Context(), e))

		// Then a failed read and an expired observation neither retry nor enter either duration distribution.
		e.Operation = rte.ComparisonOperationRead
		engine.readError = errors.New("stored read failed")
		require.NoError(t, w.handleComparison(t.Context(), e))
		e.ObservedAt = time.Now().Add(-rte.ComparisonObservationTTL)
		require.NoError(t, w.handleComparison(t.Context(), e))

		require.Len(t, demand.calls, 2)
		require.Equal(t, 2, engine.gets)
		require.Equal(t, 1, engine.calculations)

		var data metricdata.ResourceMetrics
		require.NoError(t, reader.Collect(t.Context(), &data))

		durations := map[string]float64{}
		outcomes := map[string]int64{}
		for _, m := range data.ScopeMetrics[0].Metrics {
			switch m.Name {
			case "rte.comparison.duration":
				require.Equal(t, "ms", m.Unit)
				for _, point := range m.Data.(metricdata.Histogram[float64]).DataPoints {
					require.EqualValues(t, 1, point.Count)
					op, _ := point.Attributes.Value(attribute.Key("operation"))
					variant, _ := point.Attributes.Value(attribute.Key("variant"))
					durations[op.AsString()+"/"+variant.AsString()] = point.Sum
				}
			case "rte.comparison.executions":
				for _, point := range m.Data.(metricdata.Sum[int64]).DataPoints {
					outcome, _ := point.Attributes.Value(attribute.Key("outcome"))
					outcomes[outcome.AsString()] += point.Value
				}
			}
		}

		require.Equal(t, map[string]float64{"read/direct": 1.5, "read/rte": 12, "calculate/direct": 1.5, "calculate/rte": 7}, durations)
		require.Equal(t, map[string]int64{"success": 2, "expired": 1, "error": 1}, outcomes)
	})
}
