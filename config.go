package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"

	"github.com/openmeterio/openmeter/pkg/models"
)

// configuration holds any kind of configuration that comes from the outside world and
// is necessary for running the application.
// TODO: improve configuration options
type configuration struct {
	Address string

	Log logConfiguration

	// Telemetry configuration
	Telemetry struct {
		// Telemetry HTTP server address
		Address string
	}

	// Ingest configuration
	Ingest struct {
		Kafka struct {
			Broker         string
			Partitions     int
			SchemaRegistry string
		}
	}

	// Processor configuration
	Processor struct {
		KSQLDB struct {
			URL string
		}
	}

	Meters []*models.Meter
}

// Validate validates the configuration.
func (c configuration) Validate() error {
	if c.Address == "" {
		return errors.New("server address is required")
	}

	if c.Ingest.Kafka.Broker == "" {
		return errors.New("kafka broker is required")
	}

	if c.Ingest.Kafka.SchemaRegistry == "" {
		return errors.New("schema registry URL is required")
	}

	if c.Processor.KSQLDB.URL == "" {
		return errors.New("ksqldb URL is required")
	}

	if err := c.Log.Validate(); err != nil {
		return err
	}

	if c.Telemetry.Address == "" {
		return errors.New("telemetry http server address is required")
	}

	if len(c.Meters) == 0 {
		return errors.New("at least one meter is required")
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

type logConfiguration struct {
	// Format specifies the output log format.
	// Accepted values are: json, text
	Format string

	// Level is the minimum log level that should appear on the output.
	Level string
}

// Validate validates the configuration.
func (c logConfiguration) Validate() error {
	if !slices.Contains([]string{"json", "text", "tint"}, c.Format) {
		return fmt.Errorf("invalid format: %q", c.Format)
	}

	if !slices.Contains([]string{"debug", "info", "warn", "error"}, c.Level) {
		return fmt.Errorf("invalid format: %q", c.Level)
	}

	return nil
}

// configure configures some defaults in the Viper instance.
func configure(v *viper.Viper, flags *pflag.FlagSet) {
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

	// Log configuration
	v.SetDefault("log.format", "json")
	v.SetDefault("log.level", "info")
	//
	// Telemetry configuration
	flags.String("telemetry-address", ":10000", "Telemetry HTTP server address")
	_ = v.BindPFlag("telemetry.address", flags.Lookup("telemetry-address"))
	v.SetDefault("telemetry.address", ":10000")

	// Ingest configuration
	v.SetDefault("ingest.kafka.broker", "127.0.0.1:29092")
	// TODO: default to 100 in prod
	v.SetDefault("ingest.kafka.partitions", 1)
	v.SetDefault("ingest.kafka.schemaRegistry", "http://127.0.0.1:8081")

	// kSQL configuration
	v.SetDefault("processor.ksqldb.url", "http://127.0.0.1:8088")
}
