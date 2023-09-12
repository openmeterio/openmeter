package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

type AggregationConfiguration struct {
	ClickHouse ClickHouseAggregationConfiguration
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
}

func (c ClickHouseAggregationConfiguration) Validate() error {
	if c.Address == "" {
		return errors.New("address is required")
	}

	return nil
}

// ConfigureAggregation configures some defaults in the Viper instance.
func ConfigureAggregation(v *viper.Viper) {
	v.SetDefault("aggregation.clickhouse.address", "127.0.0.1:9000")
	v.SetDefault("aggregation.clickhouse.tls", false)
	v.SetDefault("aggregation.clickhouse.database", "openmeter")
	v.SetDefault("aggregation.clickhouse.username", "default")
	v.SetDefault("aggregation.clickhouse.password", "default")
}
