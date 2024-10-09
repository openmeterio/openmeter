package app

import (
	"github.com/google/wire"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/topicresolver"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/streaming/clickhouse_connector"
)

var Config = wire.NewSet(
	wire.FieldsOf(new(config.Configuration), "Aggregation"),
	wire.FieldsOf(new(config.AggregationConfiguration), "ClickHouse"),

	wire.FieldsOf(new(config.Configuration), "Postgres"),

	wire.FieldsOf(new(config.Configuration), "Telemetry"),
	TelemetryConfig,

	wire.FieldsOf(new(config.Configuration), "Meters"),
	wire.FieldsOf(new(config.Configuration), "Namespace"),
)

var TelemetryConfig = wire.NewSet(
	wire.FieldsOf(new(config.TelemetryConfig), "Metrics"),
	wire.FieldsOf(new(config.TelemetryConfig), "Trace"),
	wire.FieldsOf(new(config.TelemetryConfig), "Log"),
)

var ClickHouse = wire.NewSet(
	NewClickHouse,
)

var Database = wire.NewSet(
	NewPostgresDriver,
	NewDB,
	NewEntPostgresDriver,
	NewEntClient,
)

var Kafka = wire.NewSet(
	NewKafkaProducer,
	NewKafkaMetrics,

	wire.FieldsOf(new(config.Configuration), "Ingest"),
	NewKafkaTopicProvisioner,
)

var Telemetry = wire.NewSet(
	NewMeterProvider,
	wire.Bind(new(metric.MeterProvider), new(*sdkmetric.MeterProvider)),
	NewTracerProvider,
	wire.Bind(new(trace.TracerProvider), new(*sdktrace.TracerProvider)),

	NewHealthChecker,

	NewTelemetryHandler,
	NewTelemetryServer,
)

var Logger = wire.NewSet(
	NewLogger,
)

var OpenMeter = wire.NewSet(
	NewMeterRepository,
	wire.Bind(new(meter.Repository), new(*meter.InMemoryRepository)),

	NewClickHouseStreamingConnector,
	wire.Bind(new(streaming.Connector), new(*clickhouse_connector.ClickhouseConnector)),

	NewNamespacedTopicResolver,
	wire.Bind(new(topicresolver.Resolver), new(*topicresolver.NamespacedTopicResolver)),

	NewKafkaIngestCollector,
	NewKafkaNamespaceHandler,
	NewIngestCollector,

	NewNamespaceHandlers,
	NewNamespaceManager,
)

var Watermill = wire.NewSet(
	NewPublisher,
	NewEventBusPublisher,
)
