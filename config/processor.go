package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

type ProcessorConfiguration struct {
	ClickHouse ClickHouseProcessorConfiguration
}

// Validate validates the configuration.
func (c ProcessorConfiguration) Validate() error {
	if err := c.ClickHouse.Validate(); err != nil {
		return fmt.Errorf("clickhouse: %w", err)
	}

	return nil
}

type ClickHouseProcessorConfiguration struct {
	Enabled  bool
	Address  string
	TLS      bool
	Username string
	Password string
	Database string
}

func (c ClickHouseProcessorConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Address == "" {
		return errors.New("address is required")
	}

	return nil
}

// configureProcessor configures some defaults in the Viper instance.
func configureProcessor(v *viper.Viper) {
	v.SetDefault("processor.clickhouse.enabled", true)
	v.SetDefault("processor.clickhouse.address", "127.0.0.1:9000")
	v.SetDefault("processor.clickhouse.tls", false)
	v.SetDefault("processor.clickhouse.database", "openmeter")
	v.SetDefault("processor.clickhouse.username", "default")
	v.SetDefault("processor.clickhouse.password", "default")
}
