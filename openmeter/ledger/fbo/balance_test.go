package fbo_test

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/fbo"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestConsumptionPreservesFutureSpending(t *testing.T) {
	type grant struct {
		at     time.Duration
		amount int64
	}

	for _, tc := range []struct {
		name        string
		grants      []grant
		futureSpend int64
		expected    float64
	}{
		{name: "later consumption keeps its allocation", grants: []grant{{0, 10}}, futureSpend: 7, expected: 3},
		{name: "later consumption exhausts credit", grants: []grant{{0, 10}}, futureSpend: 10, expected: 0},
		{name: "future grant cannot fund earlier consumption", grants: []grant{{2 * time.Hour, 10}}, expected: 0},
		{name: "later grant backs later spending", grants: []grant{{0, 4}, {2 * time.Hour, 6}}, futureSpend: 6, expected: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env, service, _ := newFBOEnv(t)
			at := env.Now()
			sourceID := ulid.Make().String()

			// given: grants and a future consumption are already committed.
			for _, grant := range tc.grants {
				bookFBOTransactions(t, env, transactions.IssueCustomerReceivableTemplate{
					At: at.Add(grant.at), Amount: alpacadecimal.NewFromInt(grant.amount),
					Currency: env.CurrencyReference(), SourceChargeID: &sourceID,
				})
			}

			if tc.futureSpend > 0 {
				future := consumeInput(env, "future-usage", tc.futureSpend)
				future.BookedAt = at.Add(3 * time.Hour)
				future.SourceBalanceAsOf = future.BookedAt
				consumeAndCommit(t, env, service, future)
			}

			// when: another collection is booked before that consumption.
			input := consumeInput(env, "earlier-usage", 10)
			input.BookedAt = at.Add(time.Hour)
			input.SourceBalanceAsOf = input.BookedAt
			consumeAndCommit(t, env, service, input)

			// then: it collects only credit left after preserving future spending.
			requireAccountAmount(t, env, env.CustomerAccounts.AccruedAccount, float64(tc.futureSpend)+tc.expected)
		})
	}
}

func TestSpendableBalanceRespectsBreakageBoundary(t *testing.T) {
	for _, tc := range []struct {
		name     string
		at       time.Duration
		expected float64
	}{
		{name: "before breakage", at: 24 * time.Hour, expected: 3},
		{name: "at breakage", at: 10 * 24 * time.Hour, expected: 0},
		{name: "after breakage", at: 11 * 24 * time.Hour, expected: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env, service, breakageService := newFBOEnv(t)

			// given: ten expiring credits, seven consumed before their breakage date.
			bookExpiringCredit(t, env, breakageService, 10)
			future := consumeInput(env, "existing-usage", 7)
			future.BookedAt = env.Now().Add(5 * 24 * time.Hour)
			future.SourceBalanceAsOf = future.BookedAt
			consumeAndCommit(t, env, service, future)

			// when: consumption is requested before, at, or after breakage.
			input := consumeInput(env, "new-usage", 10)
			input.BookedAt = env.Now().Add(tc.at)
			input.SourceBalanceAsOf = input.BookedAt
			consumeAndCommit(t, env, service, input)

			// then: future breakage can be released, but expired credit stays unavailable.
			requireAccountAmount(t, env, env.CustomerAccounts.AccruedAccount, 7+tc.expected)
			requireAccountAmount(t, env, env.CustomerAccounts.FBOAccount, 0)
		})
	}
}

