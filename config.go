package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/lmittmann/tint"
	"github.com/mitchellh/mapstructure"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/thmeitz/ksqldb-go/net"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/openmeterio/openmeter/internal/dedupe/memorydedupe"
	"github.com/openmeterio/openmeter/internal/dedupe/redisdedupe"
	"github.com/openmeterio/openmeter/internal/ingest"
	"github.com/openmeterio/openmeter/pkg/models"
)

// configuration holds any kind of configuration that comes from the outside world and
// is necessary for running the application.
// TODO: improve configuration options
type configuration struct {
	Address string

	Environment string

	Log logConfiguration

	// Telemetry configuration
	Telemetry telemetryConfig

	// Namespace configuration
	Namespace namespaceConfiguration

	// Ingest configuration
	Ingest struct {
		Kafka ingestKafkaConfiguration
	}

	// Dedupe configuration
	Dedupe dedupeConfiguration

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

	if err := c.Dedupe.Validate(); err != nil {
		return err
	}

	if err := c.Telemetry.Validate(); err != nil {
		return err
	}

	if err := c.Log.Validate(); err != nil {
		return err
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
func (c ingestKafkaConfiguration) CreateKafkaConfig() kafka.ConfigMap {
	config := kafka.ConfigMap{
		"bootstrap.servers": c.Broker,

		// Required for logging
		"go.logs.channel.enable": true,
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

	return config
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

// Requires [mapstructurex.MapDecoderHookFunc] to be high up in the decode hook chain.
type dedupeConfiguration struct {
	Enabled bool

	dedupeDriverConfiguration
}

func (c dedupeConfiguration) NewDeduplicator() (ingest.Deduplicator, error) {
	if !c.Enabled {
		return nil, errors.New("dedupe: disabled")
	}

	if c.dedupeDriverConfiguration == nil {
		return nil, errors.New("dedupe: missing driver configuration")
	}

	return c.dedupeDriverConfiguration.NewDeduplicator()
}

func (c dedupeConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.dedupeDriverConfiguration == nil {
		return errors.New("dedupe: missing driver configuration")
	}

	if err := c.dedupeDriverConfiguration.Validate(); err != nil {
		return fmt.Errorf("dedupe: %w", err)
	}

	return nil
}

type rawDedupeConfiguration struct {
	Enabled bool
	Driver  string
	Config  map[string]any
}

func (c *dedupeConfiguration) DecodeMap(v map[string]any) error {
	var rawConfig rawDedupeConfiguration

	err := mapstructure.Decode(v, &rawConfig)
	if err != nil {
		return err
	}

	c.Enabled = rawConfig.Enabled

	// Deduplication is disabled and not configured, so skip further decoding
	if !c.Enabled && rawConfig.Driver == "" {
		return nil
	}

	switch rawConfig.Driver {
	case "memory":
		var driverConfig dedupeDriverMemoryConfiguration

		err := mapstructure.Decode(rawConfig.Config, &driverConfig)
		if err != nil {
			return fmt.Errorf("dedupe: decoding memory driver config: %w", err)
		}

		c.dedupeDriverConfiguration = driverConfig

	case "redis":
		var driverConfig dedupeDriverRedisConfiguration

		decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			Metadata:         nil,
			Result:           &driverConfig,
			WeaklyTypedInput: true,
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
			),
		})
		if err != nil {
			return fmt.Errorf("dedupe: creating decoder: %w", err)
		}

		err = decoder.Decode(rawConfig.Config)
		if err != nil {
			return fmt.Errorf("dedupe: decoding redis driver config: %w", err)
		}

		c.dedupeDriverConfiguration = driverConfig

	case "":
		return errors.New("dedupe: missing driver")

	default:
		return fmt.Errorf("dedupe: unknown driver: %s", rawConfig.Driver)
	}

	return nil
}

type dedupeDriverConfiguration interface {
	NewDeduplicator() (ingest.Deduplicator, error)
	Validate() error
}

// Dedupe memory driver configuration
type dedupeDriverMemoryConfiguration struct {
	Enabled bool
	Size    int
}

func (c dedupeDriverMemoryConfiguration) NewDeduplicator() (ingest.Deduplicator, error) {
	return memorydedupe.NewDeduplicator(c.Size)
}

func (c dedupeDriverMemoryConfiguration) Validate() error {
	if c.Size == 0 {
		return errors.New("memory: size is required")
	}

	return nil
}

// Dedupe redis driver configuration
type dedupeDriverRedisConfiguration struct {
	Address    string
	Database   int
	Username   string
	Password   string
	Expiration time.Duration
	Sentinel   struct {
		Enabled    bool
		MasterName string
	}
	TLS struct {
		Enabled            bool
		InsecureSkipVerify bool
	}
}

func (c dedupeDriverRedisConfiguration) NewDeduplicator() (ingest.Deduplicator, error) {
	var tlsConfig *tls.Config

	if c.TLS.Enabled {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: c.TLS.InsecureSkipVerify,
		}
	}

	var redisClient *redis.Client

	if c.Sentinel.Enabled {
		redisClient = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    c.Sentinel.MasterName,
			SentinelAddrs: []string{c.Address},
			DB:            c.Database,
			Username:      c.Username,
			Password:      c.Password,
			TLSConfig:     tlsConfig,
		})
	} else {
		redisClient = redis.NewClient(&redis.Options{
			Addr:      c.Address,
			DB:        c.Database,
			Username:  c.Username,
			Password:  c.Password,
			TLSConfig: tlsConfig,
		})
	}

	// Enable tracing
	// TODO: use configured tracer provider
	if err := redisotel.InstrumentTracing(redisClient); err != nil {
		return nil, err
	}

	// Enable metrics
	// TODO: use configured tracer provider
	if err := redisotel.InstrumentMetrics(redisClient); err != nil {
		return nil, err
	}

	// TODO: close redis client when shutting down
	// TODO: register health check for redis
	return redisdedupe.Deduplicator{
		Redis:      redisClient,
		Expiration: c.Expiration,
	}, nil
}

