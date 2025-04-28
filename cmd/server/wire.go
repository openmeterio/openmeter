//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"log/slog"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/go-chi/chi/v5"
	"github.com/google/wire"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ingest"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/portal"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/openmeter/progressmanager"
	"github.com/openmeterio/openmeter/openmeter/registry"
	"github.com/openmeterio/openmeter/openmeter/secret"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	kafkametrics "github.com/openmeterio/openmeter/pkg/kafka/metrics"
)

type Application struct {
	common.GlobalInitializer
	common.Migrator

	Addon                   addon.Service
	AppRegistry             common.AppRegistry
	Customer                customer.Service
	Billing                 billing.Service
	EntClient               *db.Client
	EventPublisher          eventbus.Publisher
	EntitlementRegistry     *registry.Entitlement
	FeatureConnector        feature.FeatureConnector
	IngestCollector         ingest.Collector
	IngestService           ingest.Service
	KafkaProducer           *kafka.Producer
	KafkaMetrics            *kafkametrics.Metrics
	Logger                  *slog.Logger
	MetricMeter             metric.Meter
	MeterConfigInitializer  common.MeterConfigInitializer
	MeterManageService      meter.ManageService
	MeterEventService       meterevent.Service
	NamespaceHandlers       []namespace.Handler
	NamespaceManager        *namespace.Manager
	Notification            notification.Service
	Plan                    plan.Service
	PlanAddon               planaddon.Service
	Portal                  portal.Service
	ProgressManager         progressmanager.Service
	RouterHook              func(chi.Router)
	Secret                  secret.Service
	Subscription            common.SubscriptionServiceWithWorkflow
	StreamingConnector      streaming.Connector
	TelemetryServer         common.TelemetryServer
	TerminationChecker      *common.TerminationChecker
	RuntimeMetricsCollector common.RuntimeMetricsCollector
	Tracer                  trace.Tracer
}

func initializeApplication(ctx context.Context, conf config.Configuration) (Application, func(), error) {
	wire.Build(
		metadata,
		common.App,
		common.Billing,
		common.ClickHouse,
		common.Config,
		common.Customer,
		common.Database,
		common.Entitlement,
		common.Framework,
		common.Kafka,
		common.KafkaNamespaceResolver,
		common.MeterManageWithConfigMeters,
		common.MeterEvent,
		common.Namespace,
		common.NewDefaultTextMapPropagator,
		common.NewKafkaIngestCollector,
		common.NewIngestCollector,
		common.NewIngestService,
		common.NewServerPublisher,
		common.NewTelemetryRouterHook,
		common.Notification,
		common.Streaming,
		common.Portal,
		common.ProductCatalog,
		common.ProgressManager,
		common.Subscription,
		common.Secret,
		common.ServerProvisionTopics,
		common.Telemetry,
		common.NewTerminationChecker,
		common.WatermillNoPublisher,
		wire.Struct(new(Application), "*"),
	)

	return Application{}, nil, nil
}

func metadata(conf config.Configuration) common.Metadata {
	return common.NewMetadata(conf, version, "backend")
}
