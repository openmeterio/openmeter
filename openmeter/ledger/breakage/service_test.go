package breakage_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	breakageadapter "github.com/openmeterio/openmeter/openmeter/ledger/breakage/adapter"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestBreakageBookkeepingPrecedesPosting(t *testing.T) {
	env, service := newBreakageEnv(t)
	groupID := ulid.Make().String()

	// given: an expiring issuance whose ledger group has not been posted.
	err := transaction.RunWithNoValue(t.Context(), enttx.NewCreator(env.DB), func(ctx context.Context) error {
		inputs, err := service.PlanIssuance(ctx, expiringIssuance(env, groupID))
		require.NoError(t, err)
		require.Len(t, inputs, 1)

		// when: another operation reads bookkeeping in the same transaction.
		plans, err := service.ListPlans(ctx, breakage.ListPlansInput{
			CustomerID: env.CustomerID,
			Currency:   env.Currency,
			AsOf:       env.Now(),
		})
		require.NoError(t, err)
		require.Len(t, plans, 1)

		// then: the complete reference already exists and names the returned posting.
		require.Equal(t, groupID, plans[0].BreakageTransactionGroupID)
		require.Equal(t, groupID, *plans[0].SourceTransactionGroupID)
		require.Equal(t, inputs[0].AssignedID(), plans[0].BreakageTransactionID)
		require.Equal(t, 10.0, plans[0].OpenAmount.InexactFloat64())
		count, err := env.DB.LedgerTransaction.Query().Count(t.Context())
		require.NoError(t, err)
		require.Zero(t, count)

		issue, err := resolveCreditIssuance(ctx, env)
		if err != nil {
			return err
		}

		inputs = append(issue, inputs...)
		_, err = env.Deps.HistoricalLedger.CommitGroup(ctx, ledger.WithGroupID(transactions.GroupInputs(env.Namespace, nil, inputs...), groupID))

		return err
	})
	require.NoError(t, err)

	group, err := env.Deps.HistoricalLedger.GetTransactionGroup(t.Context(), models.NamespacedID{Namespace: env.Namespace, ID: groupID})
	require.NoError(t, err)
	require.Equal(t, groupID, group.ID().ID)
	require.Len(t, group.Transactions(), 2)
}