func (c dedupeDriverRedisConfiguration) Validate() error {
	if c.Address == "" {
		return errors.New("redis: address is required")
	}

	if c.Sentinel.Enabled {
		if c.Sentinel.MasterName == "" {
			return errors.New("redis: sentinel: master name is required")
		}
	}

	return nil
}

type logConfiguration struct {
	// Format specifies the output log format.
	// Accepted values are: json, text
	Format string

	// Level is the minimum log level that should appear on the output.
	//
	// Requires [mapstructure.TextUnmarshallerHookFunc] to be high up in the decode hook chain.
	Level slog.Level
}

// Validate validates the configuration.
func (c logConfiguration) Validate() error {
	if !slices.Contains([]string{"json", "text", "tint"}, c.Format) {
		return fmt.Errorf("log: invalid format: %q", c.Format)
	}

	if !slices.Contains([]slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError}, c.Level) {
		return fmt.Errorf("log: invalid level: %q", c.Level)
	}

	return nil
}

func (c logConfiguration) NewHandler(w io.Writer) slog.Handler {
	switch c.Format {
	case "json":
		return slog.NewJSONHandler(w, &slog.HandlerOptions{Level: c.Level})

	case "text":
		return slog.NewTextHandler(w, &slog.HandlerOptions{Level: c.Level})

	case "tint":
		return tint.NewHandler(os.Stdout, &tint.Options{Level: c.Level})
	}

	return slog.NewJSONHandler(w, &slog.HandlerOptions{Level: c.Level})
}

type telemetryConfig struct {
	// Telemetry HTTP server address
	Address string

	Trace traceTelemetryConfig
}

// Validate validates the configuration.
func (c telemetryConfig) Validate() error {
	if c.Address == "" {
		return errors.New("telemetry: http server address is required")
	}

	if err := c.Trace.Validate(); err != nil {
		return fmt.Errorf("telemetry: %w", err)
	}

	return nil
}

type traceTelemetryConfig struct {
	Exporter exporterTraceTelemetryConfig
	Sampler  string
}

// Validate validates the configuration.
func (c traceTelemetryConfig) Validate() error {
	if _, err := strconv.ParseFloat(c.Sampler, 64); err != nil && !slices.Contains([]string{"always", "never"}, c.Sampler) {
		return fmt.Errorf("trace: sampler either needs to be always|never or a ration, got: %s", c.Sampler)
	}

	if err := c.Exporter.Validate(); err != nil {
		return fmt.Errorf("trace: %w", err)
	}

	return nil
}

func (c traceTelemetryConfig) GetSampler() sdktrace.Sampler {
	switch c.Sampler {
	case "always":
		return sdktrace.AlwaysSample()

	case "never":
		return sdktrace.NeverSample()

	default:
		ratio, err := strconv.ParseFloat(c.Sampler, 64)
		if err != nil {
			panic(fmt.Errorf("trace: invalid ratio: %w", err))
		}

		return sdktrace.TraceIDRatioBased(ratio)
	}
}

type exporterTraceTelemetryConfig struct {
	Enabled bool
	Address string
}

// Validate validates the configuration.
func (c exporterTraceTelemetryConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Address == "" {
		return errors.New("exporter: address is required")
	}

	return nil
}

func (c exporterTraceTelemetryConfig) GetExporter() (sdktrace.SpanExporter, error) {
	if !c.Enabled {
		return nil, errors.New("telemetry: trace: exporter: disabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		c.Address,
		// Note the use of insecure transport here. TLS is recommended in production.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("telemetry: trace: exporter: %w", err)
	}

	exporter, err := otlptracegrpc.New(context.Background(), otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("telemetry: trace: exporter: failed to create: %w", err)
	}

	return exporter, nil
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

	// Environment used for identifying the service environment
	v.SetDefault("environment", "unknown")

	// Log configuration
	v.SetDefault("log.format", "json")
	v.SetDefault("log.level", "info")

	// Telemetry configuration
	flags.String("telemetry-address", ":10000", "Telemetry HTTP server address")
	_ = v.BindPFlag("telemetry.address", flags.Lookup("telemetry-address"))
	v.SetDefault("telemetry.address", ":10000")

	v.SetDefault("telemetry.trace.sampler", "never")

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

	v.SetDefault("dedupe.enabled", false)
	v.SetDefault("dedupe.driver", "memory")

	// Dedupe Memory configuration
	v.SetDefault("dedupe.config.size", 128)

	// Dedupe Redis configuration
	v.SetDefault("dedupe.config.address", "127.0.0.1:6379")
	v.SetDefault("dedupe.config.database", 0)
	v.SetDefault("dedupe.config.username", "")
	v.SetDefault("dedupe.config.password", "")
	v.SetDefault("dedupe.config.expiration", "24h")
	v.SetDefault("dedupe.config.sentinel.enabled", false)
	v.SetDefault("dedupe.config.sentinel.masterName", "")
	v.SetDefault("dedupe.config.tls.enabled", false)
	v.SetDefault("dedupe.config.tls.insecureSkipVerify", false)
}
