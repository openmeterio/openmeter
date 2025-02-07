package registrybuilder

import (
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/openmeter/credit"
	creditpgadapter "github.com/openmeterio/openmeter/openmeter/credit/adapter"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementpgadapter "github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
	"github.com/openmeterio/openmeter/openmeter/meter"
	productcatalogpgadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/registry"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

type EntitlementOptions struct {
	DatabaseClient     *db.Client
	StreamingConnector streaming.Connector
	Logger             *slog.Logger
	MeterRepository    meter.Repository
	Publisher          eventbus.Publisher
}

func GetEntitlementRegistry(opts EntitlementOptions) *registry.Entitlement {
	// Initialize database adapters
	featureDBAdapter := productcatalogpgadapter.NewPostgresFeatureRepo(opts.DatabaseClient, opts.Logger)
	entitlementDBAdapter := entitlementpgadapter.NewPostgresEntitlementRepo(opts.DatabaseClient)
	usageResetDBAdapter := entitlementpgadapter.NewPostgresUsageResetRepo(opts.DatabaseClient)
	grantDBAdapter := creditpgadapter.NewPostgresGrantRepo(opts.DatabaseClient)
	balanceSnashotDBAdapter := creditpgadapter.NewPostgresBalanceSnapshotRepo(opts.DatabaseClient)

	// Initialize connectors
	featureConnector := feature.NewFeatureConnector(featureDBAdapter, opts.MeterRepository)
	entitlementOwnerConnector := meteredentitlement.NewEntitlementGrantOwnerAdapter(
		featureDBAdapter,
		entitlementDBAdapter,
		usageResetDBAdapter,
		opts.MeterRepository,
		opts.Logger,
	)
	transactionManager := enttx.NewCreator(opts.DatabaseClient)

	creditConnector := credit.NewCreditConnector(
		grantDBAdapter,
		balanceSnashotDBAdapter,
		entitlementOwnerConnector,
		opts.StreamingConnector,
		opts.Logger,
		time.Minute,
		opts.Publisher,
		transactionManager,
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
	)
	entitlementConnector := entitlement.NewEntitlementConnector(
		entitlementDBAdapter,
		featureConnector,
		opts.MeterRepository,
		meteredEntitlementConnector,
		staticentitlement.NewStaticEntitlementConnector(),
		booleanentitlement.NewBooleanEntitlementConnector(),
		opts.Publisher,
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
