package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

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

	// Namespace configuration
	Namespace namespaceConfiguration

	// Ingest configuration
	Ingest struct {
		Kafka ingestKafkaConfiguration
	}

	// Dedupe configuration
	Dedupe struct {
		Redis  dedupeRedisConfiguration
		Memory dedupeMemoryConfiguration
	}

	// SchemaRegistry configuration
	SchemaRegistry struct {
		URL      string
		Username string
		Password string
	}

	// Processor configuration
	Processor struct {
		KSQLDB     processorKSQLDBConfiguration
		ClickHouse processorClickhouseConfiguration
	}

	// Sink configuration
	Sink struct {
		KafkaConnect sinkKafkaConnectConfiguration
	}

	Meters []*models.Meter
}

// Validate validates the configuration.
func (c configuration) Validate() error {
	if c.Address == "" {
		return errors.New("server address is required")
	}

	if err := c.Namespace.Validate(); err != nil {
		return err
	}

	if err := c.Ingest.Kafka.Validate(); err != nil {
		return err
	}

	if err := c.Processor.KSQLDB.Validate(); err != nil {
		return err
	}

	if err := c.Processor.ClickHouse.Validate(); err != nil {
		return err
	}

	if err := c.Sink.KafkaConnect.Validate(); err != nil {
		return err
	}

	if err := c.Dedupe.Redis.Validate(); err != nil {
		return err
	}

	if err := c.Log.Validate(); err != nil {
		return err
	}

	if c.Telemetry.Address == "" {
		return errors.New("telemetry http server address is required")
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

// Namespace configuration
type namespaceConfiguration struct {
	Default           string
	DisableManagement bool
}

func (c namespaceConfiguration) Validate() error {
	if c.Default == "" {
		return errors.New("default namespace is required")
	}

	return nil
}

// Ingest Kafka configuration
type ingestKafkaConfiguration struct {
	Broker              string
	SecurityProtocol    string
	SaslMechanisms      string
	SaslUsername        string
	SaslPassword        string
	Partitions          int
	EventsTopicTemplate string
}

// CreateKafkaConfig creates a Kafka config map.
func (c ingestKafkaConfiguration) CreateKafkaConfig() *kafka.ConfigMap {
	config := kafka.ConfigMap{
		"bootstrap.servers": c.Broker,
	}

	if c.SecurityProtocol != "" {
		config["security.protocol"] = c.SecurityProtocol
	}

	if c.SaslMechanisms != "" {
		config["sasl.mechanism"] = c.SaslMechanisms
	}

	if c.SaslUsername != "" {
		config["sasl.username"] = c.SaslUsername
	}

	if c.SaslPassword != "" {
		config["sasl.password"] = c.SaslPassword
	}

	return &config
}

// Validate validates the configuration.
func (c ingestKafkaConfiguration) Validate() error {
	if c.Broker == "" {
		return errors.New("kafka broker is required")
	}

	if c.EventsTopicTemplate == "" {
		return errors.New("events topic template is required")
	}

	return nil
}

// KSQLDB Processor configuration
type processorKSQLDBConfiguration struct {
	Enabled                     bool
	URL                         string
	Username                    string
	Password                    string
	DetectedEventsTopicTemplate string
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
	if !c.Enabled {
		return nil
	}

	if c.URL == "" {
		return errors.New("ksqldb URL is required")
	}

	if c.DetectedEventsTopicTemplate == "" {
		return errors.New("namespace detected events topic template is required")
	}

	return nil
}

// Clickhouse configuration
type processorClickhouseConfiguration struct {
	Enabled  bool
	Address  string
	TLS      bool
	Database string
	Username string
	Password string
}

func (c processorClickhouseConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Address == "" {
		return errors.New("clickhouse address is required")
	}

	return nil
}

// Sink configuration
type sinkKafkaConnectConfiguration struct {
	Enabled    bool
	URL        string
	ClickHouse kafkaSinkClickhouseConfiguration
}

func (c sinkKafkaConnectConfiguration) Validate() error {
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
type kafkaSinkClickhouseConfiguration struct {
	Hostname string
	Port     int
	SSL      bool
	Database string
	Username string
	Password string
}

func (c kafkaSinkClickhouseConfiguration) Validate() error {
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

// Dedupe redis configuration
type dedupeRedisConfiguration struct {
	Enabled     bool
	Address     string
	Database    int
	Username    string
	Password    string
	Expiration  time.Duration
	UseSentinel bool
	MasterName  string
}

func (c dedupeRedisConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Address == "" {
		return errors.New("dedupe redis address is required")
	}

	if c.UseSentinel {
		if c.MasterName == "" {
			return errors.New("dedupe redis master name is required")
		}
	}

	return nil
}

// Dedupe memory configuration
type dedupeMemoryConfiguration struct {
	Enabled bool
	Size    int
}

func (c dedupeMemoryConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Size == 0 {
		return errors.New("dedupe memory size is required")
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

	// Namespace configuration
	v.SetDefault("namespace.default", "default")
	v.SetDefault("namespace.disableManagement", false)

	// Ingest configuration
	v.SetDefault("ingest.kafka.broker", "127.0.0.1:29092")
	v.SetDefault("ingest.kafka.securityProtocol", "")
	v.SetDefault("ingest.kafka.saslMechanisms", "")
	v.SetDefault("ingest.kafka.saslUsername", "")
	v.SetDefault("ingest.kafka.saslPassword", "")
	v.SetDefault("ingest.kafka.partitions", 1)
	v.SetDefault("ingest.kafka.eventsTopicTemplate", "om_%s_events")

	// Schema Registry configuration
	v.SetDefault("schemaRegistry.url", "")
	v.SetDefault("schemaRegistry.username", "")
	v.SetDefault("schemaRegistry.password", "")

	// Processor ksqlDB configuration
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

	// Sink Kafka Connect configuration
	v.SetDefault("sink.kafkaConnect.enabled", false)
	v.SetDefault("sink.kafkaConnect.url", "http://127.0.0.1:8083")
	v.SetDefault("sink.kafkaConnect.clickhouse.hostname", "clickhouse")
	v.SetDefault("sink.kafkaConnect.clickhouse.port", 8123)
	v.SetDefault("sink.kafkaConnect.clickhouse.ssl", false)
	v.SetDefault("sink.kafkaConnect.clickhouse.database", "default")
	v.SetDefault("sink.kafkaConnect.clickhouse.username", "default")
	v.SetDefault("sink.kafkaConnect.clickhouse.password", "")

	// Dedupe Redis configuration
	v.SetDefault("dedupe.redis", false)
	v.SetDefault("dedupe.redis.address", "127.0.0.1:6379")
	v.SetDefault("dedupe.redis.database", 0)
	v.SetDefault("dedupe.redis.username", "")
	v.SetDefault("dedupe.redis.password", "")
	v.SetDefault("dedupe.redis.expiration", "24h")
	v.SetDefault("dedupe.redis.useSentintel", false)
	v.SetDefault("dedupe.redis.masterName", "")

	// Dedupe Memory configuration
	v.SetDefault("dedupe.memory", false)
	v.SetDefault("dedupe.memory.size", 128)
}
