package common

import (
	"github.com/google/wire"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementadapter "github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	"github.com/openmeterio/openmeter/openmeter/entitlement/balanceworker"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/topicresolver"
	"github.com/openmeterio/openmeter/openmeter/meter"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/streaming/clickhouse_connector"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
)

var Config = wire.NewSet(
	wire.FieldsOf(new(config.Configuration), "Aggregation"),
	wire.FieldsOf(new(config.AggregationConfiguration), "ClickHouse"),

	wire.FieldsOf(new(config.Configuration), "Postgres"),

	wire.FieldsOf(new(config.Configuration), "Telemetry"),
	TelemetryConfig,

	wire.FieldsOf(new(config.Configuration), "Ingest"),
	wire.FieldsOf(new(config.IngestConfiguration), "Kafka"),
	wire.FieldsOf(new(config.KafkaIngestConfiguration), "KafkaConfiguration"),
	wire.FieldsOf(new(config.KafkaIngestConfiguration), "TopicProvisionerConfig"),

	wire.FieldsOf(new(config.Configuration), "Meters"),
	wire.FieldsOf(new(config.Configuration), "Namespace"),
	wire.FieldsOf(new(config.Configuration), "Events"),
	wire.FieldsOf(new(config.Configuration), "BalanceWorker"),
	wire.FieldsOf(new(config.Configuration), "Notification"),
	wire.FieldsOf(new(config.Configuration), "Sink"),
)

var TelemetryConfig = wire.NewSet(
	wire.FieldsOf(new(config.TelemetryConfig), "Metrics"),
	wire.FieldsOf(new(config.TelemetryConfig), "Trace"),
	wire.FieldsOf(new(config.TelemetryConfig), "Log"),
)

var Framework = wire.NewSet(
	wire.Struct(new(GlobalInitializer), "*"),
	wire.Struct(new(Runner), "*"),
)

var ClickHouse = wire.NewSet(
	NewClickHouse,
)

var Database = wire.NewSet(
	NewPostgresDriver,
	NewDB,
	NewEntPostgresDriver,
	NewEntClient,
	wire.Struct(new(Migrator), "*"),
)

var Kafka = wire.NewSet(
	NewKafkaProducer,
	NewKafkaMetrics,

	KafkaTopic,
)

var KafkaTopic = wire.NewSet(
	NewKafkaAdminClient,

	NewKafkaTopicProvisionerConfig,
	NewKafkaTopicProvisioner,
)

var Telemetry = wire.NewSet(
	NewTelemetryResource,

	NewMeterProvider,
	wire.Bind(new(metric.MeterProvider), new(*sdkmetric.MeterProvider)),
	NewMeter,
	NewTracerProvider,
	wire.Bind(new(trace.TracerProvider), new(*sdktrace.TracerProvider)),
	NewTracer,

	NewHealthChecker,

	NewTelemetryHandler,
	NewTelemetryServer,
)

var Logger = wire.NewSet(
	NewTelemetryResource,
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

var Entitlements = wire.NewSet(
	wire.Struct(new(registrybuilder.EntitlementOptions), "*"),
	registrybuilder.GetEntitlementRegistry,
)

var EntitlementsAdapter = wire.NewSet(
	NewEntitlementRepo,
)

type EntitlementRepo interface {
	entitlement.EntitlementRepo
	balanceworker.BalanceWorkerRepository
}

// TODO: export the underlying type
func NewEntitlementRepo(db *db.Client) EntitlementRepo {
	return entitlementadapter.NewPostgresEntitlementRepo(db)
}

var Watermill = wire.NewSet(
	WatermillNoPublisher,

	// NewBrokerConfiguration,
	// wire.Struct(new(watermillkafka.PublisherOptions), "*"),

	NewPublisher,
	// NewEventBusPublisher,
)

// TODO: move this back to [Watermill]
// NOTE: this is also used by the sink-worker that requires control over how the publisher is closed
var WatermillNoPublisher = wire.NewSet(
	NewBrokerConfiguration,
	wire.Struct(new(watermillkafka.PublisherOptions), "*"),

	NewEventBusPublisher,
)

var WatermillRouter = wire.NewSet(
	wire.Struct(new(router.Options), "*"),
)

var BalanceWorkerAdapter = wire.NewSet(
	EntitlementsAdapter,

	wire.Bind(new(balanceworker.BalanceWorkerRepository), new(EntitlementRepo)),
	SubjectResolver,
)

func SubjectResolver() balanceworker.SubjectResolver {
	return nil
}

var BalanceWorker = wire.NewSet(
	wire.FieldsOf(new(config.BalanceWorkerConfiguration), "ConsumerConfiguration"),

	BalanceWorkerProvisionTopics,
	BalanceWorkerSubscriber,

	Entitlements,

	NewBalanceWorkerOptions,
	NewBalanceWorker,
	BalanceWorkerGroup,
)
