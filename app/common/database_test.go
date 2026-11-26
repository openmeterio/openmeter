package common

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/testutils"
)

func TestPostgresDriverSpans(t *testing.T) {
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEmpty)
	t.Cleanup(func() { testDB.Close(t) })

	recorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	meterProvider := metricnoop.NewMeterProvider()

	driver, closeDriver, err := NewPostgresDriver(
		t.Context(),
		config.PostgresConfig{URL: testDB.URL},
		meterProvider,
		meterProvider.Meter("test"),
		tracerProvider,
		slog.New(slog.DiscardHandler),
	)
	require.NoError(t, err)
	t.Cleanup(closeDriver)

	db := driver.DB()

	_, err = db.ExecContext(t.Context(), "SELECT 1")
	require.NoError(t, err)
	require.Empty(t, recorder.Ended(), "a statement outside a span must not emit a span")

	ctx, parent := tracerProvider.Tracer("test").Start(t.Context(), "parent")
	_, err = db.ExecContext(ctx, "SELECT 1")
	require.NoError(t, err)
	parent.End()

	execSpans := 0

	for _, span := range recorder.Ended() {
		if span.SpanContext().SpanID() == parent.SpanContext().SpanID() {
			continue
		}

		require.Equal(t, parent.SpanContext().SpanID(), span.Parent().SpanID(), "span %q must be a child of the caller's span", span.Name())

		if span.Name() == "sql.conn.exec" {
			execSpans++
		}
	}

	require.Equal(t, 1, execSpans, "one statement must emit exactly one exec span")
}
