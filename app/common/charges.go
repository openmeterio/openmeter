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
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	lineageadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/adapter"
	lineageservice "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	chargesservice "github.com/openmeterio/openmeter/openmeter/billing/charges/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/adapter"
	usagebasedservice "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
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
) ledgercollector.Service {
	return ledgercollector.NewService(ledgercollector.Config{
		Ledger: ledgerService,
		Dependencies: transactions.ResolverDependencies{
			AccountService: accountResolver,
			AccountCatalog: accountService,
			BalanceQuerier: balanceQuerier,
		},
		TransactionManager: enttx.NewCreator(db),
	})
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
) creditpurchase.Handler {
	return ledgerchargeadapter.NewCreditPurchaseHandler(ledgerService, balanceQuerier, accountResolver, accountService, breakageService, transactionManager)
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
) (flatfee.Service, error) {
	flatFeeSvc, err := flatfeeservice.New(flatfeeservice.Config{
		Adapter:       flatFeeAdapter,
		Handler:       flatFeeHandler,
		Lineage:       lineageService,
		MetaAdapter:   metaAdapter,
		Locker:        locker,
		RatingService: ratingService,
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
	billingService billing.Service,
	featureService feature.FeatureConnector,
	ratingService rating.Service,
	streamingConnector streaming.Connector,
) (usagebased.Service, error) {
	usageBasedSvc, err := usagebasedservice.New(usagebasedservice.Config{
		Adapter:                 usageBasedAdapter,
		Handler:                 usageBasedHandler,
		Lineage:                 lineageService,
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
	lineageService lineage.Service,
	metaAdapter meta.Adapter,
) (creditpurchase.Service, error) {
	creditPurchaseSvc, err := creditpurchaseservice.New(creditpurchaseservice.Config{
		Adapter:     creditPurchaseAdapter,
		Handler:     creditPurchaseHandler,
		Lineage:     lineageService,
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
	logger *slog.Logger,
	rootAdapter charges.Adapter,
	metaAdapter meta.Adapter,
	featureService feature.FeatureConnector,
	flatFeeSvc flatfee.Service,
	creditPurchaseSvc creditpurchase.Service,
	usageBasedSvc usagebased.Service,
	billingService billing.Service,
	recognizerService recognizer.Service,
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
	fsNamespaceLockdown []string,
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
	collectorService := ledgercollector.NewService(ledgercollector.Config{
		Ledger: ledgerService,
		Dependencies: transactions.ResolverDependencies{
			AccountService: accountResolver,
			AccountCatalog: accountService,
			BalanceQuerier: balanceQuerier,
		},
		Breakage:           breakageService,
		TransactionManager: transactionManager,
	})

	recognizerService, err := NewRecognizerService(db, ledgerService, balanceQuerier, accountResolver, accountService, lineageService)
	if err != nil {
		return nil, err
	}

	flatFeeHandler := NewChargesFlatFeeHandler(ledgerService, balanceQuerier, accountResolver, accountService, collectorService)
	usageBasedHandler := NewChargesUsageBasedHandler(ledgerService, balanceQuerier, accountResolver, accountService, collectorService)
	creditPurchaseHandler := NewChargesCreditPurchaseHandler(
		ledgerService,
		balanceQuerier,
		accountResolver,
		accountService,
		breakageService,
		transactionManager,
	)

	flatFeeAdapter, err := NewChargesFlatFeeAdapter(db, logger, metaAdapter)
	if err != nil {
		return nil, err
	}

	flatFeeSvc, err := NewChargesFlatFeeService(flatFeeAdapter, flatFeeHandler, lineageService, metaAdapter, locker, ratingService)
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

	usageBasedSvc, err := NewChargesUsageBasedService(
		usageBasedAdapter,
		usageBasedHandler,
		lineageService,
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

	if err := billingService.RegisterLineEngine(usageBasedSvc.GetLineEngine()); err != nil {
		return nil, fmt.Errorf("failed to register charges usage based line engine: %w", err)
	}

	creditPurchaseAdapter, err := NewChargesCreditPurchaseAdapter(db, logger, metaAdapter)
	if err != nil {
		return nil, err
	}

	creditPurchaseSvc, err := NewChargesCreditPurchaseService(creditPurchaseAdapter, creditPurchaseHandler, lineageService, metaAdapter)
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
