package credits

import (
	"testing"

	"github.com/invopop/gobl/currency"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	lineageadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/adapter"
	lineageservice "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/service"
	chargestestutils "github.com/openmeterio/openmeter/openmeter/billing/charges/testutils"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	ledgerchargeadapter "github.com/openmeterio/openmeter/openmeter/ledger/chargeadapter"
	ledgercollector "github.com/openmeterio/openmeter/openmeter/ledger/collector"
	"github.com/openmeterio/openmeter/openmeter/ledger/recognizer"
	ledgerresolvers "github.com/openmeterio/openmeter/openmeter/ledger/resolvers"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	omtestutils "github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

const USD = currencyx.Code(currency.USD)

type CreditsTestSuite struct {
	billingtest.BaseSuite

	Charges              charges.Service
	Ledger               ledger.Ledger
	BalanceQuerier       ledger.BalanceQuerier
	LedgerAccountService ledgeraccount.Service
	LedgerResolver       *ledgerresolvers.AccountResolver
	RevenueRecognizer    recognizer.Service
}

func TestCreditsTestSuite(t *testing.T) {
	suite.Run(t, new(CreditsTestSuite))
}

func (s *CreditsTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	logger := omtestutils.NewLogger(s.T())

	deps, err := ledgertestutils.InitDeps(s.DBClient, logger)
	s.Require().NoError(err)

	s.Ledger = deps.HistoricalLedger
	s.BalanceQuerier = deps.HistoricalLedger
	s.LedgerAccountService = deps.AccountService
	s.LedgerResolver = deps.ResolversService

	lineageAdapter, err := lineageadapter.New(lineageadapter.Config{
		Client: s.DBClient,
	})
	s.Require().NoError(err)

	lineageService, err := lineageservice.New(lineageservice.Config{
		Adapter: lineageAdapter,
	})
	s.Require().NoError(err)

	revenueRecognizer, err := recognizer.NewService(recognizer.Config{
		Ledger: deps.HistoricalLedger,
		Dependencies: transactions.ResolverDependencies{
			AccountService: deps.ResolversService,
			AccountCatalog: deps.AccountService,
			BalanceQuerier: deps.HistoricalLedger,
		},
		Lineage:            lineageService,
		TransactionManager: enttx.NewCreator(s.DBClient),
	})
	s.Require().NoError(err)
	s.RevenueRecognizer = revenueRecognizer

	collectorService := ledgercollector.NewService(ledgercollector.Config{
		Ledger: deps.HistoricalLedger,
		Dependencies: transactions.ResolverDependencies{
			AccountService: deps.ResolversService,
			AccountCatalog: deps.AccountService,
			BalanceQuerier: deps.HistoricalLedger,
		},
	})

	stack, err := chargestestutils.NewServices(s.T(), chargestestutils.Config{
		Client:                s.DBClient,
		Logger:                logger,
		BillingService:        s.BillingService,
		FeatureService:        s.FeatureService,
		StreamingConnector:    s.MockStreamingConnector,
		FlatFeeHandler:        ledgerchargeadapter.NewFlatFeeHandler(deps.HistoricalLedger, transactions.ResolverDependencies{AccountService: deps.ResolversService, AccountCatalog: deps.AccountService, BalanceQuerier: deps.HistoricalLedger}, collectorService),
		CreditPurchaseHandler: ledgerchargeadapter.NewCreditPurchaseHandler(deps.HistoricalLedger, deps.HistoricalLedger, deps.ResolversService, deps.AccountService),
		UsageBasedHandler:     ledgerchargeadapter.NewUsageBasedHandler(deps.HistoricalLedger, transactions.ResolverDependencies{AccountService: deps.ResolversService, AccountCatalog: deps.AccountService, BalanceQuerier: deps.HistoricalLedger}, collectorService),
	})
	s.Require().NoError(err)
	s.Charges = stack.ChargesService
}

func (s *CreditsTestSuite) TearDownTest() {
	s.MockStreamingConnector.Reset()
	clock.UnFreeze()
	clock.ResetTime()
}
