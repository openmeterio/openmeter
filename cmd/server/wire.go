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

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ingest"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/registry"
	"github.com/openmeterio/openmeter/openmeter/secret"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	kafkametrics "github.com/openmeterio/openmeter/pkg/kafka/metrics"
)

type Application struct {
	common.GlobalInitializer
	common.Migrator

	App                     app.Service
	AppStripe               appstripe.Service
	AppSandbox              *appsandbox.App
	Customer                customer.Service
	Billing                 billing.Service
	EntClient               *db.Client
	EventPublisher          eventbus.Publisher
	EntitlementRegistry     *registry.Entitlement
	FeatureConnector        feature.FeatureConnector
	IngestCollector         ingest.Collector
	KafkaProducer           *kafka.Producer
	KafkaMetrics            *kafkametrics.Metrics
	Logger                  *slog.Logger
	MeterRepository         meter.Repository
	NamespaceHandlers       []namespace.Handler
	NamespaceManager        *namespace.Manager
	Notification            notification.Service
	Meter                   metric.Meter
	Plan                    plan.Service
	Subscription            common.SubscriptionServiceWithWorkflow
	SubscriptionPlanAdapter plansubscription.Adapter
	RouterHook              func(chi.Router)
	Secret                  secret.Service
	StreamingConnector      streaming.Connector
	TelemetryServer         common.TelemetryServer
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
		common.NewDefaultTextMapPropagator,
		common.NewServerPublisher,
		common.NewTelemetryRouterHook,
		common.Notification,
		common.OpenMeter,
		common.ProductCatalog,
		common.Subscription,
		common.Svix,
		common.Secret,
		common.ServerProvisionTopics,
		common.Telemetry,
		common.WatermillNoPublisher,
		wire.Struct(new(Application), "*"),
	)

	return Application{}, nil, nil
}

func metadata(conf config.Configuration) common.Metadata {
	return common.NewMetadata(conf, version, "backend")
}