func TestBreakageReferencesDistinctPostingsWithIdenticalAccountingIdentity(t *testing.T) {
	env, service := newBreakageEnv(t)
	issueGroupID := ulid.Make().String()

	// given: one expiry and two consumption postings with identical entry identities.
	err := transaction.RunWithNoValue(t.Context(), enttx.NewCreator(env.DB), func(ctx context.Context) error {
		inputs, err := service.PlanIssuance(ctx, expiringIssuance(env, issueGroupID))
		if err != nil {
			return err
		}

		issue, err := resolveCreditIssuance(ctx, env)
		if err != nil {
			return err
		}

		inputs = append(issue, inputs...)
		_, err = env.Deps.HistoricalLedger.CommitGroup(ctx, ledger.WithGroupID(transactions.GroupInputs(env.Namespace, nil, inputs...), issueGroupID))

		return err
	})
	require.NoError(t, err)

	groupID := ulid.Make().String()
	var sourceEntries []ledger.EntryInput
	var sourceTransactions []string
	err = transaction.RunWithNoValue(t.Context(), enttx.NewCreator(env.DB), func(ctx context.Context) error {
		plans, err := service.ListPlans(ctx, breakage.ListPlansInput{CustomerID: env.CustomerID, Currency: env.Currency, AsOf: env.Now()})
		require.NoError(t, err)
		require.Len(t, plans, 1)

		template := transactions.TransferCustomerFBOToAccruedTemplate{
			At:       env.Now(),
			Currency: env.CurrencyReference(),
			Sources: []transactions.PostingAmount{{
				Address: plans[0].FBOAddress,
				Amount:  alpacadecimal.NewFromInt(2),
			}},
		}
		inputs, err := transactions.ResolveTransactions(ctx, transactions.ResolverDependencies{
			AccountService: env.Deps.ResolversService,
			AccountCatalog: env.Deps.AccountService,
			BalanceQuerier: env.Deps.HistoricalLedger,
		}, transactions.ResolutionScope{Namespace: env.Namespace, CustomerID: env.CustomerID}, template, template)
		require.NoError(t, err)

		for idx := range inputs {
			inputs[idx], err = ledger.PreassignIDs(inputs[idx])
			require.NoError(t, err)
			for _, entry := range inputs[idx].EntryInputs() {
				if entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO {
					sourceEntries = append(sourceEntries, entry)
					sourceTransactions = append(sourceTransactions, inputs[idx].AssignedID())
				}
			}
		}

		require.Len(t, sourceEntries, 2)
		require.Equal(t, sourceEntries[0].IdentityKey(), sourceEntries[1].IdentityKey())

		// when: each debit releases its expiry before any journal posting is committed.
		for idx, entry := range sourceEntries {
			entryID := entry.AssignedID()
			release, err := service.ReleasePlan(ctx, breakage.ReleasePlanInput{
				PostingInput:        breakage.PostingInput{TransactionGroupID: groupID},
				Plan:                plans[0],
				Amount:              alpacadecimal.NewFromInt(2),
				SourceKind:          breakage.SourceKindUsage,
				SourceTransactionID: &sourceTransactions[idx],
				SourceEntryID:       &entryID,
			})
			require.NoError(t, err)
			inputs = append(inputs, release)
		}

		// then: subsequent expiry planning sees both staged releases.
		plans, err = service.ListPlans(ctx, breakage.ListPlansInput{CustomerID: env.CustomerID, Currency: env.Currency, AsOf: env.Now()})
		require.NoError(t, err)
		require.Len(t, plans, 1)
		require.Equal(t, 6.0, plans[0].OpenAmount.InexactFloat64())

		_, err = env.Deps.HistoricalLedger.CommitGroup(ctx, ledger.WithGroupID(transactions.GroupInputs(env.Namespace, nil, inputs...), groupID))

		return err
	})
	require.NoError(t, err)

	releases, err := service.ListReleases(t.Context(), breakage.ListReleasesInput{
		CustomerID:               env.CustomerID,
		SourceTransactionGroupID: []string{groupID},
	})
	require.NoError(t, err)
	require.Len(t, releases, 2)
	for idx, entry := range sourceEntries {
		var matched *breakage.Release
		for releaseIdx := range releases {
			if *releases[releaseIdx].SourceEntryID == entry.AssignedID() {
				matched = &releases[releaseIdx]
				break
			}
		}

		require.NotNil(t, matched)
		require.Equal(t, sourceTransactions[idx], *matched.SourceTransactionID)
	}
}

func TestBreakageDiscardedPostingRejectsOuterCommit(t *testing.T) {
	env, service := newBreakageEnv(t)

	// given: bookkeeping names a group that will never be posted.
	err := transaction.RunWithNoValue(t.Context(), enttx.NewCreator(env.DB), func(ctx context.Context) error {
		_, err := service.PlanIssuance(ctx, expiringIssuance(env, ulid.Make().String()))
		require.NoError(t, err)

		// when: the caller attempts to commit only bookkeeping.
		return nil
	})

	// then: deferred integrity checks reject the commit and discard the record.
	require.Error(t, err)
	requireNoBreakageRows(t, env)
}

func TestBreakageBookkeepingRollsBackWithCaller(t *testing.T) {
	env, service := newBreakageEnv(t)
	failure := errors.New("caller failed")

	// given: a plan has been staged in the caller's transaction.
	err := transaction.RunWithNoValue(t.Context(), enttx.NewCreator(env.DB), func(ctx context.Context) error {
		_, err := service.PlanIssuance(ctx, expiringIssuance(env, ulid.Make().String()))
		require.NoError(t, err)

		// when: the surrounding business operation fails before posting.
		return failure
	})

	// then: no bookkeeping survives independently of the operation.
	require.ErrorIs(t, err, failure)
	requireNoBreakageRows(t, env)
}

