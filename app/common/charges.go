package common

import (
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	chargesadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	creditpurchaseadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/adapter"
	creditpurchaselineengine "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/lineengine"
	creditpurchaseservice "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeeadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/adapter"
	flatfeeservice "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	lineageadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/adapter"
	lineageservice "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/service"
	chargeslinerouter "github.com/openmeterio/openmeter/openmeter/billing/charges/linerouter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	chargesservice "github.com/openmeterio/openmeter/openmeter/billing/charges/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/adapter"
	usagebasedservice "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	ledgerbreakage "github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	ledgerbreakageadapter "github.com/openmeterio/openmeter/openmeter/ledger/breakage/adapter"
	ledgerchargeadapter "github.com/openmeterio/openmeter/openmeter/ledger/chargeadapter"
	ledgercollector "github.com/openmeterio/openmeter/openmeter/ledger/collector"
	"github.com/openmeterio/openmeter/openmeter/ledger/recognizer"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/featuregate"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
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

func NewChargesCollectorService(
	db *entdb.Client,
	ledgerService ledger.Ledger,
	balanceQuerier ledger.BalanceQuerier,
	accountResolver ledger.AccountResolver,
	accountService ledgeraccount.Service,
) (ledgercollector.Service, error) {
	collectorService, err := ledgercollector.NewService(ledgercollector.Config{
		Ledger: ledgerService,
		Dependencies: transactions.ResolverDependencies{
			AccountService: accountResolver,
			AccountCatalog: accountService,
			BalanceQuerier: balanceQuerier,
		},
		AccountLocker:      accountService,
		TransactionManager: enttx.NewCreator(db),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges collector service: %w", err)
	}

	return collectorService, nil
}

func NewLedgerBreakageService(
	creditsConfig config.CreditsConfiguration,
	db *entdb.Client,
	balanceQuerier ledger.BalanceQuerier,
	accountResolver ledger.AccountResolver,
	accountService ledgeraccount.Service,
) (ledgerbreakage.Service, error) {
	if !creditsConfig.Enabled {
		return ledgerbreakage.NewNoopService(), nil
	}

	breakageAdapter, err := ledgerbreakageadapter.New(ledgerbreakageadapter.Config{
		Client: db,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ledger breakage adapter: %w", err)
	}

	breakageService, err := ledgerbreakage.NewService(ledgerbreakage.Config{
		Adapter: breakageAdapter,
		Dependencies: transactions.ResolverDependencies{
			AccountService: accountResolver,
			AccountCatalog: accountService,
			BalanceQuerier: balanceQuerier,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ledger breakage service: %w", err)
	}

	return breakageService, nil
}

func NewChargesFlatFeeHandler(
	ledgerService ledger.Ledger,
	balanceQuerier ledger.BalanceQuerier,
	accountResolver ledger.AccountResolver,
	accountService ledgeraccount.Service,
	collectorService ledgercollector.Service,
) flatfee.Handler {
	return ledgerchargeadapter.NewFlatFeeHandler(ledgerService, transactions.ResolverDependencies{
		AccountService: accountResolver,
		AccountCatalog: accountService,
		BalanceQuerier: balanceQuerier,
	}, collectorService)
}

func NewChargesCreditPurchaseHandler(
	ledgerService ledger.Ledger,
	balanceQuerier ledger.BalanceQuerier,
	accountResolver ledger.AccountResolver,
	accountService ledgeraccount.Service,
	breakageService ledgerbreakage.Service,
	transactionManager transaction.Creator,
) (creditpurchase.Handler, error) {
	handler, err := ledgerchargeadapter.NewCreditPurchaseHandler(ledgerService, balanceQuerier, accountResolver, accountService, breakageService, transactionManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create charges credit purchase handler: %w", err)
	}

	return handler, nil
}

func NewChargesUsageBasedHandler(
	ledgerService ledger.Ledger,
	balanceQuerier ledger.BalanceQuerier,
	accountResolver ledger.AccountResolver,
	accountService ledgeraccount.Service,
	collectorService ledgercollector.Service,
) usagebased.Handler {
	return ledgerchargeadapter.NewUsageBasedHandler(ledgerService, transactions.ResolverDependencies{
		AccountService: accountResolver,
		AccountCatalog: accountService,
		BalanceQuerier: balanceQuerier,
	}, collectorService)
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

func NewChargesLineageAdapter(
	db *entdb.Client,
) (lineage.Adapter, error) {
	lineageAdapter, err := lineageadapter.New(lineageadapter.Config{
		Client: db,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges lineage adapter: %w", err)
	}

	return lineageAdapter, nil
}

func NewChargesLineageService(
	lineageAdapter lineage.Adapter,
) (lineage.Service, error) {
	lineageService, err := lineageservice.New(lineageservice.Config{
		Adapter: lineageAdapter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges lineage service: %w", err)
	}

	return lineageService, nil
}

func NewChargesFlatFeeService(
	flatFeeAdapter flatfee.Adapter,
	flatFeeHandler flatfee.Handler,
	lineageService lineage.Service,
	metaAdapter meta.Adapter,
	locker *lockr.Locker,
	ratingService rating.Service,
	currenciesService currencies.Service,
	creditsConfig config.CreditsConfiguration,
) (flatfee.Service, error) {
	flatFeeSvc, err := flatfeeservice.New(flatfeeservice.Config{
		Adapter:                 flatFeeAdapter,
		Handler:                 flatFeeHandler,
		Lineage:                 lineageService,
		MetaAdapter:             metaAdapter,
		Locker:                  locker,
		RatingService:           ratingService,
		Currencies:              currenciesService,
		CustomCurrenciesEnabled: creditsConfig.CustomCurrenciesEnabled,
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
	lineageService lineage.Service,
	locker *lockr.Locker,
	metaAdapter meta.Adapter,
	invoiceUpdater invoiceupdater.Updater,
	billingService billing.Service,
	featureService feature.FeatureConnector,
	ratingService rating.Service,
	currenciesService currencies.Service,
	streamingConnector streaming.Connector,
	creditsConfig config.CreditsConfiguration,
) (usagebased.Service, error) {
	usageBasedSvc, err := usagebasedservice.New(usagebasedservice.Config{
		Adapter:                 usageBasedAdapter,
		Handler:                 usageBasedHandler,
		Lineage:                 lineageService,
		Locker:                  locker,
		MetaAdapter:             metaAdapter,
		InvoiceUpdater:          invoiceUpdater,
		CustomerOverrideService: billingService,
		FeatureService:          featureService,
		RatingService:           ratingService,
		Currencies:              currenciesService,
		StreamingConnector:      streamingConnector,
		CustomCurrenciesEnabled: creditsConfig.CustomCurrenciesEnabled,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges usage based service: %w", err)
	}

	return usageBasedSvc, nil
}

func NewChargesInvoiceUpdater(
	billingService billing.Service,
	logger *slog.Logger,
) (invoiceupdater.Updater, error) {
	updater, err := invoiceupdater.New(invoiceupdater.Config{
		BillingService: billingService,
		Logger:         logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges invoice updater: %w", err)
	}

	return updater, nil
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
	lineageService lineage.Service,
	metaAdapter meta.Adapter,
	creditsConfig config.CreditsConfiguration,
) (creditpurchase.Service, error) {
	creditPurchaseSvc, err := creditpurchaseservice.New(creditpurchaseservice.Config{
		Adapter:                 creditPurchaseAdapter,
		Handler:                 creditPurchaseHandler,
		Lineage:                 lineageService,
		MetaAdapter:             metaAdapter,
		CustomCurrenciesEnabled: creditsConfig.CustomCurrenciesEnabled,
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
	logger *slog.Logger,
	rootAdapter charges.Adapter,
	metaAdapter meta.Adapter,
	featureService feature.FeatureConnector,
	flatFeeSvc flatfee.Service,
	creditPurchaseSvc creditpurchase.Service,
	usageBasedSvc usagebased.Service,
	billingService billing.Service,
	recognizerService recognizer.Service,
	taxCodeService taxcode.Service,
	currencyResolver currencies.CurrencyResolver,
	fsNamespaceLockdown []string,
) (charges.Service, error) {
	chargesSvc, err := chargesservice.New(chargesservice.Config{
		Logger:                logger,
		Adapter:               rootAdapter,
		MetaAdapter:           metaAdapter,
		FeatureService:        featureService,
		FlatFeeService:        flatFeeSvc,
		CreditPurchaseService: creditPurchaseSvc,
		UsageBasedService:     usageBasedSvc,
		BillingService:        billingService,
		RecognizerService:     recognizerService,
		TaxCodeService:        taxCodeService,
		CurrencyResolver:      currencyResolver,
		FSNamespaceLockdown:   fsNamespaceLockdown,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges service: %w", err)
	}

	return chargesSvc, nil
}

func NewRecognizerService(
	db *entdb.Client,
	ledgerService ledger.Ledger,
	balanceQuerier ledger.BalanceQuerier,
	accountResolver ledger.AccountResolver,
	accountService ledgeraccount.Service,
	lineageService lineage.Service,
) (recognizer.Service, error) {
	return recognizer.NewService(recognizer.Config{
		Ledger: ledgerService,
		Dependencies: transactions.ResolverDependencies{
			AccountService: accountResolver,
			AccountCatalog: accountService,
			BalanceQuerier: balanceQuerier,
		},
		Lineage:            lineageService,
		TransactionManager: enttx.NewCreator(db),
	})
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
	balanceQuerier ledger.BalanceQuerier,
	accountResolver ledger.AccountResolver,
	accountService ledgeraccount.Service,
	breakageService ledgerbreakage.Service,
	taxCodeService taxcode.Service,
	currencyResolver currencies.CurrencyResolver,
	currenciesService currencies.Service,
	fsNamespaceLockdown []string,
	creditsConfig config.CreditsConfiguration,
	featureGate *featuregate.FeatureGateChecker,
) (*ChargesRegistry, error) {
	metaAdapter, err := NewChargesMetaAdapter(db, logger)
	if err != nil {
		return nil, err
	}

	lineageAdapter, err := NewChargesLineageAdapter(db)
	if err != nil {
		return nil, err
	}

	lineageService, err := NewChargesLineageService(lineageAdapter)
	if err != nil {
		return nil, err
	}

	transactionManager := enttx.NewCreator(db)
	collectorService, err := ledgercollector.NewService(ledgercollector.Config{
		Ledger: ledgerService,
		Dependencies: transactions.ResolverDependencies{
			AccountService: accountResolver,
			AccountCatalog: accountService,
			BalanceQuerier: balanceQuerier,
		},
		Breakage:           breakageService,
		AccountLocker:      accountService,
		TransactionManager: transactionManager,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges collector service: %w", err)
	}

	recognizerService, err := NewRecognizerService(db, ledgerService, balanceQuerier, accountResolver, accountService, lineageService)
	if err != nil {
		return nil, err
	}

	flatFeeHandler := NewChargesFlatFeeHandler(ledgerService, balanceQuerier, accountResolver, accountService, collectorService)
	usageBasedHandler := NewChargesUsageBasedHandler(ledgerService, balanceQuerier, accountResolver, accountService, collectorService)
	creditPurchaseHandler, err := NewChargesCreditPurchaseHandler(
		ledgerService,
		balanceQuerier,
		accountResolver,
		accountService,
		breakageService,
		transactionManager,
	)
	if err != nil {
		return nil, err
	}

	flatFeeAdapter, err := NewChargesFlatFeeAdapter(db, logger, metaAdapter)
	if err != nil {
		return nil, err
	}

	flatFeeSvc, err := NewChargesFlatFeeService(flatFeeAdapter, flatFeeHandler, lineageService, metaAdapter, locker, ratingService, currenciesService, creditsConfig)
	if err != nil {
		return nil, err
	}

	if err := billingService.RegisterLineEngine(flatFeeSvc.GetLineEngine()); err != nil {
		return nil, fmt.Errorf("failed to register charges flat fee line engine: %w", err)
	}

	usageBasedAdapter, err := NewChargesUsageBasedAdapter(db, logger, metaAdapter)
	if err != nil {
		return nil, err
	}

	invoiceUpdater, err := NewChargesInvoiceUpdater(billingService, logger)
	if err != nil {
		return nil, err
	}

	usageBasedSvc, err := NewChargesUsageBasedService(
		usageBasedAdapter,
		usageBasedHandler,
		lineageService,
		locker,
		metaAdapter,
		invoiceUpdater,
		billingService,
		featureService,
		ratingService,
		currenciesService,
		streamingConnector,
		creditsConfig,
	)
	if err != nil {
		return nil, err
	}

	if err := billingService.RegisterLineEngine(usageBasedSvc.GetLineEngine()); err != nil {
		return nil, fmt.Errorf("failed to register charges usage based line engine: %w", err)
	}

	creditPurchaseAdapter, err := NewChargesCreditPurchaseAdapter(db, logger, metaAdapter)
	if err != nil {
		return nil, err
	}

	creditPurchaseSvc, err := NewChargesCreditPurchaseService(creditPurchaseAdapter, creditPurchaseHandler, lineageService, metaAdapter, creditsConfig)
	if err != nil {
		return nil, err
	}

	creditPurchaseLineEngine, err := creditpurchaselineengine.New(creditpurchaselineengine.Config{
		RatingService: ratingService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges credit purchase line engine: %w", err)
	}

	if err := billingService.RegisterLineEngine(creditPurchaseLineEngine); err != nil {
		return nil, fmt.Errorf("failed to register charges credit purchase line engine: %w", err)
	}
	createLineRouter, err := chargeslinerouter.New(chargeslinerouter.Config{
		CreditsEnabled:           creditsConfig.Enabled,
		CreditThenInvoiceEnabled: creditsConfig.EnableCreditThenInvoice,
		FeatureGate:              featureGate,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create charges create line router: %w", err)
	}

	if err := billingService.RegisterCreateLineRouter(createLineRouter); err != nil {
		return nil, fmt.Errorf("failed to register charges create line router: %w", err)
	}

	rootAdapter, err := NewChargesAdapter(db, logger)
	if err != nil {
		return nil, err
	}

	chargesSvc, err := NewChargesService(
		logger,
		rootAdapter,
		metaAdapter,
		featureService,
		flatFeeSvc,
		creditPurchaseSvc,
		usageBasedSvc,
		billingService,
		recognizerService,
		taxCodeService,
		currencyResolver,
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
		RecognizerService:     recognizerService,
	}, nil
}
