package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAggregationCacheConfigurationValidate(t *testing.T) {
	validBase := func() AggregationCacheConfiguration {
		return AggregationCacheConfiguration{
			Enabled:             false,
			RefreshInterval:     10 * time.Minute,
			MinimumUsageAge:     time.Hour,
			WindowSize:          AggregationCacheGrainHour,
			MeterQueryThreshold: 0,
		}
	}

	t.Run("Valid", func(t *testing.T) {
		require.NoError(t, validBase().Validate())
	})

	t.Run("MinimumUsageAgeBelowRefreshInterval", func(t *testing.T) {
		cfg := validBase()
		cfg.MinimumUsageAge = 5 * time.Minute
		err := cfg.Validate()
		assert.ErrorContains(t, err, "minimum usage age must be greater than or equal to refresh interval")
	})

	t.Run("MinimumUsageAgeEqualToRefreshInterval", func(t *testing.T) {
		// Equality is the documented boundary (>=), not a rejection case.
		cfg := validBase()
		cfg.MinimumUsageAge = cfg.RefreshInterval
		require.NoError(t, cfg.Validate())
	})

	t.Run("InvalidGrainMonth", func(t *testing.T) {
		cfg := validBase()
		cfg.WindowSize = "month"
		err := cfg.Validate()
		assert.ErrorContains(t, err, "window size must be one of")
	})

	t.Run("InvalidGrainEmpty", func(t *testing.T) {
		cfg := validBase()
		cfg.WindowSize = ""
		err := cfg.Validate()
		assert.ErrorContains(t, err, "window size must be one of")
	})

	t.Run("InvalidGrainCasingMismatchesMeterWindowSize", func(t *testing.T) {
		// AggregationCacheGrain is intentionally lowercase and does not accept
		// meter.WindowSize's uppercase spelling (e.g. "HOUR").
		cfg := validBase()
		cfg.WindowSize = "HOUR"
		err := cfg.Validate()
		assert.ErrorContains(t, err, "window size must be one of")
	})

	t.Run("PositiveMeterQueryThresholdReserved", func(t *testing.T) {
		cfg := validBase()
		cfg.MeterQueryThreshold = 1
		err := cfg.Validate()
		assert.ErrorContains(t, err, "hot-meter selection not implemented")
	})

	t.Run("NegativeMeterQueryThreshold", func(t *testing.T) {
		cfg := validBase()
		cfg.MeterQueryThreshold = -1
		err := cfg.Validate()
		assert.ErrorContains(t, err, "meter query threshold must not be negative")
	})
}

func TestAggregationConfigurationValidateCache(t *testing.T) {
	validBase := func() AggregationConfiguration {
		return AggregationConfiguration{
			ClickHouse: ClickHouseAggregationConfiguration{
				Address:         "127.0.0.1:9000",
				DialTimeout:     10 * time.Second,
				MaxOpenConns:    5,
				MaxIdleConns:    5,
				ConnMaxLifetime: 10 * time.Minute,
				BlockBufferSize: 10,
			},
			EventsTableName: "om_events",
			Cache: AggregationCacheConfiguration{
				RefreshInterval: 10 * time.Minute,
				MinimumUsageAge: time.Hour,
				WindowSize:      AggregationCacheGrainHour,
			},
		}
	}

	t.Run("DisabledCacheIgnoresDecimalPrecisionAndAsyncInsertRules", func(t *testing.T) {
		// The cache-specific cross-field rules must only fire when the cache is enabled,
		// so every existing (cache-disabled) deployment config keeps validating exactly
		// as before this configuration was introduced.
		cfg := validBase()
		cfg.EnableDecimalPrecision = false
		cfg.AsyncInsert = true
		cfg.AsyncInsertWait = false
		require.NoError(t, cfg.Validate())
	})

	t.Run("EnabledCacheRequiresDecimalPrecision", func(t *testing.T) {
		cfg := validBase()
		cfg.Cache.Enabled = true
		cfg.EnableDecimalPrecision = false
		err := cfg.Validate()
		assert.ErrorContains(t, err, "cache requires enableDecimalPrecision to be true")
	})

	t.Run("EnabledCacheRejectsAsyncInsertWithoutWait", func(t *testing.T) {
		cfg := validBase()
		cfg.Cache.Enabled = true
		cfg.EnableDecimalPrecision = true
		cfg.AsyncInsert = true
		cfg.AsyncInsertWait = false
		err := cfg.Validate()
		assert.ErrorContains(t, err, "cache requires asyncInsertWait to be true when asyncInsert is enabled")
	})

	t.Run("EnabledCacheAllowsAsyncInsertWithWait", func(t *testing.T) {
		cfg := validBase()
		cfg.Cache.Enabled = true
		cfg.EnableDecimalPrecision = true
		cfg.AsyncInsert = true
		cfg.AsyncInsertWait = true
		require.NoError(t, cfg.Validate())
	})

	t.Run("EnabledCacheAllowsSyncInsert", func(t *testing.T) {
		cfg := validBase()
		cfg.Cache.Enabled = true
		cfg.EnableDecimalPrecision = true
		cfg.AsyncInsert = false
		cfg.AsyncInsertWait = false
		require.NoError(t, cfg.Validate())
	})

	t.Run("PropagatesCacheValidationErrors", func(t *testing.T) {
		cfg := validBase()
		cfg.Cache.MeterQueryThreshold = 1
		err := cfg.Validate()
		assert.ErrorContains(t, err, "cache:")
		assert.ErrorContains(t, err, "hot-meter selection not implemented")
	})
}
