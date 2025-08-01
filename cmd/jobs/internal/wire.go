//go:build wireinject
// +build wireinject

package internal

import (
	"context"
	"log/slog"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/wire"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingworkerautoadvance "github.com/openmeterio/openmeter/openmeter/billing/worker/advance"
	billingworkercollect "github.com/openmeterio/openmeter/openmeter/billing/worker/collect"
	billingworkersubscription "github.com/openmeterio/openmeter/openmeter/billing/worker/subscription"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/registry"
	"github.com/openmeterio/openmeter/openmeter/secret"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	kafkametrics "github.com/openmeterio/openmeter/pkg/kafka/metrics"
)

type Application struct {
	common.GlobalInitializer
	common.Migrator

	App                           app.Service
	AppStripe                     appstripe.Service
	AppSandboxProvisioner         common.AppSandboxProvisioner
	Customer                      customer.Service
	Billing                       billing.Service
	BillingAutoAdvancer           *billingworkerautoadvance.AutoAdvancer
	BillingCollector              *billingworkercollect.InvoiceCollector
	BillingSubscriptionReconciler *billingworkersubscription.Reconciler
	EntClient                     *db.Client
	EventPublisher                eventbus.Publisher
	EntitlementRegistry           *registry.Entitlement
	FeatureConnector              feature.FeatureConnector
	KafkaProducer                 *kafka.Producer
	KafkaMetrics                  *kafkametrics.Metrics
	Logger                        *slog.Logger
	MeterService                  meter.Service
	NamespaceManager              *namespace.Manager
	Meter                         metric.Meter
	NotificationService           notification.Service
	Plan                          plan.Service
	Secret                        secret.Service
	Subscription                  common.SubscriptionServiceWithWorkflow
	StreamingConnector            streaming.Connector
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
		common.Meter,
		common.Namespace,
		common.NewBillingAutoAdvancer,
		common.NewBillingCollector,
		common.NewBillingSubscriptionHandler,
		common.NewBillingSubscriptionReconciler,
		common.NewDefaultTextMapPropagator,
		common.NewServerPublisher,
		common.Notification,
		common.Streaming,
		common.ProductCatalog,
		common.ProgressManager,
		common.Subscription,
		common.Lockr,
		common.Secret,
		common.ServerProvisionTopics,
		common.TelemetryWithoutServer,
		common.WatermillNoPublisher,
		wire.Struct(new(Application), "*"),
	)

	return Application{}, nil, nil
}

func metadata(conf config.Configuration) common.Metadata {
	return common.NewMetadata(conf, version, "jobs")
}
