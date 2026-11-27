package readonly_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/framework/transaction/readonly"
)

func TestObserver(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { require.NoError(t, provider.Shutdown(t.Context())) })

	var logs bytes.Buffer

	observer, err := readonly.NewObserver(readonly.ObserverConfig{
		Logger: slog.New(slog.NewJSONHandler(&logs, nil)),
		Meter:  provider.Meter("read-only-test"),
	})
	require.NoError(t, err)

	// given two read-only requests of one operation and one of another that never writes
	first, firstRejectedWrite := observer.MarkReadOnly(t.Context(), "get-entitlement-value")
	second, _ := observer.MarkReadOnly(t.Context(), "get-entitlement-value")
	observer.MarkReadOnly(t.Context(), "list-customers")

	snapshotCreate := transaction.WriteAttempt{Kind: transaction.WriteKindMutation, Target: "BalanceSnapshot", Operation: "OpCreate"}

	// when the same write happens in both requests, and each locks a different customer
	require.NoError(t, transaction.ReportWrite(first, snapshotCreate), "observing never fails the request")
	require.NoError(t, transaction.ReportWrite(second, snapshotCreate))
	require.NoError(t, transaction.ReportWrite(first, transaction.WriteAttempt{Kind: transaction.WriteKindAdvisoryLock, Target: "customer:01A"}))
	require.NoError(t, transaction.ReportWrite(second, transaction.WriteAttempt{Kind: transaction.WriteKindAdvisoryLock, Target: "customer:01B"}))
	require.NoError(t, firstRejectedWrite(), "an observed write is not a rejected one")

	// then every request and write is counted, with lock keys kept out of the labels
	var metrics metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &metrics))

	counts := map[string]map[attribute.Set]int64{}
	for _, scope := range metrics.ScopeMetrics {
		for _, m := range scope.Metrics {
			sum, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok, m.Name)

			counts[m.Name] = map[attribute.Set]int64{}
			for _, point := range sum.DataPoints {
				counts[m.Name][point.Attributes] = point.Value
			}
		}
	}

	require.Equal(t, map[string]map[attribute.Set]int64{
		"openmeter.database.read_only.requests": {
			attribute.NewSet(attribute.String("operation", "get-entitlement-value")): 2,
			attribute.NewSet(attribute.String("operation", "list-customers")):        1,
		},
		"openmeter.database.read_only.writes": {
			attribute.NewSet(
				attribute.String("operation", "get-entitlement-value"),
				attribute.String("write.kind", "mutation"),
				attribute.String("write.target", "BalanceSnapshot"),
				attribute.String("write.operation", "OpCreate"),
			): 2,
			attribute.NewSet(
				attribute.String("operation", "get-entitlement-value"),
				attribute.String("write.kind", "advisory_lock"),
				attribute.String("write.target", ""),
				attribute.String("write.operation", ""),
			): 2,
		},
	}, counts)

	// and each distinct write is logged once, however often it repeats
	type logLine struct {
		Message     string   `json:"msg"`
		Operation   string   `json:"operation"`
		WriteKind   string   `json:"write_kind"`
		WriteTarget string   `json:"write_target"`
		Stack       []string `json:"stack"`
	}

	var logged []logLine
	for _, line := range bytes.Split(bytes.TrimSpace(logs.Bytes()), []byte("\n")) {
		var entry logLine
		require.NoError(t, json.Unmarshal(line, &entry))

		// the stack starts at the code that reported the write, which is this test
		require.NotEmpty(t, entry.Stack)
		require.Contains(t, entry.Stack[0], "readonly_test.TestObserver:")

		entry.Stack = nil
		logged = append(logged, entry)
	}

	require.Equal(t, []logLine{
		{Message: "read-only request attempted a database write", Operation: "get-entitlement-value", WriteKind: "mutation", WriteTarget: "BalanceSnapshot"},
		{Message: "read-only request attempted a database write", Operation: "get-entitlement-value", WriteKind: "advisory_lock", WriteTarget: "customer:01A"},
	}, logged)
}

func TestObserverRejectsWrites(t *testing.T) {
	provider := sdkmetric.NewMeterProvider()
	t.Cleanup(func() { require.NoError(t, provider.Shutdown(t.Context())) })

	var logs bytes.Buffer

	newObserver := func(rejectWrites bool) *readonly.Observer {
		observer, err := readonly.NewObserver(readonly.ObserverConfig{
			Logger:       slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelError})),
			Meter:        provider.Meter("read-only-test"),
			RejectWrites: rejectWrites,
		})
		require.NoError(t, err)

		return observer
	}

	attempt := transaction.WriteAttempt{Kind: transaction.WriteKindMutation, Target: "BalanceSnapshot", Operation: "OpCreate"}

	// given a request marked by an observing observer and one marked by a rejecting observer
	observed, _ := newObserver(false).MarkReadOnly(t.Context(), "get-entitlement-value")
	rejected, rejectedWrite := newObserver(true).MarkReadOnly(t.Context(), "get-entitlement-value")
	require.NoError(t, rejectedWrite(), "nothing is rejected before the request writes")

	// when both attempt a write, and the rejected one ignores the error and locks a row next
	observedErr := transaction.ReportWrite(observed, attempt)
	rejectedErr := transaction.ReportWrite(rejected, attempt)
	require.Error(t, transaction.ReportWrite(rejected, transaction.WriteAttempt{Kind: transaction.WriteKindRowLock, Target: "Entitlement"}))

	// then only the rejecting one fails it, and says which request wrote what
	require.NoError(t, observedErr)
	require.False(t, transaction.RejectsWrites(observed))

	require.ErrorIs(t, rejectedErr, readonly.ErrWriteRejected)
	require.ErrorContains(t, rejectedErr, `mutation by "get-entitlement-value" (target "BalanceSnapshot", operation "OpCreate")`)
	require.True(t, transaction.RejectsWrites(rejected), "transactions of the request are opened READ ONLY")

	// and the request can still be failed with its first rejected write, however it handled the error
	require.Equal(t, rejectedErr, rejectedWrite())

	// and both rejected writes are logged as errors
	require.Equal(t, 2, bytes.Count(logs.Bytes(), []byte(`"level":"ERROR","msg":"read-only request attempted a database write"`)), logs.String())
}

func TestObserverConfigValidation(t *testing.T) {
	_, err := readonly.NewObserver(readonly.ObserverConfig{})
	require.ErrorContains(t, err, "logger is required")
	require.ErrorContains(t, err, "meter is required")
}
