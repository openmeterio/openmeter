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
	// Populate creates the materialized view with data from the events table
	// This is not safe to use in production as requires to stop ingestion
	PopulateMeter bool
	// CreateOrReplace is used to force the recreation of the materialized view
	// This is not safe to use in production as it will drop the existing views
	CreateOrReplaceMeter bool
}

// Validate validates the configuration.
func (c AggregationConfiguration) Validate() error {
	if err := c.ClickHouse.Validate(); err != nil {
		return fmt.Errorf("clickhouse: %w", err)
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
}

func (c ClickHouseAggregationConfiguration) Validate() error {
	if c.Address == "" {
		return errors.New("address is required")
	}

	if c.DialTimeout <= 0 {
		return errors.New("dial timeout must be greater than 0")
	}

	if c.MaxOpenConns <= 0 {
		return errors.New("max open connections must be greater than 0")
	}

	if c.MaxIdleConns <= 0 {
		return errors.New("max idle connections must be greater than 0")
	}

	if c.ConnMaxLifetime <= 0 {
		return errors.New("connection max lifetime must be greater than 0")
	}

	if c.BlockBufferSize <= 0 {
		return errors.New("block buffer size must be greater than 0")
	}

	return nil
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
	v.SetDefault("aggregation.clickhouse.address", "127.0.0.1:9000")
	v.SetDefault("aggregation.clickhouse.tls", false)
	v.SetDefault("aggregation.clickhouse.database", "openmeter")
	v.SetDefault("aggregation.clickhouse.username", "default")
	v.SetDefault("aggregation.clickhouse.password", "default")

	// ClickHouse connection options
	v.SetDefault("aggregation.clickhouse.dialTimeout", "10s")
	v.SetDefault("aggregation.clickhouse.maxOpenConns", 5)
	v.SetDefault("aggregation.clickhouse.maxIdleConns", 5)
	v.SetDefault("aggregation.clickhouse.connMaxLifetime", "10m")
	v.SetDefault("aggregation.clickhouse.blockBufferSize", 10)
}
