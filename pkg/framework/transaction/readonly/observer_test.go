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
	first := observer.MarkReadOnly(t.Context(), "get-entitlement-value")
	second := observer.MarkReadOnly(t.Context(), "get-entitlement-value")
	observer.MarkReadOnly(t.Context(), "list-customers")

	snapshotCreate := transaction.WriteAttempt{Kind: transaction.WriteKindMutation, Target: "BalanceSnapshot", Operation: "OpCreate"}

	// when the same write happens in both requests, and each locks a different customer
	require.NoError(t, transaction.ReportWrite(first, snapshotCreate), "observing never fails the request")
	require.NoError(t, transaction.ReportWrite(second, snapshotCreate))
	require.NoError(t, transaction.ReportWrite(first, transaction.WriteAttempt{Kind: transaction.WriteKindAdvisoryLock, Target: "customer:01A"}))
	require.NoError(t, transaction.ReportWrite(second, transaction.WriteAttempt{Kind: transaction.WriteKindAdvisoryLock, Target: "customer:01B"}))

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
		Message     string `json:"msg"`
		Operation   string `json:"operation"`
		WriteKind   string `json:"write_kind"`
		WriteTarget string `json:"write_target"`
		Caller      string `json:"caller"`
	}

	var logged []logLine
	for _, line := range bytes.Split(bytes.TrimSpace(logs.Bytes()), []byte("\n")) {
		var entry logLine
		require.NoError(t, json.Unmarshal(line, &entry))
		require.NotEmpty(t, entry.Caller)

		entry.Caller = ""
		logged = append(logged, entry)
	}

	require.Equal(t, []logLine{
		{Message: "read-only request wrote to the database", Operation: "get-entitlement-value", WriteKind: "mutation", WriteTarget: "BalanceSnapshot"},
		{Message: "read-only request wrote to the database", Operation: "get-entitlement-value", WriteKind: "advisory_lock", WriteTarget: "customer:01A"},
	}, logged)
}

func TestObserverConfigValidation(t *testing.T) {
	_, err := readonly.NewObserver(readonly.ObserverConfig{})
	require.ErrorContains(t, err, "logger is required")
	require.ErrorContains(t, err, "meter is required")
}
