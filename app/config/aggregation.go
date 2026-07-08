package config

import (
	"crypto/tls"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/spf13/viper"
)

type AggregationConfiguration struct {
	ClickHouse ClickHouseAggregationConfiguration

	EventsTableName string

	// Set true for ClickHouse first store the incoming inserts into an in-memory buffer
	// before flushing them regularly to disk.
	// See https://clickhouse.com/docs/en/cloud/bestpractices/asynchronous-inserts
	AsyncInsert bool
	// Set true if you want an insert statement to return with an acknowledgment immediately
	// without waiting for the data got inserted into the buffer.
	// Setting true can cause silent errors that you need to monitor separately.
	AsyncInsertWait bool

	// See https://clickhouse.com/docs/en/operations/settings/settings
	// For example, you can set the `max_insert_threads` setting to control the number of threads
	// or the `parallel_view_processing` setting to enable pushing to attached views concurrently.
	InsertQuerySettings map[string]string

	// MeterQuerySettings is the settings for the meter query
	// For example, you can set the `enable_parallel_replicas` and `max_parallel_replicas` settings.
	// See https://clickhouse.com/docs/en/operations/settings/settings
	MeterQuerySettings map[string]string

	// EnablePrewhere is the setting to enable prewhere for the meter query.
	EnablePrewhere bool

	// EnableDecimalPrecision enables high precision decimal calculations for meter queries.
	// When enabled, values are calculated using Decimal128 instead of Float64, providing
	// higher precision for financial calculations at the cost of some performance.
	EnableDecimalPrecision bool

	// Cache configures the refreshable materialized view based meter cache.
	Cache AggregationCacheConfiguration
}

// Validate validates the configuration.
func (c AggregationConfiguration) Validate() error {
	var errs []error

	if err := c.ClickHouse.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("clickhouse: %w", err))
	}

	if c.EventsTableName == "" {
		errs = append(errs, errors.New("events table is required"))
	}

	if c.AsyncInsertWait && !c.AsyncInsert {
		errs = append(errs, errors.New("async insert wait is set but async insert is not"))
	}

	if err := c.Cache.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("cache: %w", err))
	}

	// The cache reads via the newest-wins argMax over Decimal128 columns, so it can only
	// serve queries that already run in decimal precision mode; enabling the cache without
	// EnableDecimalPrecision would leave every eligible query silently unservable and defeats
	// the purpose of turning the cache on.
	if c.Cache.Enabled && !c.EnableDecimalPrecision {
		errs = append(errs, errors.New("cache requires enableDecimalPrecision to be true"))
	}

	// Async inserts without waiting for the buffer flush acknowledge before rows are queryable,
	// so a refresh scheduled right after ingestion can miss just-inserted rows and never revisit
	// them under APPEND semantics; requiring AsyncInsertWait when the cache is enabled keeps
	// refreshes able to observe the rows they are supposed to cover.
	if c.Cache.Enabled && c.AsyncInsert && !c.AsyncInsertWait {
		errs = append(errs, errors.New("cache requires asyncInsertWait to be true when asyncInsert is enabled"))
	}

	return errors.Join(errs...)
}

// ClickHouseAggregationConfiguration is the configuration for the ClickHouse aggregation engine
type ClickHouseAggregationConfiguration struct {
	Address  string
	TLS      bool
	Username string
	Password string
	Database string

	// ClickHouse connection options
	DialTimeout     time.Duration
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	BlockBufferSize uint8

	Tracing     bool
	PoolMetrics ClickhousePoolMetricsConfig

	Retry ClickhouseQueryRetryConfig
}

// Validate validates the configuration.
func (c ClickHouseAggregationConfiguration) Validate() error {
	var errs []error

	if c.Address == "" {
		errs = append(errs, errors.New("address is required"))
	}

	if c.DialTimeout <= 0 {
		errs = append(errs, errors.New("dial timeout must be greater than 0"))
	}

	if c.MaxOpenConns <= 0 {
		errs = append(errs, errors.New("max open connections must be greater than 0"))
	}

	if c.MaxIdleConns <= 0 {
		errs = append(errs, errors.New("max idle connections must be greater than 0"))
	}

	if c.ConnMaxLifetime <= 0 {
		errs = append(errs, errors.New("connection max lifetime must be greater than 0"))
	}

	if c.BlockBufferSize <= 0 {
		errs = append(errs, errors.New("block buffer size must be greater than 0"))
	}

	if err := c.Retry.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("retry: %w", err))
	}

	if err := c.PoolMetrics.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("pool metrics: %w", err))
	}

	return errors.Join(errs...)
}

