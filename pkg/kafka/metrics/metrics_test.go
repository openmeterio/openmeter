package metrics

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/openmeterio/openmeter/pkg/kafka/metrics/stats"
)

//go:embed stats/testdata/stats.json
var statsJSON []byte

func NewTestStats(t *testing.T) *stats.Stats {
	t.Helper()

	var s stats.Stats

	err := json.Unmarshal(statsJSON, &s)
	if err != nil {
		t.Errorf("failed to unmarshal stats JSON: %v", err)
	}

	return &s
}

func TestWithMetrics(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
	}{
		{
			name:    "default",
			options: []Option{},
		},
		{
			name: "extended metrics",
			options: []Option{
				WithExtendedMetrics(),
			},
		},
		{
			name: "disabled broker metrics",
			options: []Option{
				WithBrokerMetricsDisabled(),
			},
		},
		{
			name: "disabled topic metrics",
			options: []Option{
				WithTopicMetricsDisabled(),
			},
		},
		{
			name: "disabled consumer group metrics",
			options: []Option{
				WithConsumerGroupMetricsDisabled(),
			},
		},
	}

	testStats := NewTestStats(t)

	for _, test := range tests {
		testMeter := noop.NewMeterProvider().Meter("test")

		t.Run(test.name, func(t *testing.T) {
			kafkaMetrics, err := New(testMeter, test.options...)
			require.NoErrorf(t, err, "creating kafka metrics should not fail")

			kafkaMetrics.Add(t.Context(), testStats)
		})
	}
}
