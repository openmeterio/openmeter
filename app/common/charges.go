package common

import (
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	chargesadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	creditpurchaseadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/adapter"
	creditpurchaseservice "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeeadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/adapter"
	flatfeeservice "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	chargesservice "github.com/openmeterio/openmeter/openmeter/billing/charges/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/adapter"
	usagebasedservice "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	ledgerchargeadapter "github.com/openmeterio/openmeter/openmeter/ledger/chargeadapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

func NewChargesMetaAdapter(
	db *entdb.Client,
	logger *slog.Logger,
) (meta.Adapter, error) {
	metaAdapter, err := metaadapter.New(metaadapter.Config{
		Client: db,
		Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges meta adapter: %w", err)
	}

	return metaAdapter, nil
}

func NewChargesFlatFeeHandler(
	ledgerService ledger.Ledger,
	accountResolver ledger.AccountResolver,
	accountService ledgeraccount.Service,
) flatfee.Handler {
	return ledgerchargeadapter.NewFlatFeeHandler(ledgerService, accountResolver, accountService)
}

func NewChargesCreditPurchaseHandler(
	ledgerService ledger.Ledger,
	accountResolver ledger.AccountResolver,
	accountService ledgeraccount.Service,
) creditpurchase.Handler {
	return ledgerchargeadapter.NewCreditPurchaseHandler(ledgerService, accountResolver, accountService)
}

func NewChargesUsageBasedHandler() usagebased.Handler {
	return usagebased.UnimplementedHandler{}
}

func NewChargesFlatFeeAdapter(
	db *entdb.Client,
	logger *slog.Logger,
	metaAdapter meta.Adapter,
) (flatfee.Adapter, error) {
	flatFeeAdapter, err := flatfeeadapter.New(flatfeeadapter.Config{
		Client:      db,
		Logger:      logger,
		MetaAdapter: metaAdapter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges flat fee adapter: %w", err)
	}

	return flatFeeAdapter, nil
}

func NewChargesFlatFeeService(
	flatFeeAdapter flatfee.Adapter,
	flatFeeHandler flatfee.Handler,
	metaAdapter meta.Adapter,
	locker *lockr.Locker,
) (flatfee.Service, error) {
	flatFeeSvc, err := flatfeeservice.New(flatfeeservice.Config{
		Adapter:     flatFeeAdapter,
		Handler:     flatFeeHandler,
		MetaAdapter: metaAdapter,
		Locker:      locker,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges flat fee service: %w", err)
	}

	return flatFeeSvc, nil
}

func NewChargesUsageBasedAdapter(
	db *entdb.Client,
	logger *slog.Logger,
	metaAdapter meta.Adapter,
) (usagebased.Adapter, error) {
	usageBasedAdapter, err := usagebasedadapter.New(usagebasedadapter.Config{
		Client:      db,
		Logger:      logger,
		MetaAdapter: metaAdapter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges usage based adapter: %w", err)
	}

	return usageBasedAdapter, nil
}

func NewChargesUsageBasedService(
	usageBasedAdapter usagebased.Adapter,
	usageBasedHandler usagebased.Handler,
	locker *lockr.Locker,
	metaAdapter meta.Adapter,
	billingService billing.Service,
	featureService feature.FeatureConnector,
	ratingService rating.Service,
	streamingConnector streaming.Connector,
) (usagebased.Service, error) {
	usageBasedSvc, err := usagebasedservice.New(usagebasedservice.Config{
		Adapter:                 usageBasedAdapter,
		Handler:                 usageBasedHandler,
		Locker:                  locker,
		MetaAdapter:             metaAdapter,
		CustomerOverrideService: billingService,
		FeatureService:          featureService,
		RatingService:           ratingService,
		StreamingConnector:      streamingConnector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges usage based service: %w", err)
	}

	return usageBasedSvc, nil
}

func NewChargesCreditPurchaseAdapter(
	db *entdb.Client,
	logger *slog.Logger,
	metaAdapter meta.Adapter,
) (creditpurchase.Adapter, error) {
	creditPurchaseAdapter, err := creditpurchaseadapter.New(creditpurchaseadapter.Config{
		Client:      db,
		Logger:      logger,
		MetaAdapter: metaAdapter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges credit purchase adapter: %w", err)
	}

	return creditPurchaseAdapter, nil
}

func NewChargesCreditPurchaseService(
	creditPurchaseAdapter creditpurchase.Adapter,
	creditPurchaseHandler creditpurchase.Handler,
	metaAdapter meta.Adapter,
) (creditpurchase.Service, error) {
	creditPurchaseSvc, err := creditpurchaseservice.New(creditpurchaseservice.Config{
		Adapter:     creditPurchaseAdapter,
		Handler:     creditPurchaseHandler,
		MetaAdapter: metaAdapter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges credit purchase service: %w", err)
	}

	return creditPurchaseSvc, nil
}

func NewChargesAdapter(
	db *entdb.Client,
	logger *slog.Logger,
) (charges.Adapter, error) {
	rootAdapter, err := chargesadapter.New(chargesadapter.Config{
		Client: db,
		Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges adapter: %w", err)
	}

	return rootAdapter, nil
}

func NewChargesService(
	rootAdapter charges.Adapter,
	metaAdapter meta.Adapter,
	featureService feature.FeatureConnector,
	flatFeeSvc flatfee.Service,
	creditPurchaseSvc creditpurchase.Service,
	usageBasedSvc usagebased.Service,
	billingService billing.Service,
	fsNamespaceLockdown []string,
) (charges.Service, error) {
	chargesSvc, err := chargesservice.New(chargesservice.Config{
		Adapter:               rootAdapter,
		MetaAdapter:           metaAdapter,
		FeatureService:        featureService,
		FlatFeeService:        flatFeeSvc,
		CreditPurchaseService: creditPurchaseSvc,
		UsageBasedService:     usageBasedSvc,
		BillingService:        billingService,
		FSNamespaceLockdown:   fsNamespaceLockdown,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges service: %w", err)
	}

	return chargesSvc, nil
}

// newChargesRegistry constructs the full charges stack.
// Private: must be initialized via NewBillingRegistry.
func newChargesRegistry(
	db *entdb.Client,
	logger *slog.Logger,
	locker *lockr.Locker,
	billingService billing.Service,
	ratingService rating.Service,
	featureService feature.FeatureConnector,
	streamingConnector streaming.Connector,
	ledgerService ledger.Ledger,
	accountResolver ledger.AccountResolver,
	accountService ledgeraccount.Service,
	fsNamespaceLockdown []string,
) (*ChargesRegistry, error) {
	metaAdapter, err := NewChargesMetaAdapter(db, logger)
	if err != nil {
		return nil, err
	}

	flatFeeHandler := NewChargesFlatFeeHandler(ledgerService, accountResolver, accountService)
	creditPurchaseHandler := NewChargesCreditPurchaseHandler(ledgerService, accountResolver, accountService)
	usageBasedHandler := NewChargesUsageBasedHandler()

	flatFeeAdapter, err := NewChargesFlatFeeAdapter(db, logger, metaAdapter)
	if err != nil {
		return nil, err
	}

	flatFeeSvc, err := NewChargesFlatFeeService(flatFeeAdapter, flatFeeHandler, metaAdapter, locker)
	if err != nil {
		return nil, err
	}

	usageBasedAdapter, err := NewChargesUsageBasedAdapter(db, logger, metaAdapter)
	if err != nil {
		return nil, err
	}

	usageBasedSvc, err := NewChargesUsageBasedService(
		usageBasedAdapter,
		usageBasedHandler,
		locker,
		metaAdapter,
		billingService,
		featureService,
		ratingService,
		streamingConnector,
	)
	if err != nil {
		return nil, err
	}

	creditPurchaseAdapter, err := NewChargesCreditPurchaseAdapter(db, logger, metaAdapter)
	if err != nil {
		return nil, err
	}

	creditPurchaseSvc, err := NewChargesCreditPurchaseService(creditPurchaseAdapter, creditPurchaseHandler, metaAdapter)
	if err != nil {
		return nil, err
	}

	rootAdapter, err := NewChargesAdapter(db, logger)
	if err != nil {
		return nil, err
	}

	chargesSvc, err := NewChargesService(
		rootAdapter,
		metaAdapter,
		featureService,
		flatFeeSvc,
		creditPurchaseSvc,
		usageBasedSvc,
		billingService,
		fsNamespaceLockdown,
	)
	if err != nil {
		return nil, err
	}

	return &ChargesRegistry{
		Service:               chargesSvc,
		FlatFeeService:        flatFeeSvc,
		UsageBasedService:     usageBasedSvc,
		CreditPurchaseService: creditPurchaseSvc,
	}, nil
}
