package rte_test

import (
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	creditstest "github.com/openmeterio/openmeter/test/credits"
)

func (s *RTEIntegrationSuite) TestCalculateStoreAndRead() {
	ctx := s.T().Context()
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	// Given a customer whose credits are posted through the real ledger.
	ns := s.GetUniqueNamespace("rte-balance")
	cust := s.CreateLedgerBackedCustomer(ns, "rte-customer")
	currency := currencies.NewCurrencyReference(creditstest.USD)
	dependencies := transactions.ResolverDependencies{
		AccountService: s.LedgerResolver,
		AccountCatalog: s.LedgerAccountService,
		BalanceQuerier: s.BalanceQuerier,
	}

	scope := transactions.ResolutionScope{
		CustomerID: cust.GetID(),
		Namespace:  ns,
	}

	postCredits := func(amount int64) {
		inputs, err := transactions.ResolveTransactions(ctx, dependencies, scope, transactions.IssueCustomerReceivableTemplate{
			At:       now,
			Amount:   alpacadecimal.NewFromInt(amount),
			Currency: currency,
		})
		s.Require().NoError(err)

		_, err = s.Ledger.CommitGroup(ctx, transactions.GroupInputs(ns, nil, inputs...))
		s.Require().NoError(err)

		// Issue alone leaves matching receivable debt; settle the payment so
		// these are funded credits, as in the existing ledger fixtures.
		settle, err := transactions.ResolveTransactions(ctx, dependencies, scope,
			transactions.AuthorizeCustomerReceivablePaymentTemplate{
				At:       now,
				Amount:   alpacadecimal.NewFromInt(amount),
				Currency: currency,
			},
			transactions.SettleCustomerReceivableFromPaymentTemplate{
				At:       now,
				Amount:   alpacadecimal.NewFromInt(amount),
				Currency: currency,
			},
		)
		s.Require().NoError(err)

		_, err = s.Ledger.CommitGroup(ctx, transactions.GroupInputs(ns, nil, settle...))
		s.Require().NoError(err)
	}

	postCredits(100)

	engine := s.newRTE()
	key := creditbalance.Key{CustomerID: cust.GetID(), Currency: currency}

	_, err := engine.CreditBalances().Get(ctx, key)
	s.True(models.IsGenericNotFoundError(err))

	// When explicitly calculated, the current live balance is preserved too.
	calculated, err := engine.CreditBalances().Calculate(ctx, key)
	s.Require().NoError(err)
	s.Equal(float64(100), calculated.Value.Booked.InexactFloat64())
	s.Equal(float64(100), calculated.Value.Spendable.InexactFloat64())
	s.Equal(float64(100), calculated.Value.Live.InexactFloat64())
	s.Equal(float64(0), calculated.Value.Pending.InexactFloat64())

	// Then a fresh service instance still reads that stored result after the
	// actual ledger changes. Only another calculate request picks up the change.
	postCredits(25)
	reader := s.newRTE()
	stored, err := reader.CreditBalances().Get(ctx, key)
	s.Require().NoError(err)
	s.True(calculated.CalculatedAt.Equal(stored.CalculatedAt))
	s.Equal(float64(100), stored.Value.Booked.InexactFloat64())
	s.Equal(float64(100), stored.Value.Spendable.InexactFloat64())
	s.Equal(float64(100), stored.Value.Live.InexactFloat64())
	s.Equal(float64(0), stored.Value.Pending.InexactFloat64())

	clock.UnFreeze()
	clock.FreezeTime(now.Add(time.Second))
	defer clock.UnFreeze()

	refreshed, err := reader.CreditBalances().Calculate(ctx, key)
	s.Require().NoError(err)
	s.True(refreshed.CalculatedAt.After(stored.CalculatedAt))
	s.Equal(float64(125), refreshed.Value.Booked.InexactFloat64())
	s.Equal(float64(125), refreshed.Value.Spendable.InexactFloat64())
	s.Equal(float64(125), refreshed.Value.Live.InexactFloat64())
}
