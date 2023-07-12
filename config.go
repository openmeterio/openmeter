package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/thmeitz/ksqldb-go/net"
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
		Kafka ingestKafkaConfiguration
	}

	// SchemaRegistry configuration
	SchemaRegistry struct {
		URL      string
		Username string
		Password string
	}

	// Processor configuration
	Processor struct {
		KSQLDB processorKSQLDBConfiguration
	}

	Meters []*models.Meter
}

// Validate validates the configuration.
func (c configuration) Validate() error {
	if c.Address == "" {
		return errors.New("server address is required")
	}

	if err := c.Ingest.Kafka.Validate(); err != nil {
		return err
	}

	if c.SchemaRegistry.URL == "" {
		return errors.New("schema registry URL is required")
	}

	if err := c.Processor.KSQLDB.Validate(); err != nil {
		return err
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

type ingestKafkaConfiguration struct {
	Broker           string
	SecurityProtocol string
	SaslMechanisms   string
	SaslUsername     string
	SaslPassword     string
	Partitions       int
}

// CreateKafkaConfig creates a Kafka config map.
func (c ingestKafkaConfiguration) CreateKafkaConfig() *kafka.ConfigMap {
	config := kafka.ConfigMap{
		"bootstrap.servers": c.Broker,
	}

	// TODO(hekike): is this really how it works? Looks like a copy-paste error at first.
	if c.SecurityProtocol != "" {
		config["security.protocol"] = c.SecurityProtocol
	}

	if c.SecurityProtocol != "" {
		config["sasl.mechanism"] = c.SaslMechanisms
	}

	if c.SecurityProtocol != "" {
		config["sasl.username"] = c.SaslUsername
	}

	if c.SecurityProtocol != "" {
		config["sasl.password"] = c.SaslPassword
	}

	return &config
}

// Validate validates the configuration.
func (c ingestKafkaConfiguration) Validate() error {
	if c.Broker == "" {
		return errors.New("kafka broker is required")
	}

	// TODO(hekike): is this really how it works?
	if c.SecurityProtocol != "" {
		if c.SaslMechanisms == "" {
			return errors.New("kafka sasl mechanisms is required")
		}

		if c.SaslUsername == "" {
			return errors.New("kafka sasl username is required")
		}

		if c.SaslPassword == "" {
			return errors.New("kafka sasl password is required")
		}
	}

	return nil
}

type processorKSQLDBConfiguration struct {
	URL      string
	Username string
	Password string
}

// CreateKafkaConfig creates a Kafka config map.
func (c processorKSQLDBConfiguration) CreateKSQLDBConfig() net.Options {
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
func (c processorKSQLDBConfiguration) Validate() error {
	if c.URL == "" {
		return errors.New("ksqldb URL is required")
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
	v.SetDefault("ingest.kafka.securityProtocol", "")
	v.SetDefault("ingest.kafka.saslMechanisms", "")
	v.SetDefault("ingest.kafka.saslUsername", "")
	v.SetDefault("ingest.kafka.saslPassword", "")
	// TODO: default to 100 in prod
	v.SetDefault("ingest.kafka.partitions", 1)

	// Schema Registry configuration
	v.SetDefault("schemaRegistry.url", "http://127.0.0.1:8081")
	v.SetDefault("schemaRegistry.username", "")
	v.SetDefault("schemaRegistry.password", "")

	// kSQL configuration
	v.SetDefault("processor.ksqldb.url", "http://127.0.0.1:8088")
	v.SetDefault("processor.ksqldb.username", "")
	v.SetDefault("processor.ksqldb.password", "")
}
