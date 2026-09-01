package ingest_test

import (
	"log/slog"
	"testing"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/openmeterio/openmeter/openmeter/ingest"
)

func TestIngestEventsRecordsRequestEventCount(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		require.NoError(t, provider.Shutdown(t.Context()))
	})

	service, err := ingest.NewService(ingest.Config{
		Collector:   ingest.NewInMemoryCollector(),
		Logger:      slog.Default(),
		MetricMeter: provider.Meter("ingest-test"),
	})
	require.NoError(t, err)

	_, err = service.IngestEvents(t.Context(), ingest.IngestEventsRequest{
		Namespace: "default",
		Events:    []event.Event{event.New()},
	})
	require.NoError(t, err)

	_, err = service.IngestEvents(t.Context(), ingest.IngestEventsRequest{
		Namespace: "default",
		Events:    []event.Event{event.New(), event.New(), event.New()},
	})
	require.NoError(t, err)

	var metrics metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &metrics))

	var histogram metricdata.Histogram[int64]
	found := false
	for _, scopeMetrics := range metrics.ScopeMetrics {
		for _, metric := range scopeMetrics.Metrics {
			if metric.Name == "openmeter.ingest.request.events" {
				var ok bool
				histogram, ok = metric.Data.(metricdata.Histogram[int64])
				require.True(t, ok)
				found = true
			}
		}
	}
	require.True(t, found)

	require.Len(t, histogram.DataPoints, 1)
	require.Equal(t, uint64(2), histogram.DataPoints[0].Count)
	require.Equal(t, int64(4), histogram.DataPoints[0].Sum)
	namespace, ok := histogram.DataPoints[0].Attributes.Value(attribute.Key("namespace"))
	require.True(t, ok)
	require.Equal(t, "default", namespace.AsString())

	require.Equal(
		t,
		[]float64{0, 1, 2, 3, 4, 5, 10, 25, 100, 1000, 10000},
		histogram.DataPoints[0].Bounds,
	)

	// The one-event and three-event requests must land in separate buckets. Under
	// the SDK default boundaries both fall into (0, 5], which is what collapses
	// the distribution.
	require.Equal(t, uint64(1), histogram.DataPoints[0].BucketCounts[1], "(0, 1] should hold the single-event request")
	require.Equal(t, uint64(1), histogram.DataPoints[0].BucketCounts[3], "(2, 3] should hold the three-event request")
}
