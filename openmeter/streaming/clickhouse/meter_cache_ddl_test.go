package clickhouse

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	progressmanager "github.com/openmeterio/openmeter/openmeter/progressmanager/adapter"
)

func TestCreateMeterCacheTableSQL(t *testing.T) {
	assert.Equal(t,
		"CREATE TABLE IF NOT EXISTS openmeter.om_meter_cache (namespace String, meter_key LowCardinality(String), meter_hash UInt64, windowstart DateTime, subject String, group_by Array(String), created_at DateTime64(3), sum_value Nullable(Decimal128(19)), count_value UInt64, value_count UInt64, min_value Nullable(Decimal128(19)), max_value Nullable(Decimal128(19)), uniq_state AggregateFunction(uniqExact, Nullable(String)), latest_state AggregateFunction(argMax, Nullable(Decimal128(19)), DateTime)) ENGINE = ReplacingMergeTree(created_at) ORDER BY (namespace, meter_key, meter_hash, windowstart, subject, group_by)",
		createMeterCacheTable{Database: "openmeter"}.toSQL(),
	)
}

func TestCreateMeterCacheInvalidationsTableSQL(t *testing.T) {
	assert.Equal(t,
		"CREATE TABLE IF NOT EXISTS openmeter.om_meter_cache_invalidations (namespace String, event_type LowCardinality(String), window_lo DateTime, window_hi DateTime, created_at DateTime64(3) DEFAULT now64(3)) ENGINE = MergeTree ORDER BY (namespace, event_type, window_lo) TTL toDateTime(created_at) + toIntervalDay(7)",
		createMeterCacheInvalidationsTable{Database: "openmeter"}.toSQL(),
	)
}

func TestConnectorCreatesMeterCacheTables(t *testing.T) {
	newConfig := func(mockCH *MockClickHouse) Config {
		return Config{
			Logger:          slog.Default(),
			ClickHouse:      mockCH,
			Database:        "testdb",
			EventsTableName: "events",
			ProgressManager: progressmanager.NewMockProgressManager(),
		}
	}

	execWithPrefix := func(prefix string) interface{} {
		return mock.MatchedBy(func(sql string) bool {
			return strings.HasPrefix(sql, prefix)
		})
	}

	enabledCache := CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	}

	t.Run("cache enabled provisions both cache tables", func(t *testing.T) {
		mockCH := NewMockClickHouse()
		mockCH.On("Exec", mock.Anything, execWithPrefix("CREATE TABLE IF NOT EXISTS testdb.events"), mock.Anything).Return(nil).Once()
		mockCH.On("Exec", mock.Anything, execWithPrefix("CREATE TABLE IF NOT EXISTS testdb.om_meter_cache ("), mock.Anything).Return(nil).Once()
		mockCH.On("Exec", mock.Anything, execWithPrefix("CREATE TABLE IF NOT EXISTS testdb.om_meter_cache_invalidations"), mock.Anything).Return(nil).Once()

		config := newConfig(mockCH)
		config.Cache = enabledCache

		_, err := New(t.Context(), config)
		require.NoError(t, err)

		mockCH.AssertExpectations(t)
		mockCH.AssertNumberOfCalls(t, "Exec", 3)
	})

	t.Run("cache disabled keeps a zero-footprint schema", func(t *testing.T) {
		mockCH := NewMockClickHouse()
		mockCH.On("Exec", mock.Anything, execWithPrefix("CREATE TABLE IF NOT EXISTS testdb.events"), mock.Anything).Return(nil).Once()

		_, err := New(t.Context(), newConfig(mockCH))
		require.NoError(t, err)

		mockCH.AssertExpectations(t)
		mockCH.AssertNumberOfCalls(t, "Exec", 1)
	})

	t.Run("SkipCreateTables wins over cache enabled", func(t *testing.T) {
		mockCH := NewMockClickHouse()

		config := newConfig(mockCH)
		config.SkipCreateTables = true
		config.Cache = enabledCache

		_, err := New(t.Context(), config)
		require.NoError(t, err)

		mockCH.AssertNumberOfCalls(t, "Exec", 0)
	})
}
