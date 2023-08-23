// Package config loads application configuration.
package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/pkg/models"
)

// Configuration holds any kind of Configuration that comes from the outside world and
// is necessary for running the application.
// TODO: improve Configuration options
type Configuration struct {
	Address string

	Environment string

	// Telemetry configuration
	Telemetry TelemetryConfig

	// Namespace configuration
	Namespace NamespaceConfiguration

	// Ingest configuration
	Ingest IngestConfiguration

	// Processor configuration
	Processor ProcessorConfiguration

	// Dedupe configuration
	Dedupe DedupeConfiguration

	// SchemaRegistry configuration
	SchemaRegistry struct {
		URL      string
		Username string
		Password string
	}

	// Sink configuration
	Sink struct {
		KafkaConnect SinkKafkaConnectConfiguration
	}

	Meters []*models.Meter
}

// Validate validates the configuration.
func (c Configuration) Validate() error {
	if c.Address == "" {
		return errors.New("server address is required")
	}

	if err := c.Namespace.Validate(); err != nil {
		return fmt.Errorf("namespace: %w", err)
	}

	if err := c.Ingest.Validate(); err != nil {
		return fmt.Errorf("ingest: %w", err)
	}

	if err := c.Processor.Validate(); err != nil {
		return fmt.Errorf("processor: %w", err)
	}

	if err := c.Sink.KafkaConnect.Validate(); err != nil {
		return err
	}

	if err := c.Dedupe.Validate(); err != nil {
		return fmt.Errorf("dedupe: %w", err)
	}

	if err := c.Telemetry.Validate(); err != nil {
		return fmt.Errorf("telemetry: %w", err)
	}

	for _, m := range c.Meters {
		// set default window size
		if m.WindowSize == "" {
			m.WindowSize = models.WindowSizeMinute
		}

		if err := m.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Sink configuration
type SinkKafkaConnectConfiguration struct {
	Enabled    bool
	URL        string
	ClickHouse KafkaSinkClickhouseConfiguration
}

func (c SinkKafkaConnectConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.URL == "" {
		return errors.New("kafka connect url is required")
	}

	if err := c.ClickHouse.Validate(); err != nil {
		return err
	}

	return nil
}

// Clickhouse configuration
// This may feel repetative but clikhouse sink and processor configs can be different,
// for example Kafka Connect ClickHouse plugin uses 8123 HTTP port while client uses native protocol's 9000 port.
// Hostname can be also different, as Kafka Connect and ClickHouse communicates inside the docker compose network.
// This why we default hostname in config to `clickhouse`.
type KafkaSinkClickhouseConfiguration struct {
	Hostname string
	Port     int
	SSL      bool
	Database string
	Username string
	Password string
}

func (c KafkaSinkClickhouseConfiguration) Validate() error {
	if c.Hostname == "" {
		return errors.New("kafka sink clickhouse hostname is required")
	}
	if c.Port == 0 {
		return errors.New("kafka sink clickhouse port is required")
	}
	if c.Database == "" {
		return errors.New("kafka sink clickhouse database is required")
	}
	if c.Username == "" {
		return errors.New("kafka sink clickhouse username is required")
	}

	return nil
}

// Configure configures some defaults in the Viper instance.
func Configure(v *viper.Viper, flags *pflag.FlagSet) {
	// Viper settings
	v.AddConfigPath(".")

	// Environment variable settings
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

	// Server configuration
	flags.String("address", ":8888", "Server address")
	_ = v.BindPFlag("address", flags.Lookup("address"))
	v.SetDefault("address", ":8888")

	// Environment used for identifying the service environment
	v.SetDefault("environment", "unknown")

	configureTelemetry(v, flags)
	configureNamespace(v)
	configureIngest(v)

	// Schema Registry configuration
	v.SetDefault("schemaRegistry.url", "")
	v.SetDefault("schemaRegistry.username", "")
	v.SetDefault("schemaRegistry.password", "")

	configureProcessor(v)

	// Sink Kafka Connect configuration
	v.SetDefault("sink.kafkaConnect.enabled", false)
	v.SetDefault("sink.kafkaConnect.url", "http://127.0.0.1:8083")
	v.SetDefault("sink.kafkaConnect.clickhouse.hostname", "clickhouse")
	v.SetDefault("sink.kafkaConnect.clickhouse.port", 8123)
	v.SetDefault("sink.kafkaConnect.clickhouse.ssl", false)
	v.SetDefault("sink.kafkaConnect.clickhouse.database", "default")
	v.SetDefault("sink.kafkaConnect.clickhouse.username", "default")
	v.SetDefault("sink.kafkaConnect.clickhouse.password", "")

	configureDedupe(v)
}
