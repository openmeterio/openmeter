package service

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/billing/ledger"
	"github.com/openmeterio/openmeter/openmeter/billing/ledger/adapter"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/tools/migrate"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ServiceTestSuite struct {
	suite.Suite
	*require.Assertions

	TestDB   *testutils.TestDB
	DBClient *db.Client

	CustomerService customer.Service
	LedgerService   ledger.Service
}

func TestService(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

func (s *ServiceTestSuite) SetupSuite() {
	s.Assertions = require.New(s.T())

	s.TestDB = testutils.InitPostgresDB(s.T())
	s.DBClient = db.NewClient(db.Driver(s.TestDB.EntDriver.Driver()))

	if os.Getenv("TEST_DISABLE_ATLAS") != "" {
		s.Require().NoError(s.DBClient.Schema.Create(context.Background()))
	} else {
		s.Require().NoError(migrate.Up(s.TestDB.URL))
	}
	// customer

	customerAdapter, err := customeradapter.New(customeradapter.Config{
		Client: s.DBClient,
		Logger: slog.Default(),
	})
	s.NoError(err)

	publisher := eventbus.NewMock(s.T())

	meterAdapter, err := meteradapter.New(nil)
	s.NoError(err)

	entitlementRegistry := registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:     s.DBClient,
		StreamingConnector: streamingtestutils.NewMockStreamingConnector(s.T()),
		Logger:             slog.Default(),
		MeterService:       meterAdapter,
		Publisher:          publisher,
		EntitlementsConfiguration: config.EntitlementsConfiguration{
			GracePeriod: isodate.String("P1D"),
		},
	})

	customerService, err := customerservice.New(customerservice.Config{
		Adapter:              customerAdapter,
		EntitlementConnector: entitlementRegistry.Entitlement,
		Publisher:            publisher,
	})
	s.NoError(err)
	s.CustomerService = customerService

	// ledger

	ledgerAdapter, err := adapter.New(adapter.Config{
		Client: s.DBClient,
	})
	s.NoError(err)

	ledgerService, err := NewService(Config{
		Adapter: ledgerAdapter,
	})
	s.Require().NoError(err)

	s.LedgerService = ledgerService
}

func (s *ServiceTestSuite) TearDownSuite() {
	s.TestDB.EntDriver.Close()
	s.TestDB.PGDriver.Close()
}

func (s *ServiceTestSuite) CreateTestCustomer(ns string, subjectKey string) *customer.Customer {
	s.T().Helper()

	customer, err := s.CustomerService.CreateCustomer(context.Background(), customer.CreateCustomerInput{
		Namespace: ns,

		CustomerMutate: customer.CustomerMutate{
			Name:     "Test Customer",
			Currency: lo.ToPtr(currencyx.Code(currency.USD)),
			UsageAttribution: customer.CustomerUsageAttribution{
				SubjectKeys: []string{subjectKey},
			},
		},
	})

	s.NoError(err)
	return customer
}

func (s *ServiceTestSuite) TestLedgerSanity() {
	namespace := "ns-ledger-sanity"
	customer := s.CreateTestCustomer(namespace, "test-customer")

	ledgerRef := ledger.LedgerRef{
		Customer: customer.GetID(),
		Currency: currencyx.Code(currency.USD),
	}

	var purchasedCredits, promotionalCredits ledger.Subledger

	s.NoError(s.LedgerService.WithLockedLedger(context.Background(), ledger.WithLockedLedgerInput{
		LedgerRef: ledgerRef,
		Callback: func(ctx context.Context, l ledger.LedgerMutationService) error {
			var err error
			// Let's upsert two subledgers: one for purchased credits/refunds, one for promotioal credits
			purchasedCredits, err = l.UpsertSubledger(ctx, ledger.UpsertSubledgerInput{
				Key:      "purchased-credits",
				Priority: 0,
				Name:     "Purchased credits",
			})
			s.NoError(err)
			s.NotEmpty(purchasedCredits)

			promotionalCredits, err = l.UpsertSubledger(ctx, ledger.UpsertSubledgerInput{
				Key:      "promotional-credits",
				Priority: 1,
				Name:     "Promotional credits",
			})
			s.NoError(err)
			s.NotEmpty(promotionalCredits)

			l.CreateTransaction(ctx, ledger.CreateTransactionInput{
				Subledger: purchasedCredits,
				Amount:    alpacadecimal.NewFromInt(100),
				TransactionMeta: ledger.TransactionMeta{
					Name: "Credits purchased",
				},
			})
			s.NoError(err)

			return nil
		},
	}))

	// When we only have available balance on the purchased credits
	balance, err := s.LedgerService.GetBalance(context.Background(), ledgerRef)
	s.NoError(err)
	s.Equal(100.0, balance.Balance.InexactFloat64())
	s.Len(balance.SubledgerBalances, 2)
	// Resulting balances are sorted by priority
	s.Equal(100.0, balance.SubledgerBalances[0].Balance.InexactFloat64())
	s.Equal("purchased-credits", balance.SubledgerBalances[0].Subledger.Key)
	s.Equal(0.0, balance.SubledgerBalances[1].Balance.InexactFloat64())
	s.Equal("promotional-credits", balance.SubledgerBalances[1].Subledger.Key)

	// Then we should only see balance allocated from there
	s.NoError(s.LedgerService.WithLockedLedger(context.Background(), ledger.WithLockedLedgerInput{
		LedgerRef: ledgerRef,
		Callback: func(ctx context.Context, l ledger.LedgerMutationService) error {
			wdRes, err := l.Withdraw(ctx, ledger.WithdrawInput{
				Amount: alpacadecimal.NewFromInt(120),
				TransactionMeta: ledger.TransactionMeta{
					Name: "Credits spent",
				},
			})
			s.NoError(err)
			s.Equal(100.0, wdRes.TotalWithdrawn.InexactFloat64())
			s.Len(wdRes.Transactions, 1)
			s.Equal(-100.0, wdRes.Transactions[0].Amount.InexactFloat64())
			s.Equal(purchasedCredits.ID, wdRes.Transactions[0].SubledgerID)
			return nil
		},
	}))

	// Then we don't have any balance left
	balance, err = s.LedgerService.GetBalance(context.Background(), ledgerRef)
	s.NoError(err)
	s.Equal(0.0, balance.Balance.InexactFloat64())
	s.Len(balance.SubledgerBalances, 2)
}