func TestBreakagePostingFailureRollsBackBookkeeping(t *testing.T) {
	env, service := newBreakageEnv(t)
	groupID := ulid.Make().String()
	issue, err := resolveCreditIssuance(t.Context(), env)
	require.NoError(t, err)
	_, err = env.Deps.HistoricalLedger.CommitGroup(t.Context(), ledger.WithGroupID(transactions.GroupInputs(env.Namespace, nil, issue...), groupID))
	require.NoError(t, err)

	// given: bookkeeping is written before posting a group with a colliding ID.
	err = transaction.RunWithNoValue(t.Context(), enttx.NewCreator(env.DB), func(ctx context.Context) error {
		inputs, err := service.PlanIssuance(ctx, expiringIssuance(env, groupID))
		require.NoError(t, err)

		// when: the ledger rejects the posting.
		_, err = env.Deps.HistoricalLedger.CommitGroup(ctx, ledger.WithGroupID(transactions.GroupInputs(env.Namespace, nil, inputs...), groupID))

		return err
	})

	// then: bookkeeping is rolled back and the original credit remains intact.
	require.Error(t, err)
	count, err := env.DB.LedgerBreakageRecord.Query().Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, count)
	require.Equal(t, 10.0, env.SumBalance(t, env.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority)).InexactFloat64())
}

func TestBreakageRequiresCallerTransaction(t *testing.T) {
	env, service := newBreakageEnv(t)

	inputs, err := service.PlanIssuance(t.Context(), expiringIssuance(env, ulid.Make().String()))
	require.ErrorContains(t, err, "requires the caller's transaction")
	require.Nil(t, inputs)
	requireNoBreakageRows(t, env)
}

func newBreakageEnv(t *testing.T) (*ledgertestutils.IntegrationEnv, breakage.Service) {
	t.Helper()

	env := ledgertestutils.NewIntegrationEnv(t, "breakage-posting")
	adapter, err := breakageadapter.New(breakageadapter.Config{Client: env.DB})
	require.NoError(t, err)
	service, err := breakage.NewService(breakage.Config{
		Adapter: adapter,
		Dependencies: transactions.ResolverDependencies{
			AccountService: env.Deps.ResolversService,
			AccountCatalog: env.Deps.AccountService,
			BalanceQuerier: env.Deps.HistoricalLedger,
		},
	})
	require.NoError(t, err)

	return env, service
}

func expiringIssuance(env *ledgertestutils.IntegrationEnv, groupID string) breakage.PlanIssuanceInput {
	return breakage.PlanIssuanceInput{
		PostingInput: breakage.PostingInput{TransactionGroupID: groupID},
		CustomerID:   env.CustomerID,
		Amount:       alpacadecimal.NewFromInt(10),
		Currency:     env.CurrencyReference(),
		ExpiresAt:    env.Now().Add(24 * time.Hour),
	}
}

func resolveCreditIssuance(ctx context.Context, env *ledgertestutils.IntegrationEnv) ([]ledger.TransactionInput, error) {
	return transactions.ResolveTransactions(ctx, transactions.ResolverDependencies{
		AccountService: env.Deps.ResolversService,
		AccountCatalog: env.Deps.AccountService,
		BalanceQuerier: env.Deps.HistoricalLedger,
	}, transactions.ResolutionScope{Namespace: env.Namespace, CustomerID: env.CustomerID}, transactions.IssueCustomerReceivableTemplate{
		At:       env.Now(),
		Amount:   alpacadecimal.NewFromInt(10),
		Currency: env.CurrencyReference(),
	})
}

func requireNoBreakageRows(t *testing.T, env *ledgertestutils.IntegrationEnv) {
	t.Helper()

	count, err := env.DB.LedgerBreakageRecord.Query().Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, count)
	count, err = env.DB.LedgerTransactionGroup.Query().Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, count)
}
