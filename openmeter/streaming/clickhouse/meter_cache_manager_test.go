package clickhouse

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
)

// managerFixtureConnector builds a *Connector directly (bypassing New, which requires a
// live ClickHouse connection) for exercising the CH-free branches of the cache manager
// surface: config field reads and SQL/metadata generation, no round-trips.
func managerFixtureConnector(cache CacheConfig) *Connector {
	return &Connector{
		config: Config{
			Database:        "openmeter",
			EventsTableName: "om_events",
			Cache:           cache,
		},
	}
}

func TestDesiredMeterCacheViewRejectsLatest(t *testing.T) {
	c := managerFixtureConnector(CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	})

	// LATEST is excluded from the cache entirely (meterCacheStaticReject): the reconciler's
	// desired-state map must never resolve a name/hash for it, which is what keeps a LATEST
	// meter out of both MV creation and the orphan-row GC keep set.
	_, err := c.DesiredMeterCacheView("my_namespace", meter.Meter{
		Key:           "meter1",
		EventType:     "event1",
		Aggregation:   meter.MeterAggregationLatest,
		ValueProperty: lo.ToPtr("$.value"),
	})
	require.ErrorContains(t, err, "LATEST")

	// A non-LATEST meter on the same connector still resolves normally, proving the
	// rejection is aggregation-specific and not a fixture-wiring accident.
	_, err = c.DesiredMeterCacheView("my_namespace", meter.Meter{
		Key:           "meter2",
		EventType:     "event1",
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value"),
	})
	require.NoError(t, err)
}
