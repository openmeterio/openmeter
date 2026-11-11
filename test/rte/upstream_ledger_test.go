package rte_test

import (
	"context"
	"errors"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/eventoutbox"
	"github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransactiongroup"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/ledger/historical"
	historicaladapter "github.com/openmeterio/openmeter/openmeter/ledger/historical/adapter"
	"github.com/openmeterio/openmeter/openmeter/ledger/routingrules"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/rte"
	rteadapter "github.com/openmeterio/openmeter/openmeter/rte/adapter"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/rte/upstream"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *RTEIntegrationSuite) TestLedgerSourceFactSharesPostingTransaction() {
	// given: the real ledger uses the same transactional outbox as RTE.
	ctx := s.T().Context()
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()
	ns := s.GetUniqueNamespace("rte-ledger-source")
	cust := s.CreateLedgerBackedCustomer(ns, "rte-ledger-customer")
	pipeline := s.newEventPipeline()
	writer := historical.NewLedger(historicaladapter.NewRepo(s.DBClient), s.LedgerAccountService, s.LedgerAccountService,
		routingrules.DefaultValidator, historical.WithEventPublisher(pipeline.publisher))
	inputs, err := transactions.ResolveTransactions(ctx, transactions.ResolverDependencies{
		AccountService: s.LedgerResolver, AccountCatalog: s.LedgerAccountService, BalanceQuerier: s.BalanceQuerier,
	}, transactions.ResolutionScope{CustomerID: cust.GetID(), Namespace: ns}, transactions.IssueCustomerReceivableTemplate{
		At: start, Amount: alpacadecimal.NewFromInt(10), Currency: currencies.NewCurrencyReference("USD"),
	})
	s.Require().NoError(err)
	rollback := errors.New("abort posting")
	var rolledBack models.NamespacedID

	// when: the enclosing operation fails after the ledger publishes its fact.
	err = transaction.RunWithNoValue(ctx, enttx.NewCreator(s.DBClient), func(ctx context.Context) error {
		group, err := writer.CommitGroup(ctx, transactions.GroupInputs(ns, nil, inputs...))
		if err != nil {
			return err
		}

		rolledBack = group.ID()
		tx, err := entutils.GetDriverFromContext(ctx)
		if err != nil {
			return err
		}

		client := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client()
		count, err := client.EventOutbox.Query().Where(eventoutbox.TransactionIDEQ(tx.ID())).Count(ctx)
		s.Require().NoError(err)
		s.Equal(1, count)

		return rollback
	})
	s.Require().ErrorIs(err, rollback)

	// then: neither the committed-posting fact nor its postings survive rollback.
	exists, err := s.DBClient.LedgerTransactionGroup.Query().Where(ledgertransactiongroup.IDEQ(rolledBack.ID)).Exist(ctx)
	s.Require().NoError(err)
	s.False(exists)
	group, err := writer.CommitGroup(ctx, transactions.GroupInputs(ns, nil, inputs...))
	s.Require().NoError(err)
	logger := testutils.NewDiscardLogger(s.T())
	repository, err := rteadapter.NewDemandRepository(rteadapter.Config{Client: s.DBClient, Logger: logger})
	s.Require().NoError(err)
	resolver, err := upstream.New(upstream.Config{Client: s.DBClient, Logger: logger})
	s.Require().NoError(err)
	base := creditbalance.Key{CustomerID: cust.GetID(), Currency: currencies.NewCurrencyReference("USD")}
	unrestricted := base
	unrestricted.FeatureFilter = customerbalance.NewUnrestrictedFeatureFilter()
	otherCurrency := base
	otherCurrency.Currency = currencies.NewCurrencyReference("EUR")
	for _, key := range []rte.Key{base, unrestricted, otherCurrency} {
		event, err := rte.NewCalculationRequestedEvent(key)
		s.Require().NoError(err)
		s.Require().NoError(repository.Activate(ctx, rte.ActivateDemandInput{Target: event.CalculationTarget}))
		defer func() {
			s.Require().NoError(repository.Deactivate(ctx, event.CalculationTarget))
		}()
	}

	selectors, err := resolver.Resolve(ctx, ledger.TransactionGroupCommittedEvent{Namespace: ns, GroupID: group.ID().ID})
	s.Require().NoError(err)
	s.Require().Len(selectors, 1)
	s.Require().Len(selectors[0].Targets, 2)
	for _, selector := range selectors {
		for _, target := range selector.Targets {
			key, err := target.DecodeKey()
			s.Require().NoError(err)
			s.Equal("USD", key.(creditbalance.Key).Currency.Code.String())
		}
	}

	// Custom currency selection follows persisted identity, including historical
	// currencies that reuse a display code.
	custom := s.CreateCustomCurrency(ns, "TOKENS")
	customKey := base
	customKey.Currency = custom.Reference()
	otherCustomKey := customKey
	otherID := ulid.Make().String()
	otherCustomKey.Currency = currencies.CurrencyReference{Code: custom.GetCode(), CustomCurrencyID: &otherID}
	for _, key := range []rte.Key{customKey, otherCustomKey} {
		event, err := rte.NewCalculationRequestedEvent(key)
		s.Require().NoError(err)
		s.Require().NoError(repository.Activate(ctx, rte.ActivateDemandInput{Target: event.CalculationTarget}))
		defer func() {
			s.Require().NoError(repository.Deactivate(ctx, event.CalculationTarget))
		}()
	}

	customInputs, err := transactions.ResolveTransactions(ctx, transactions.ResolverDependencies{
		AccountService: s.LedgerResolver, AccountCatalog: s.LedgerAccountService, BalanceQuerier: s.BalanceQuerier,
	}, transactions.ResolutionScope{CustomerID: cust.GetID(), Namespace: ns}, transactions.IssueCustomerReceivableTemplate{
		At: start, Amount: alpacadecimal.NewFromInt(10), Currency: custom.Reference(),
	})
	s.Require().NoError(err)
	customGroup, err := writer.CommitGroup(ctx, transactions.GroupInputs(ns, nil, customInputs...))
	s.Require().NoError(err)
	selectors, err = resolver.Resolve(ctx, ledger.TransactionGroupCommittedEvent{Namespace: ns, GroupID: customGroup.ID().ID})
	s.Require().NoError(err)
	s.Require().Len(selectors, 1)
	s.Require().Len(selectors[0].Targets, 1)
	key, err := selectors[0].Targets[0].DecodeKey()
	s.Require().NoError(err)
	s.True(key.(creditbalance.Key).Currency.Equal(custom.Reference()))
}
