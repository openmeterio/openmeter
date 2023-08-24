package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
	"github.com/thmeitz/ksqldb-go/net"
)

type ProcessorConfiguration struct {
	KSQLDB     KSQLDBProcessorConfiguration
	ClickHouse ClickHouseProcessorConfiguration
}

// Validate validates the configuration.
func (c ProcessorConfiguration) Validate() error {
	if err := c.KSQLDB.Validate(); err != nil {
		return fmt.Errorf("ksqldb: %w", err)
	}

	if err := c.ClickHouse.Validate(); err != nil {
		return fmt.Errorf("clickhouse: %w", err)
	}

	return nil
}

type KSQLDBProcessorConfiguration struct {
	Enabled                     bool
	URL                         string
	Username                    string
	Password                    string
	DetectedEventsTopicTemplate string
}

// CreateKafkaConfig creates a Kafka config map.
func (c KSQLDBProcessorConfiguration) CreateKSQLDBConfig() net.Options {
	config := net.Options{
		BaseUrl:   c.URL,
		AllowHTTP: true,
	}

	if strings.HasPrefix(c.URL, "https://") {
		config.AllowHTTP = false
	}

	if c.Username != "" || c.Password != "" {
		config.Credentials = net.Credentials{
			Username: c.Username,
			Password: c.Password,
		}
	}

	return config
}

// Validate validates the configuration.
func (c KSQLDBProcessorConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.URL == "" {
		return errors.New("URL is required")
	}

	if c.DetectedEventsTopicTemplate == "" {
		return errors.New("namespace detected events topic template is required")
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
	v.SetDefault("processor.ksqldb.enabled", true)
	v.SetDefault("processor.ksqldb.url", "http://127.0.0.1:8088")
	v.SetDefault("processor.ksqldb.username", "")
	v.SetDefault("processor.ksqldb.password", "")
	v.SetDefault("processor.ksqldb.detectedEventsTopicTemplate", "om_%s_detected_events")

	// Processor Clickhouse configuration
	v.SetDefault("processor.clickhouse.enabled", false)
	v.SetDefault("processor.clickhouse.address", "127.0.0.1:9000")
	v.SetDefault("processor.clickhouse.tls", false)
	v.SetDefault("processor.clickhouse.database", "default")
	v.SetDefault("processor.clickhouse.username", "default")
	v.SetDefault("processor.clickhouse.password", "")
}
