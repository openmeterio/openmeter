package registrybuilder

import (
	"time"

	"github.com/openmeterio/openmeter/openmeter/credit"
	creditpgadapter "github.com/openmeterio/openmeter/openmeter/credit/adapter"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementpgadapter "github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcatalogpgadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/registry"
)

func GetEntitlementRegistry(opts registry.EntitlementOptions) *registry.Entitlement {
	// Initialize database adapters
	featureDBAdapter := productcatalogpgadapter.NewPostgresFeatureRepo(opts.DatabaseClient, opts.Logger)
	entitlementDBAdapter := entitlementpgadapter.NewPostgresEntitlementRepo(opts.DatabaseClient)
	usageResetDBAdapter := entitlementpgadapter.NewPostgresUsageResetRepo(opts.DatabaseClient)
	grantDBAdapter := creditpgadapter.NewPostgresGrantRepo(opts.DatabaseClient)
	balanceSnashotDBAdapter := creditpgadapter.NewPostgresBalanceSnapshotRepo(opts.DatabaseClient)

	// Initialize connectors
	featureConnector := productcatalog.NewFeatureConnector(featureDBAdapter, opts.MeterRepository)
	entitlementOwnerConnector := meteredentitlement.NewEntitlementGrantOwnerAdapter(
		featureDBAdapter,
		entitlementDBAdapter,
		usageResetDBAdapter,
		opts.MeterRepository,
		opts.Logger,
	)
	creditConnector := credit.NewCreditConnector(
		grantDBAdapter,
		balanceSnashotDBAdapter,
		entitlementOwnerConnector,
		opts.StreamingConnector,
		opts.Logger,
		time.Minute,
		opts.Publisher,
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
