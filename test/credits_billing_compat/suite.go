package creditsbillingcompat

import (
	metricnoop "go.opentelemetry.io/otel/metric/noop"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	legacylineageadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage/adapter"
	legacylineageservice "github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage/service"
	chargestestutils "github.com/openmeterio/openmeter/openmeter/billing/charges/testutils"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/openmeter/ledger/advance"
	advancetestutils "github.com/openmeterio/openmeter/openmeter/ledger/advance/testutils"
	ledgerbreakage "github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	ledgerbreakageadapter "github.com/openmeterio/openmeter/openmeter/ledger/breakage/adapter"
	ledgerchargeadapter "github.com/openmeterio/openmeter/openmeter/ledger/chargeadapter"
	ledgercollector "github.com/openmeterio/openmeter/openmeter/ledger/collector"
	"github.com/openmeterio/openmeter/openmeter/ledger/creditvoid"
	creditvoidadapter "github.com/openmeterio/openmeter/openmeter/ledger/creditvoid/adapter"
	"github.com/openmeterio/openmeter/openmeter/ledger/recognizer"
	ledgerresolvers "github.com/openmeterio/openmeter/openmeter/ledger/resolvers"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	omtestutils "github.com/openmeterio/openmeter/openmeter/testutils"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type BaseSuite struct {
	billingtest.BaseSuite

	// Charges + ledger
	CreditVoidService creditvoid.Service
	FlatFeeHandler    flatfee.Handler
	RevenueRecognizer recognizer.Service
	AdvanceService    advance.Service

	Charges              charges.Service
	FlatFeeSvc           flatfee.Service
	CreditPurchaseSvc    creditpurchase.Service
	UsageBasedSvc        usagebased.Service
	CurrencyService      currencies.Service
	CurrencyResolver     currencies.CurrencyResolver
	LineageService       legacylineage.Service
	Ledger               ledger.Ledger
	BalanceQuerier       ledger.BalanceQuerier
	LedgerAccountService ledgeraccount.Service
	LedgerResolver       *ledgerresolvers.AccountResolver
	BreakageService      ledgerbreakage.Service
}

// NewFlatFeeManuallyManagedGatheringLine creates a manually managed fixture accepted by charge-backed pending line creation.
func NewFlatFeeManuallyManagedGatheringLine(input billing.NewFlatFeeLineInput) billing.GatheringLine {
	input.ManagedBy = billing.ManuallyManagedLine

	return billing.NewFlatFeeGatheringLine(input)
}

// SetupAccounts provisions the namespace's business accounts and each customer's ledger accounts.
func (s *BaseSuite) SetupAccounts(namespace string, customerIDs ...string) {
	s.T().Helper()
	ctx := s.T().Context()

	_, err := s.LedgerResolver.EnsureBusinessAccounts(ctx, namespace)
	s.Require().NoError(err)

	for _, customerID := range customerIDs {
		_, err := s.LedgerResolver.CreateCustomerAccounts(ctx, customer.CustomerID{
			Namespace: namespace,
			ID:        customerID,
		})
		s.Require().NoError(err)
	}
}

func (s *BaseSuite) RequireChargeBasedGatheringLine(line billing.GatheringLine, expectedChargeID *string) {
	s.T().Helper()

	s.Require().True(line.Engine.IsCharge(), "gathering line %s uses engine %s", line.ID, line.Engine)
	s.Require().NotNil(expectedChargeID, "expected charge ID is required")
	s.Require().NotEmpty(*expectedChargeID, "gathering line must reference its backing charge")
	s.Require().Equal(expectedChargeID, line.ChargeID, "gathering line must reference the expected charge")
}

func (s *BaseSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
	s.setupCreditsSuite()
}

func (s *BaseSuite) setupCreditsSuite() {
	t := s.T()
	t.Log("setup suite")

	// Set up charges stack
	logger := omtestutils.NewLogger(s.T())

	deps, err := ledgertestutils.InitDeps(s.DBClient, logger)
	s.NoError(err)

	s.Ledger = deps.HistoricalLedger
	s.BalanceQuerier = deps.HistoricalLedger
	s.LedgerAccountService = deps.AccountService
	s.LedgerResolver = deps.ResolversService

	lineageAdapter, err := legacylineageadapter.New(legacylineageadapter.Config{
		Client: s.DBClient,
	})
	s.NoError(err)

	lineageService, err := legacylineageservice.New(legacylineageservice.Config{
		Adapter: lineageAdapter,
	})
	s.NoError(err)
	s.LineageService = lineageService

	transactionManager := enttx.NewCreator(s.DBClient)

	breakageAdapter, err := ledgerbreakageadapter.New(ledgerbreakageadapter.Config{
		Client: s.DBClient,
	})
	s.NoError(err)

	breakageService, err := ledgerbreakage.NewService(ledgerbreakage.Config{
		Adapter: breakageAdapter,
		Dependencies: transactions.ResolverDependencies{
			AccountService: deps.ResolversService,
			AccountCatalog: deps.AccountService,
			BalanceQuerier: deps.HistoricalLedger,
		},
	})
	s.NoError(err)
	s.BreakageService = breakageService
	s.AdvanceService = advancetestutils.NewService(s.T(), deps, breakageService)

	creditVoidAdapter, err := creditvoidadapter.New(creditvoidadapter.Config{
		Client: s.DBClient,
	})
	s.NoError(err)

	creditVoidService, err := creditvoid.NewService(creditvoid.Config{
		Adapter: creditVoidAdapter,
		Ledger:  deps.HistoricalLedger,
		Dependencies: transactions.ResolverDependencies{
			AccountService: deps.ResolversService,
			AccountCatalog: deps.AccountService,
			BalanceQuerier: deps.HistoricalLedger,
		},
		Breakage:           breakageService,
		AccountLocker:      deps.AccountService,
		TransactionManager: transactionManager,
	})
	s.NoError(err)
	s.CreditVoidService = creditVoidService

	revenueRecognizer, err := recognizer.NewService(recognizer.Config{
		Ledger: deps.HistoricalLedger,
		Dependencies: transactions.ResolverDependencies{
			AccountService: deps.ResolversService,
			AccountCatalog: deps.AccountService,
			BalanceQuerier: deps.HistoricalLedger,
		},
		Lineage:            lineageService,
		TransactionManager: transactionManager,
	})
	s.NoError(err)
	s.RevenueRecognizer = revenueRecognizer

	collectorService, err := ledgercollector.NewService(ledgercollector.Config{
		Meter:   metricnoop.NewMeterProvider().Meter("test"),
		Logger:  logger,
		Advance: s.AdvanceService,
		Ledger:  deps.HistoricalLedger,
		Dependencies: transactions.ResolverDependencies{
			AccountService: deps.ResolversService,
			AccountCatalog: deps.AccountService,
			BalanceQuerier: deps.HistoricalLedger,
		},
		Breakage:           breakageService,
		AccountLocker:      deps.AccountService,
		TransactionManager: transactionManager,
	})
	s.NoError(err)
	flatFeeHandler := ledgerchargeadapter.NewFlatFeeHandler(
		deps.HistoricalLedger,
		transactions.ResolverDependencies{AccountService: deps.ResolversService, AccountCatalog: deps.AccountService, BalanceQuerier: deps.HistoricalLedger},
		collectorService,
	)
	s.FlatFeeHandler = flatFeeHandler

	creditPurchaseHandler, err := ledgerchargeadapter.NewCreditPurchaseHandler(ledgerchargeadapter.CreditPurchaseHandlerConfig{
		Ledger:             deps.HistoricalLedger,
		BalanceQuerier:     deps.HistoricalLedger,
		AccountResolver:    deps.ResolversService,
		AccountCatalog:     deps.AccountService,
		AdvanceService:     s.AdvanceService,
		BreakageService:    breakageService,
		TransactionManager: transactionManager,
	})
	s.NoError(err)

	stack, err := chargestestutils.NewServices(s.T(), chargestestutils.Config{
		Client:                s.DBClient,
		Logger:                logger,
		BillingService:        s.BillingService,
		FeatureMeterResolver:  s.FeatureMeterResolver,
		StreamingConnector:    s.MockStreamingConnector,
		TaxCodeService:        s.TaxCodeService,
		CustomerService:       s.CustomerService,
		FlatFeeHandler:        flatFeeHandler,
		CreditPurchaseHandler: creditPurchaseHandler,
		UsageBasedHandler:     ledgerchargeadapter.NewUsageBasedHandler(deps.HistoricalLedger, transactions.ResolverDependencies{AccountService: deps.ResolversService, AccountCatalog: deps.AccountService, BalanceQuerier: deps.HistoricalLedger}, collectorService),
	})
	s.NoError(err)
	s.Charges = stack.ChargesService
	s.FlatFeeSvc = stack.FlatFeeService
	s.CreditPurchaseSvc = stack.CreditPurchaseService
	s.UsageBasedSvc = stack.UsageBasedService
	s.CurrencyService = stack.CurrencyService
	s.CurrencyResolver = stack.CurrencyResolver
}