func TestSpendableBalanceSumsIndependentSourceMinima(t *testing.T) {
	env, service, _ := newFBOEnv(t)
	at := env.Now()
	sourceA, sourceB := ulid.Make().String(), ulid.Make().String()

	// given: source A dips to two, then is restored as source B dips to three.
	// The combined journal never drops below twelve; collectable capacity is five.
	for _, source := range []string{sourceA, sourceB} {
		bookFBOTransactions(t, env, transactions.IssueCustomerReceivableTemplate{
			At: at, Amount: alpacadecimal.NewFromInt(10), Currency: env.CurrencyReference(), SourceChargeID: &source,
		})
	}

	first := consumeInput(env, "first", 8)
	first.BookedAt = at.Add(time.Hour)
	first.SourceBalanceAsOf = first.BookedAt
	original := consumeAndCommit(t, env, service, first)
	require.Len(t, original.Transactions(), 1)
	tx := original.Transactions()[0]
	var negative, positive ledger.Entry
	for _, entry := range tx.Entries() {
		if entry.Amount().IsNegative() {
			negative = entry
		} else {
			positive = entry
		}
	}

	require.NotNil(t, negative)
	require.NotNil(t, positive)
	require.Equal(t, sourceA, *negative.Provenance().SourceChargeID)
	reversal, err := transactions.ReverseOriginEntryPair(transactions.ReverseOriginEntryPairInput{
		At: at.Add(2 * time.Hour), Amount: alpacadecimal.NewFromInt(8), Transaction: tx,
		NegativeEntry: negative, PositiveEntry: positive,
	})
	require.NoError(t, err)
	_, err = env.Deps.HistoricalLedger.CommitGroup(t.Context(), reversal.AsGroupInput(env.Namespace, nil))
	require.NoError(t, err)
	bookFBOTransactions(t, env, transactions.TransferCustomerFBOToAccruedTemplate{
		At: at.Add(2 * time.Hour), Currency: env.CurrencyReference(),
		Sources: []transactions.PostingAmount{{
			Address: negative.PostingAddress(), Amount: alpacadecimal.NewFromInt(7),
			Identity: ledger.EntryIdentityParts{Provenance: ledger.Provenance{SourceChargeID: &sourceB, SpendChargeID: lo.ToPtr("second")}},
		}},
	})

	// when: the API omits source grouping at the opening boundary.
	query := fbo.SpendableBalanceBucketQuery{Namespace: env.Namespace, Filters: ledger.Filters{
		AccountID: lo.ToPtr(env.CustomerAccounts.FBOAccount.ID().ID), AsOf: &at,
		Route: ledger.RouteFilter{Currency: env.CurrencyReference()},
	}}
	combined, err := service.GetSpendableBalanceBuckets(t.Context(), query)
	require.NoError(t, err)

	// then: presentation grouping cannot borrow capacity between sources.
	require.Len(t, combined, 1)
	require.Empty(t, combined[0].GroupByValues)
	require.Equal(t, 5.0, combined[0].SettledAmount.InexactFloat64())
	accountBalance, err := env.Deps.HistoricalLedger.GetAccountBalance(t.Context(), env.CustomerAccounts.FBOAccount,
		query.Filters.Route, ledger.BalanceQuery{AsOf: &at})
	require.NoError(t, err)
	require.Equal(t, 20.0, accountBalance.InexactFloat64())
	combinedMinimum, err := env.Deps.HistoricalLedger.GetMinimumBalanceBuckets(t.Context(), ledger.MinimumBalanceBucketQuery{
		BalanceBucketQuery: ledger.BalanceBucketQuery(query),
	})
	require.NoError(t, err)
	require.Len(t, combinedMinimum, 1)
	require.Equal(t, 12.0, combinedMinimum[0].SettledAmount.InexactFloat64())

	query.GroupBy = []string{ledger.BalanceBucketGroupBySourceChargeID}
	bySource, err := service.GetSpendableBalanceBuckets(t.Context(), query)
	require.NoError(t, err)
	require.Len(t, bySource, 2)
	require.Equal(t, 2.0, bySource[0].SettledAmount.InexactFloat64())
	require.Equal(t, 3.0, bySource[1].SettledAmount.InexactFloat64())

	query.Filters.Provenance.SourceChargeID = mo.Some(&sourceB)
	bySource, err = service.GetSpendableBalanceBuckets(t.Context(), query)
	require.NoError(t, err)
	require.Len(t, bySource, 1)
	require.Equal(t, sourceB, *bySource[0].GroupByValues[ledger.BalanceBucketGroupBySourceChargeID])
	require.Equal(t, 3.0, bySource[0].SettledAmount.InexactFloat64())

	// The committed return is available once its own effective boundary is reached.
	query.Filters.AsOf = lo.ToPtr(at.Add(2 * time.Hour))
	query.Filters.Provenance.SourceChargeID = mo.Some(&sourceA)
	returned, err := service.GetSpendableBalanceBuckets(t.Context(), query)
	require.NoError(t, err)
	require.Len(t, returned, 1)
	require.Equal(t, 10.0, returned[0].SettledAmount.InexactFloat64())

	query.Filters.AccountID = lo.ToPtr(env.CustomerAccounts.ReceivableAccount.ID().ID)
	_, err = service.GetSpendableBalanceBuckets(t.Context(), query)
	require.ErrorContains(t, err, "customer FBO account")
}

