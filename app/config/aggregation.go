package config

import (
	"crypto/tls"
	"errors"
	"fmt"
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
<<<<<<< HEAD
=======

	// Engine specific options

	// Populate creates the materialized view with data from the events table
	// This is not safe to use in production as requires to stop ingestion
	PopulateMeter bool
	// CreateOrReplace is used to force the recreation of the materialized view
	// This is not safe to use in production as it will drop the existing views
	CreateOrReplaceMeter bool
	// QueryRawEvents is used to query the raw events table instead of the materialized view
	QueryRawEvents bool

	// QueryCache is the cache configuration
	QueryCache AggregationQueryCacheConfiguration
>>>>>>> 4c356376 (feat(meter): query cache configurable)
}

// Validate validates the configuration.
func (c AggregationConfiguration) Validate() error {
	if err := c.ClickHouse.Validate(); err != nil {
		return fmt.Errorf("clickhouse: %w", err)
	}

	if c.EventsTableName == "" {
		return errors.New("events table is required")
	}

	if c.AsyncInsertWait && !c.AsyncInsert {
		return errors.New("async insert wait is set but async insert is not")
	}

	return nil
}

// AggregationQueryCacheConfiguration is the cache configuration for the aggregation engine
type AggregationQueryCacheConfiguration struct {
	Enabled bool

	// Minimum query period that can be cached
	MinimumCacheableQueryPeriod time.Duration
	// Minimum age after usage data is cachable
	MinimumCacheableUsageAge time.Duration
}

func (c AggregationQueryCacheConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.MinimumCacheableQueryPeriod <= 0 {
		return errors.New("minimum cacheable query period is required")
	}

	if c.MinimumCacheableUsageAge <= 0 {
		return errors.New("minimum cacheable usage age is required")
	}

	return nil
}

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

	Tracing bool
}

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
		options.TLS = &tls.Config{}
	}

	return options
}

// ConfigureAggregation configures some defaults in the Viper instance.
func ConfigureAggregation(v *viper.Viper) {
	v.SetDefault("aggregation.eventsTableName", "om_events")
	v.SetDefault("aggregation.asyncInsert", false)
	v.SetDefault("aggregation.asyncInsertWait", false)

	// Cache configuration
	v.SetDefault("aggregation.queryCache.enabled", false)
	v.SetDefault("aggregation.queryCache.minimumCacheableQueryPeriod", "72h")
	v.SetDefault("aggregation.queryCache.minimumCacheableUsageAge", "24h")

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
}