func (c ClickHouseAggregationConfiguration) GetClientOptions() *clickhouse.Options {
	options := &clickhouse.Options{
		Addr: []string{c.Address},
		Auth: clickhouse.Auth{
			Database: c.Database,
			Username: c.Username,
			Password: c.Password,
		},
		DialTimeout:      c.DialTimeout,
		MaxOpenConns:     c.MaxOpenConns,
		MaxIdleConns:     c.MaxIdleConns,
		ConnMaxLifetime:  c.ConnMaxLifetime,
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
		BlockBufferSize:  c.BlockBufferSize,
	}
	// This minimal TLS.Config is normally sufficient to connect to the secure native port (normally 9440) on a ClickHouse server.
	// See: https://clickhouse.com/docs/en/integrations/go#using-tls
	if c.TLS {
		options.TLS = &tls.Config{
			MinVersion: tls.VersionTLS13,
		}
	}

	return options
}

type ClickhouseQueryRetryConfig struct {
	Enabled           bool
	MaxTries          int
	RetryWaitDuration time.Duration
	MaxDelay          time.Duration
}

func (c ClickhouseQueryRetryConfig) Validate() error {
	var errs []error

	if !c.Enabled {
		return nil
	}

	if c.MaxTries < 1 {
		errs = append(errs, errors.New("max retries must be greater than or equal to 1"))
	}

	if c.RetryWaitDuration <= 0 {
		errs = append(errs, errors.New("retry wait duration must be greater than 0"))
	}

	if c.MaxDelay < 0 {
		errs = append(errs, errors.New("max delay must not be negative"))
	}

	return errors.Join(errs...)
}

type ClickhousePoolMetricsConfig struct {
	Enabled      bool
	PollInterval time.Duration
}

func (c ClickhousePoolMetricsConfig) Validate() error {
	var errs []error

	if !c.Enabled {
		return nil
	}

	if c.PollInterval <= 0 {
		errs = append(errs, errors.New("poll interval must be greater than 0"))
	}

	return errors.Join(errs...)
}

// AggregationCacheGrain is the bucket width the meter cache maintains per meter.
// It intentionally supports a narrower set of values than meter.WindowSize (which also
// allows SECOND and MONTH): the cache tiles live queries onto fixed rollup buckets, and
// second-level buckets are prohibitively expensive to maintain while month buckets are
// wide enough to be assembled from day/hour buckets at read time instead.
type AggregationCacheGrain string

const (
	AggregationCacheGrainMinute AggregationCacheGrain = "minute"
	AggregationCacheGrainHour   AggregationCacheGrain = "hour"
	AggregationCacheGrainDay    AggregationCacheGrain = "day"
)

// Values provides the list of valid values for the AggregationCacheGrain enum.
func (AggregationCacheGrain) Values() []string {
	return []string{
		string(AggregationCacheGrainMinute),
		string(AggregationCacheGrainHour),
		string(AggregationCacheGrainDay),
	}
}

// AggregationCacheConfiguration configures the refreshable materialized view based meter
// cache. The cache is inert (Enabled: false) by default: creating and maintaining the
// per-meter materialized views is a ClickHouse-side cost that operators must opt into
// after confirming they run a single ClickHouse server (system.view_refreshes and the
// scheduled refreshes it drives are per-server state; clusters/load-balanced setups are
// not supported in v1 and the read-side gate falls back to live queries there).
type AggregationCacheConfiguration struct {
	// Enabled turns on materialized-view backed caching for meter queries.
	Enabled bool

	// RefreshInterval is how often ClickHouse re-runs each per-meter materialized view.
	RefreshInterval time.Duration

	// MinimumUsageAge is the freshness horizon: buckets newer than this are always served
	// live so that in-flight, not-yet-settled usage is never read from a stale cache row.
	MinimumUsageAge time.Duration

	// WindowSize is the rollup bucket width (grain) maintained by the cache. Choosing
	// "minute" multiplies the number of stored rows accordingly; consider the storage
	// impact before enabling minute-grain caching on high-cardinality meters.
	WindowSize AggregationCacheGrain

	// MeterQueryThreshold reserves the "cache only hot meters" selection strategy for a
	// future release. v1 always caches every meter (threshold == 0); a positive value is
	// rejected at validation time so it cannot silently no-op.
	MeterQueryThreshold int
}

