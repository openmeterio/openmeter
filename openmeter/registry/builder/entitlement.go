package registrybuilder

import (
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/credit"
	creditpgadapter "github.com/openmeterio/openmeter/openmeter/credit/adapter"
	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	credithook "github.com/openmeterio/openmeter/openmeter/credit/hook"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	entitlementpgadapter "github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	entitlementsubscriptionhook "github.com/openmeterio/openmeter/openmeter/entitlement/hooks/subscription"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	entitlementservice "github.com/openmeterio/openmeter/openmeter/entitlement/service"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
	"github.com/openmeterio/openmeter/openmeter/meter"
	productcatalogpgadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/registry"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

type EntitlementOptions struct {
	DatabaseClient            *db.Client
	EntitlementsConfiguration config.EntitlementsConfiguration
	StreamingConnector        streaming.Connector
	Logger                    *slog.Logger
	MeterService              meter.Service
	CustomerService           customer.Service
	Publisher                 eventbus.Publisher
	Tracer                    trace.Tracer
	Locker                    *lockr.Locker
}

func GetEntitlementRegistry(opts EntitlementOptions) *registry.Entitlement {
	// Initialize database adapters
	featureDBAdapter := productcatalogpgadapter.NewPostgresFeatureRepo(opts.DatabaseClient, opts.Logger)
	entitlementDBAdapter := entitlementpgadapter.NewPostgresEntitlementRepo(opts.DatabaseClient)
	usageResetDBAdapter := entitlementpgadapter.NewPostgresUsageResetRepo(opts.DatabaseClient)
	grantDBAdapter := creditpgadapter.NewPostgresGrantRepo(opts.DatabaseClient)
	balanceSnashotDBAdapter := creditpgadapter.NewPostgresBalanceSnapshotRepo(opts.DatabaseClient)

	// Initialize connectors
	featureConnector := feature.NewFeatureConnector(featureDBAdapter, opts.MeterService, opts.Publisher, nil)
	entitlementOwnerConnector := meteredentitlement.NewEntitlementGrantOwnerAdapter(
		featureDBAdapter,
		entitlementDBAdapter,
		usageResetDBAdapter,
		opts.MeterService,
		opts.CustomerService,
		opts.Logger,
		opts.Tracer,
	)
	transactionManager := enttx.NewCreator(opts.DatabaseClient)

	balanceSnapshotService := balance.NewSnapshotService(balance.SnapshotServiceConfig{
		OwnerConnector:     entitlementOwnerConnector,
		StreamingConnector: opts.StreamingConnector,
		Repo:               balanceSnashotDBAdapter,
	})

	creditConnector := credit.NewCreditConnector(
		credit.CreditConnectorConfig{
			GrantRepo:              grantDBAdapter,
			BalanceSnapshotService: balanceSnapshotService,
			OwnerConnector:         entitlementOwnerConnector,
			StreamingConnector:     opts.StreamingConnector,
			Logger:                 opts.Logger,
			Tracer:                 opts.Tracer,
			Granularity:            time.Minute,
			SnapshotGracePeriod:    opts.EntitlementsConfiguration.GetGracePeriod(),
			TransactionManager:     transactionManager,
			Publisher:              opts.Publisher,
		},
	)
	creditBalanceConnector := creditConnector
	grantConnector := creditConnector
	meteredEntitlementConnector := meteredentitlement.NewMeteredEntitlementConnector(
		opts.StreamingConnector,
		entitlementOwnerConnector,
		creditBalanceConnector,
		grantConnector,
		grantDBAdapter,
		entitlementDBAdapter,
		opts.Publisher,
		opts.Logger,
		opts.Tracer,
	)

	meteredEntitlementConnector.RegisterHooks(
		meteredentitlement.ConvertHook(entitlementsubscriptionhook.NewEntitlementSubscriptionHook(entitlementsubscriptionhook.EntitlementSubscriptionHookConfig{})),
	)

	entitlementConnector := entitlementservice.NewEntitlementService(
		entitlementservice.ServiceConfig{
			EntitlementRepo:             entitlementDBAdapter,
			FeatureConnector:            featureConnector,
			CustomerService:             opts.CustomerService,
			MeterService:                opts.MeterService,
			MeteredEntitlementConnector: meteredEntitlementConnector,
			StaticEntitlementConnector:  staticentitlement.NewStaticEntitlementConnector(),
			BooleanEntitlementConnector: booleanentitlement.NewBooleanEntitlementConnector(),
			Publisher:                   opts.Publisher,
			Locker:                      opts.Locker,
		},
	)

	entitlementConnector.RegisterHooks(
		entitlementsubscriptionhook.NewEntitlementSubscriptionHook(entitlementsubscriptionhook.EntitlementSubscriptionHookConfig{}),
		credithook.NewEntitlementHook(grantDBAdapter),
	)

	return &registry.Entitlement{
		Feature:            featureConnector,
		FeatureRepo:        featureDBAdapter,
		EntitlementOwner:   entitlementOwnerConnector,
		CreditBalance:      creditBalanceConnector,
		Grant:              grantConnector,
		GrantRepo:          grantDBAdapter,
		MeteredEntitlement: meteredEntitlementConnector,
		Entitlement:        entitlementConnector,
		EntitlementRepo:    entitlementDBAdapter,
	}
}
