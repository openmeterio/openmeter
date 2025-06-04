package registrybuilder

import (
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/credit"
	creditpgadapter "github.com/openmeterio/openmeter/openmeter/credit/adapter"
	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	entitlementpgadapter "github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
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
	featureConnector := feature.NewFeatureConnector(featureDBAdapter, opts.MeterService, opts.Publisher)
	entitlementOwnerConnector := meteredentitlement.NewEntitlementGrantOwnerAdapter(
		featureDBAdapter,
		entitlementDBAdapter,
		usageResetDBAdapter,
		opts.MeterService,
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
	entitlementConnector := entitlementservice.NewEntitlementConnector(
		entitlementDBAdapter,
		featureConnector,
		opts.MeterService,
		meteredEntitlementConnector,
		staticentitlement.NewStaticEntitlementConnector(),
		booleanentitlement.NewBooleanEntitlementConnector(),
		opts.Publisher,
		opts.Locker,
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