func (s *ServiceTestSuite) TestLedgerTwoSubledgers() {
	namespace := "ns-ledger-two-subledgers"
	customer := s.CreateTestCustomer(namespace, "test-customer")

	ledgerRef := ledger.LedgerRef{
		Customer: customer.GetID(),
		Currency: currencyx.Code(currency.USD),
	}

	var purchasedCredits, promotionalCredits ledger.Subledger

	s.NoError(s.LedgerService.WithLockedLedger(context.Background(), ledger.WithLockedLedgerInput{
		LedgerRef: ledgerRef,
		Callback: func(ctx context.Context, l ledger.LedgerMutationService) error {
			var err error
			// Let's upsert two subledgers: one for purchased credits/refunds, one for promotioal credits
			purchasedCredits, err = l.UpsertSubledger(ctx, ledger.UpsertSubledgerInput{
				Key:      "purchased-credits",
				Priority: 0,
				Name:     "Purchased credits",
			})
			s.NoError(err)
			s.NotEmpty(purchasedCredits)

			promotionalCredits, err = l.UpsertSubledger(ctx, ledger.UpsertSubledgerInput{
				Key:      "promotional-credits",
				Priority: 1,
				Name:     "Promotional credits",
			})
			s.NoError(err)
			s.NotEmpty(promotionalCredits)

			l.CreateTransaction(ctx, ledger.CreateTransactionInput{
				Subledger: purchasedCredits,
				Amount:    alpacadecimal.NewFromInt(75),
				TransactionMeta: ledger.TransactionMeta{
					Name: "Credits purchased",
				},
			})
			s.NoError(err)

			l.CreateTransaction(ctx, ledger.CreateTransactionInput{
				Subledger: promotionalCredits,
				Amount:    alpacadecimal.NewFromInt(55),
				TransactionMeta: ledger.TransactionMeta{
					Name: "Promotional credits purchased",
				},
			})
			s.NoError(err)

			return nil
		},
	}))

	// Both balances are available
	balance, err := s.LedgerService.GetBalance(context.Background(), ledgerRef)
	s.NoError(err)
	s.Equal(130.0, balance.Balance.InexactFloat64())
	s.Len(balance.SubledgerBalances, 2)
	// Resulting balances are sorted by priority
	s.Equal(75.0, balance.SubledgerBalances[0].Balance.InexactFloat64())
	s.Equal("purchased-credits", balance.SubledgerBalances[0].Subledger.Key)
	s.Equal(55.0, balance.SubledgerBalances[1].Balance.InexactFloat64())
	s.Equal("promotional-credits", balance.SubledgerBalances[1].Subledger.Key)

	// Then we should only see balance allocated from there
	s.NoError(s.LedgerService.WithLockedLedger(context.Background(), ledger.WithLockedLedgerInput{
		LedgerRef: ledgerRef,
		Callback: func(ctx context.Context, l ledger.LedgerMutationService) error {
			wdRes, err := l.Withdraw(ctx, ledger.WithdrawInput{
				Amount: alpacadecimal.NewFromInt(120),
				TransactionMeta: ledger.TransactionMeta{
					Name: "Credits spent",
				},
			})
			s.NoError(err)
			s.Equal(120.0, wdRes.TotalWithdrawn.InexactFloat64())
			s.Len(wdRes.Transactions, 2)
			s.Equal(-55.0, wdRes.Transactions[0].Amount.InexactFloat64())
			s.Equal(promotionalCredits.ID, wdRes.Transactions[0].SubledgerID)
			s.Equal(-65.0, wdRes.Transactions[1].Amount.InexactFloat64())
			s.Equal(purchasedCredits.ID, wdRes.Transactions[1].SubledgerID)
			return nil
		},
	}))

	// The purchased credits still have some balance left
	balance, err = s.LedgerService.GetBalance(context.Background(), ledgerRef)
	s.NoError(err)
	s.Equal(10.0, balance.Balance.InexactFloat64())
	s.Len(balance.SubledgerBalances, 2)
	s.Equal(10.0, balance.SubledgerBalances[0].Balance.InexactFloat64())
	s.Equal("purchased-credits", balance.SubledgerBalances[0].Subledger.Key)
	s.Equal(0.0, balance.SubledgerBalances[1].Balance.InexactFloat64())
	s.Equal("promotional-credits", balance.SubledgerBalances[1].Subledger.Key)
}
