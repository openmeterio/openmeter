package common

import (
	"log/slog"

	"github.com/google/wire"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingadapter "github.com/openmeterio/openmeter/openmeter/billing/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	billingratingservice "github.com/openmeterio/openmeter/openmeter/billing/rating/service"
	billingservice "github.com/openmeterio/openmeter/openmeter/billing/service"
	billingcustomer "github.com/openmeterio/openmeter/openmeter/billing/validators/customer"
	billingsubscription "github.com/openmeterio/openmeter/openmeter/billing/validators/subscription"
	billingworkerautoadvance "github.com/openmeterio/openmeter/openmeter/billing/worker/advance"
	billingworkercollect "github.com/openmeterio/openmeter/openmeter/billing/worker/collect"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	subscriptionsyncadapter "github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/reconciler"
	subscriptionsyncservice "github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

// BillingRegistry bundles the billing and charges services. External callers that need
// billing or charges should depend on BillingRegistry rather than individual services.
type BillingRegistry struct {
	Billing billing.Service
	Charges *ChargesRegistry
}

func (r BillingRegistry) ChargesServiceOrNil() charges.Service {
	if r.Charges == nil {
		return nil
	}

	return r.Charges.Service
}

// ChargesRegistry groups all charge-type services.
type ChargesRegistry struct {
	Service               charges.Service
	FlatFeeService        flatfee.Service
	UsageBasedService     usagebased.Service
	CreditPurchaseService creditpurchase.Service
}

// Billing is the Wire provider set for the billing and charges stack.
// Downstream consumers should depend on BillingRegistry.
var Billing = wire.NewSet(
	BillingAdapter,
	NewBillingRatingService,
	NewBillingRegistry,
	NewBillingCustomerOverrideService,
)

func BillingAdapter(
	logger *slog.Logger,
	db *entdb.Client,
) (billing.Adapter, error) {
	return billingadapter.New(billingadapter.Config{
		Client: db,
		Logger: logger,
	})
}

// newBillingService creates the billing service and registers validators/hooks.
// Downstream consumers should use BillingRegistry.
func newBillingService(
	logger *slog.Logger,
	appService app.Service,
	billingAdapter billing.Adapter,
	billingRatingService rating.Service,
	customerService customer.Service,
	featureConnector feature.FeatureConnector,
	meterService meter.Service,
	streamingConnector streaming.Connector,
	eventPublisher eventbus.Publisher,
	billingConfig config.BillingConfiguration,
	subscriptionServices SubscriptionServiceWithWorkflow,
	db *entdb.Client,
	fsConfig config.BillingFeatureSwitchesConfiguration,
	tracer trace.Tracer,
	taxCodeService taxcode.Service,
) (billing.Service, error) {
	service, err := billingservice.New(billingservice.Config{
		Adapter:                      billingAdapter,
		RatingService:                billingRatingService,
		AppService:                   appService,
		CustomerService:              customerService,
		TaxCodeService:               taxCodeService,
		FeatureService:               featureConnector,
		Logger:                       logger,
		MeterService:                 meterService,
		StreamingConnector:           streamingConnector,
		Publisher:                    eventPublisher,
		AdvancementStrategy:          billingConfig.AdvancementStrategy,
		FSNamespaceLockdown:          fsConfig.NamespaceLockdown,
		MaxParallelQuantitySnapshots: billingConfig.MaxParallelQuantitySnapshots,
	})
	if err != nil {
		return nil, err
	}

	return service, nil
}

// NewBillingRegistry assembles the billing and optional charges services.
func NewBillingRegistry(
	logger *slog.Logger,
	appService app.Service,
	billingAdapter billing.Adapter,
	billingRatingService rating.Service,
	customerService customer.Service,
	featureConnector feature.FeatureConnector,
	meterService meter.Service,
	streamingConnector streaming.Connector,
	eventPublisher eventbus.Publisher,
	billingConfig config.BillingConfiguration,
	subscriptionServices SubscriptionServiceWithWorkflow,
	db *entdb.Client,
	fsConfig config.BillingFeatureSwitchesConfiguration,
	creditsConfig config.CreditsConfiguration,
	tracer trace.Tracer,
	taxCodeService taxcode.Service,
	locker *lockr.Locker,
	ledgerService ledger.Ledger,
	accountResolver ledger.AccountResolver,
	accountService ledgeraccount.Service,
) (BillingRegistry, error) {
	billingService, err := newBillingService(
		logger,
		appService,
		billingAdapter,
		billingRatingService,
		customerService,
		featureConnector,
		meterService,
		streamingConnector,
		eventPublisher,
		billingConfig,
		subscriptionServices,
		db,
		fsConfig,
		tracer,
		taxCodeService,
	)
	if err != nil {
		return BillingRegistry{}, err
	}

	var chargesRegistry *ChargesRegistry

	if creditsConfig.Enabled {
		chargesRegistry, err = newChargesRegistry(
			db,
			logger,
			locker,
			billingService,
			billingRatingService,
			featureConnector,
			streamingConnector,
			ledgerService,
			accountResolver,
			accountService,
		)
		if err != nil {
			return BillingRegistry{}, err
		}
	}

	billingRegistry := BillingRegistry{
		Billing: billingService,
		Charges: chargesRegistry,
	}

	// Hook registration

	// Customer validate (and sync subscription on delete)
	// To prevent circular dependencies, we create the validator here
	subscriptionSyncAdapter, err := NewBillingSubscriptionSyncAdapter(db)
	if err != nil {
		return BillingRegistry{}, err
	}
	subscriptionSyncService, err := NewBillingSubscriptionSyncService(logger, subscriptionServices, billingRegistry, subscriptionSyncAdapter, tracer)
	if err != nil {
		return BillingRegistry{}, err
	}

	validator, err := billingcustomer.NewValidator(billingRegistry.Billing, subscriptionSyncService, subscriptionServices.Service)
	if err != nil {
		return BillingRegistry{}, err
	}

	customerService.RegisterRequestValidator(validator)

	// Subscription validate

	subscriptionValidator, err := billingsubscription.NewValidator(billingRegistry.Billing)
	if err != nil {
		return BillingRegistry{}, err
	}

	if err = subscriptionServices.Service.RegisterHook(subscriptionValidator); err != nil {
		return BillingRegistry{}, err
	}

	return billingRegistry, nil
}

func NewBillingCustomerOverrideService(billingRegistry BillingRegistry) billing.CustomerOverrideService {
	return billingRegistry.Billing
}

func NewBillingRatingService() rating.Service {
	return billingratingservice.New()
}

func NewBillingAutoAdvancer(logger *slog.Logger, billingRegistry BillingRegistry) (*billingworkerautoadvance.AutoAdvancer, error) {
	return billingworkerautoadvance.NewAdvancer(billingworkerautoadvance.Config{
		BillingService: billingRegistry.Billing,
		Logger:         logger,
	})
}

func NewBillingCollector(logger *slog.Logger, billingRegistry BillingRegistry, fs config.BillingFeatureSwitchesConfiguration) (*billingworkercollect.InvoiceCollector, error) {
	return billingworkercollect.NewInvoiceCollector(billingworkercollect.Config{
		BillingService:   billingRegistry.Billing,
		Logger:           logger,
		LockedNamespaces: fs.NamespaceLockdown,
	})
}

func NewBillingSubscriptionReconciler(logger *slog.Logger, subsServices SubscriptionServiceWithWorkflow, subscriptionSync subscriptionsync.Service, customerService customer.Service) (*reconciler.Reconciler, error) {
	return reconciler.NewReconciler(reconciler.ReconcilerConfig{
		SubscriptionService: subsServices.Service,
		SubscriptionSync:    subscriptionSync,
		Logger:              logger,
		CustomerService:     customerService,
	})
}

func NewBillingSubscriptionSyncAdapter(db *entdb.Client) (subscriptionsync.Adapter, error) {
	return subscriptionsyncadapter.New(subscriptionsyncadapter.Config{
		Client: db,
	})
}

func NewBillingSubscriptionSyncService(logger *slog.Logger, subsServices SubscriptionServiceWithWorkflow, billingRegistry BillingRegistry, subscriptionSyncAdapter subscriptionsync.Adapter, tracer trace.Tracer) (subscriptionsync.Service, error) {
	return subscriptionsyncservice.New(subscriptionsyncservice.Config{
		SubscriptionService:     subsServices.Service,
		BillingService:          billingRegistry.Billing,
		ChargesService:          billingRegistry.ChargesServiceOrNil(),
		SubscriptionSyncAdapter: subscriptionSyncAdapter,
		Logger:                  logger,
		Tracer:                  tracer,
	})
}