func TestSpendableBalanceQueryRequiresCompleteSources(t *testing.T) {
	at := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name    string
		filters ledger.Filters
		groupBy []string
	}{
		{name: "missing boundary", filters: ledger.Filters{AccountID: lo.ToPtr("account")}},
		{name: "missing account", filters: ledger.Filters{AsOf: &at}},
		{name: "transaction", filters: ledger.Filters{AsOf: &at, AccountID: lo.ToPtr("account"), TransactionID: lo.ToPtr("transaction")}},
		{name: "period", filters: ledger.Filters{AsOf: &at, AccountID: lo.ToPtr("account"), BookedAtPeriod: &timeutil.OpenPeriod{From: &at}}},
		{name: "spend filter", filters: ledger.Filters{AsOf: &at, AccountID: lo.ToPtr("account"), Provenance: ledger.ProvenanceFilter{SpendChargeID: mo.Some(lo.ToPtr("spend"))}}},
		{name: "origin filter", filters: ledger.Filters{AsOf: &at, AccountID: lo.ToPtr("account"), Provenance: ledger.ProvenanceFilter{CollectionOriginID: mo.Some[*string](nil)}}},
		{name: "spend grouping", filters: ledger.Filters{AsOf: &at, AccountID: lo.ToPtr("account")}, groupBy: []string{ledger.BalanceBucketGroupBySpendChargeID}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Error(t, (fbo.SpendableBalanceBucketQuery{Namespace: "ns", Filters: tc.filters, GroupBy: tc.groupBy}).Validate())
		})
	}
}

func bookFBOTransactions(t *testing.T, env *ledgertestutils.IntegrationEnv, templates ...transactions.TransactionTemplate) ledger.TransactionGroup {
	t.Helper()
	inputs, err := transactions.ResolveTransactions(t.Context(), transactions.ResolverDependencies{
		AccountService: env.Deps.ResolversService, AccountCatalog: env.Deps.AccountService, BalanceQuerier: env.Deps.HistoricalLedger,
	}, transactions.ResolutionScope{CustomerID: env.CustomerID, Namespace: env.Namespace}, templates...)
	require.NoError(t, err)
	group, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(env.Namespace, nil, inputs...))
	require.NoError(t, err)

	return group
}

func consumeAndCommit(t *testing.T, env *ledgertestutils.IntegrationEnv, service fbo.Service, input fbo.ConsumeInput) ledger.TransactionGroup {
	t.Helper()
	group, err := transaction.Run(t.Context(), enttx.NewCreator(env.DB), func(ctx context.Context) (ledger.TransactionGroup, error) {
		inputs, err := service.Consume(ctx, input)
		if err != nil || len(inputs) == 0 {
			return nil, err
		}

		return env.Deps.HistoricalLedger.CommitGroup(ctx, ledger.WithGroupID(transactions.GroupInputs(env.Namespace, input.Annotations, inputs...), input.TransactionGroupID))
	})
	require.NoError(t, err)

	return group
}