// Validate validates the configuration.
func (c AggregationCacheConfiguration) Validate() error {
	var errs []error

	if !slices.Contains(c.WindowSize.Values(), string(c.WindowSize)) {
		errs = append(errs, fmt.Errorf("window size must be one of %v, got %q", c.WindowSize.Values(), c.WindowSize))
	}

	// Refreshes only recompute buckets covered by the dirty-window lookback (see
	// meter_cache_mv.go); if the freshness horizon were shorter than the refresh interval,
	// a bucket could become "settled" and eligible for caching in between two refreshes
	// without ever having been recomputed, serving a stale (pre-settlement) row as final.
	if c.MinimumUsageAge < c.RefreshInterval {
		errs = append(errs, errors.New("minimum usage age must be greater than or equal to refresh interval"))
	}

	// Hot-meter selection (caching only meters above a query-volume threshold) is not
	// implemented yet; the key is reserved so existing configs keep working once it ships
	// without silently changing behavior for anyone who sets a positive value today.
	if c.MeterQueryThreshold > 0 {
		errs = append(errs, errors.New("hot-meter selection not implemented"))
	}

	if c.MeterQueryThreshold < 0 {
		errs = append(errs, errors.New("meter query threshold must not be negative"))
	}

	return errors.Join(errs...)
}

// ConfigureAggregation configures some defaults in the Viper instance.
func ConfigureAggregation(v *viper.Viper) {
	v.SetDefault("aggregation.eventsTableName", "om_events")
	v.SetDefault("aggregation.asyncInsert", false)
	v.SetDefault("aggregation.asyncInsertWait", false)

	v.SetDefault("aggregation.clickhouse.address", "127.0.0.1:9000")
	v.SetDefault("aggregation.clickhouse.tls", false)
	v.SetDefault("aggregation.clickhouse.database", "openmeter")
	v.SetDefault("aggregation.clickhouse.username", "default")
	v.SetDefault("aggregation.clickhouse.password", "default")
	v.SetDefault("aggregation.clickhouse.tracing", false)
	// ClickHouse connection options
	v.SetDefault("aggregation.clickhouse.dialTimeout", "10s")
	v.SetDefault("aggregation.clickhouse.maxOpenConns", 5)
	v.SetDefault("aggregation.clickhouse.maxIdleConns", 5)
	v.SetDefault("aggregation.clickhouse.connMaxLifetime", "10m")
	v.SetDefault("aggregation.clickhouse.blockBufferSize", 10)

	// Retry
	v.SetDefault("aggregation.clickhouse.retry.enabled", false)
	v.SetDefault("aggregation.clickhouse.retry.maxTries", 3)
	v.SetDefault("aggregation.clickhouse.retry.retryWaitDuration", "20ms")
	v.SetDefault("aggregation.clickhouse.retry.maxDelay", "5s")

	// Pool metrics
	v.SetDefault("aggregation.clickhouse.poolMetrics.enabled", true)
	v.SetDefault("aggregation.clickhouse.poolMetrics.pollInterval", "5s")

	// Decimal precision
	v.SetDefault("aggregation.enableDecimalPrecision", false)

	// Meter cache
	v.SetDefault("aggregation.cache.enabled", false)
	v.SetDefault("aggregation.cache.refreshInterval", "10m")
	v.SetDefault("aggregation.cache.minimumUsageAge", "1h")
	v.SetDefault("aggregation.cache.windowSize", "hour")
	v.SetDefault("aggregation.cache.meterQueryThreshold", 0)
}
