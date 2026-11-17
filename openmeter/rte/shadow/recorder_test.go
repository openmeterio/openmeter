package shadow

import (
	"context"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestRecorderRetainsKeyCohortsAcrossReadsAndCalculations(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Given two producers using the same sampling percentage.
		observed := [2]map[string]rte.ComparisonOperation{{}, {}}
		var recorders [2]*Recorder
		for i := range recorders {
			var err error
			recorders[i], err = NewRecorder(t.Context(), Config{
				SamplePercent: 25, Meter: noop.NewMeterProvider().Meter("test"), Logger: testutils.NewDiscardLogger(t),
				Publish: func(_ context.Context, event marshaler.Event) error {
					e := event.(rte.BalanceComparisonEvent)
					observed[i][string(e.Key)] = e.Operation

					return nil
				},
			})
			require.NoError(t, err)
			defer recorders[i].Close()
		}

		// When one producer receives reads and another sees calculations.
		for i := range 96 {
			id := models.NamespacedID{Namespace: "ns", ID: fmt.Sprint(i)}
			recorders[0].ObserveMeteredRead(t.Context(), id, time.Millisecond)
			recorders[1].ObserveMeteredCalculation(t.Context(), id, time.Millisecond)
		}

		synctest.Wait()

		// Then sampled keys retain both operations, preserving their workload mix.
		require.NotEmpty(t, observed[0])
		require.Less(t, len(observed[0]), 96)
		require.Len(t, observed[1], len(observed[0]))
		for key, op := range observed[0] {
			require.Equal(t, rte.ComparisonOperationRead, op)
			require.Equal(t, rte.ComparisonOperationCalculate, observed[1][key])
		}
	})
}

func TestRecorderBoundsBlockedPublicationAndReportsLoss(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		type requestKey struct{}
		ctx, cancelRequest := context.WithCancel(context.WithValue(t.Context(), requestKey{}, true))
		reader := sdkmetric.NewManualReader()
		provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		entered := make(chan context.Context)

		// Given a stuck publisher and a request that has already completed.
		recorder, err := NewRecorder(t.Context(), Config{
			SamplePercent: 100, Meter: provider.Meter("test"), Logger: testutils.NewDiscardLogger(t),
			Publish: func(ctx context.Context, _ marshaler.Event) error {
				entered <- ctx
				<-ctx.Done()

				return ctx.Err()
			},
		})
		require.NoError(t, err)
		id := models.NamespacedID{Namespace: "ns", ID: "id"}
		recorder.ObserveMeteredRead(ctx, id, time.Millisecond)
		cancelRequest()
		publishCtx := <-entered
		require.Nil(t, publishCtx.Value(requestKey{}))
		require.NoError(t, publishCtx.Err())

		// When more requests arrive than can fit, each producer still returns.
		for range queueCapacity + 1 {
			recorder.ObserveMeteredRead(ctx, id, time.Millisecond)
		}

		recorder.Close()
		recorder.ObserveMeteredRead(ctx, id, time.Millisecond)

		// Then in-flight, overflowed and abandoned observations are all accounted for.
		var metrics metricdata.ResourceMetrics
		require.NoError(t, reader.Collect(t.Context(), &metrics))
		counts := map[string]int64{}
		for _, point := range metrics.ScopeMetrics[0].Metrics[0].Data.(metricdata.Sum[int64]).DataPoints {
			outcome, ok := point.Attributes.Value(attribute.Key("outcome"))
			require.True(t, ok)
			counts[outcome.AsString()] = point.Value
		}

		require.Equal(t, map[string]int64{"queue_full": 1, "failed": 1, "stopped": queueCapacity + 1}, counts)
	})
}
