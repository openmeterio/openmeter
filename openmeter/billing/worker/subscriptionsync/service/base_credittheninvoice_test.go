package service

import (
	"log/slog"

	metricnoop "go.opentelemetry.io/otel/metric/noop"

	chargestestutils "github.com/openmeterio/openmeter/openmeter/billing/charges/testutils"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	advancetestutils "github.com/openmeterio/openmeter/openmeter/ledger/advance/testutils"
	ledgerbreakage "github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	ledgerchargeadapter "github.com/openmeterio/openmeter/openmeter/ledger/chargeadapter"
	ledgercollector "github.com/openmeterio/openmeter/openmeter/ledger/collector"
	ledgerresolvers "github.com/openmeterio/openmeter/openmeter/ledger/resolvers"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/featuregate"
)

type CreditThenInvoiceSuiteBase struct {
	SuiteBase
	LegacySyncService *Service
	ChargeSyncService *Service
	BalanceQuerier    ledger.BalanceQuerier
	LedgerResolver    *ledgerresolvers.AccountResolver
}

func (s *CreditThenInvoiceSuiteBase) SetupSuite() {
	s.SuiteBase.SetupSuite()
	s.LegacySyncService = s.Service

	logger := slog.Default()

	ledgerDeps, err := ledgertestutils.InitDeps(s.DBClient, logger)
	s.NoError(err)

	s.BalanceQuerier = ledgerDeps.HistoricalLedger
	s.LedgerResolver = ledgerDeps.ResolversService
	s.Ledger = ledgerDeps.HistoricalLedger

	transactionManager := enttx.NewCreator(s.DBClient)

	breakageService := ledgerbreakage.NewNoopService()

	advanceService := advancetestutils.NewService(s.T(), ledgerDeps, breakageService)

	collectorService, err := ledgercollector.NewService(ledgercollector.Config{
		Meter:   metricnoop.NewMeterProvider().Meter("test"),
		Logger:  logger,
		Advance: advanceService,
		Ledger:  ledgerDeps.HistoricalLedger,
		Dependencies: transactions.ResolverDependencies{
			AccountService: ledgerDeps.ResolversService,
			AccountCatalog: ledgerDeps.AccountService,
			BalanceQuerier: ledgerDeps.HistoricalLedger,
		},
		Breakage:           breakageService,
		AccountLocker:      ledgerDeps.AccountService,
		TransactionManager: transactionManager,
	})
	s.NoError(err)

	creditPurchaseHandler, err := ledgerchargeadapter.NewCreditPurchaseHandler(ledgerchargeadapter.CreditPurchaseHandlerConfig{
		Ledger:             ledgerDeps.HistoricalLedger,
		BalanceQuerier:     ledgerDeps.HistoricalLedger,
		AccountResolver:    ledgerDeps.ResolversService,
		AccountCatalog:     ledgerDeps.AccountService,
		AdvanceService:     advanceService,
		BreakageService:    breakageService,
		TransactionManager: transactionManager,
	})
	s.NoError(err)

	stack, err := chargestestutils.NewServices(s.T(), chargestestutils.Config{
		Client:               s.DBClient,
		Logger:               logger,
		BillingService:       s.BillingService,
		FeatureMeterResolver: s.FeatureMeterResolver,
		StreamingConnector:   s.MockStreamingConnector,
		TaxCodeService:       s.TaxCodeEnv.Service,
		CustomerService:      s.CustomerService,
		SubscriptionService:  s.SubscriptionService,
		FlatFeeHandler: ledgerchargeadapter.NewFlatFeeHandler(
			ledgerDeps.HistoricalLedger,
			transactions.ResolverDependencies{
				AccountService: ledgerDeps.ResolversService,
				AccountCatalog: ledgerDeps.AccountService,
				BalanceQuerier: ledgerDeps.HistoricalLedger,
			},
			collectorService,
		),
		CreditPurchaseHandler: creditPurchaseHandler,
		UsageBasedHandler: ledgerchargeadapter.NewUsageBasedHandler(
			ledgerDeps.HistoricalLedger,
			transactions.ResolverDependencies{
				AccountService: ledgerDeps.ResolversService,
				AccountCatalog: ledgerDeps.AccountService,
				BalanceQuerier: ledgerDeps.HistoricalLedger,
			},
			collectorService,
		),
	})
	s.NoError(err)

	s.Charges = stack.ChargesService

	service, err := New(Config{
		BillingService:          s.BillingService,
		LegacyBillingLineEngine: s.LegacyBillingLineEngine,
		ChargesService:          s.Charges,
		Logger:                  s.Service.logger,
		Tracer:                  s.Service.tracer,
		SubscriptionSyncAdapter: s.Adapter,
		SubscriptionService:     s.SubscriptionService,
		FeatureFlags: FeatureFlags{
			EnableCreditThenInvoice: true,
		},
		FeatureGate: featuregate.NewFeatureGateChecker(featuregate.NewNoop(), featuregate.Flags{
			featuregate.CtxKeyCredits: string(featuregate.CtxKeyCredits),
		}, map[featuregate.FeatureFlag]bool{featuregate.CtxKeyCredits: true}),
	})
	s.NoError(err)

	s.Service = service
	s.ChargeSyncService = service
}

func (s *CreditThenInvoiceSuiteBase) BeforeTest(suiteName, testName string) {
	s.SuiteBase.BeforeTest(suiteName, testName)

	_, err := s.LedgerResolver.EnsureBusinessAccounts(s.T().Context(), s.Namespace)
	s.NoError(err)

	_, err = s.LedgerResolver.CreateCustomerAccounts(s.T().Context(), s.Customer.GetID())
	s.NoError(err)
}
