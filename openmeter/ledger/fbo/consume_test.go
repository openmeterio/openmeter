package fbo_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	currencytestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	advancetestutils "github.com/openmeterio/openmeter/openmeter/ledger/advance/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	breakageadapter "github.com/openmeterio/openmeter/openmeter/ledger/breakage/adapter"
	"github.com/openmeterio/openmeter/openmeter/ledger/fbo"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	omtestutils "github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func TestConsumptionKeepsSameCodeCurrencyBreakageSeparate(t *testing.T) {
	env, service, breakageService := newFBOEnv(t)
	env.Currency = currencyx.Code("ACME")
	alpha := currencytestutils.NewCustomCurrency(t, env.Currency, 2)
	beta := currencytestutils.NewCustomCurrency(t, env.Currency, 2)

	// given: only beta's credit expires, although both currencies have the same code.
	env.CustomCurrency = &beta
	bookExpiringCredit(t, env, breakageService, 10)
	env.CustomCurrency = &alpha
	fundCredit(t, env, 10)

	err := transaction.RunWithNoValue(t.Context(), enttx.NewCreator(env.DB), func(ctx context.Context) error {
		// when: alpha is consumed through the FBO service.
		input := consumeInput(env, "alpha-consumption", 6)
		inputs, err := service.Consume(ctx, input)
		require.NoError(t, err)

		// then: beta's breakage plan is neither attached nor released.
		require.Len(t, inputs, 1)
		plans, err := breakageService.ListPlans(ctx, breakage.ListPlansInput{
			CustomerID: env.CustomerID,
			Currency:   env.Currency,
			AsOf:       env.Now(),
		})
		require.NoError(t, err)
		require.Len(t, plans, 1)
		require.Equal(t, 10.0, plans[0].OpenAmount.InexactFloat64())
		groupInput := ledger.WithGroupID(transactions.GroupInputs(env.Namespace, nil, inputs...), input.TransactionGroupID)
		_, err = env.Deps.HistoricalLedger.CommitGroup(ctx, groupInput)

		return err
	})
	require.NoError(t, err)
	requireAccountAmount(t, env, env.CustomerAccounts.FBOAccount, 4)
	env.CustomCurrency = &beta
	balance, err := env.Deps.HistoricalLedger.GetAccountBalance(t.Context(), env.CustomerAccounts.FBOAccount,
		ledger.RouteFilter{Currency: env.CurrencyReference()}, ledger.BalanceQuery{AsOf: lo.ToPtr(env.Now())})
	require.NoError(t, err)
	require.Equal(t, 10.0, balance.InexactFloat64())
}

func TestConsumptionComposesBreakageAndAdvanceInOneGroup(t *testing.T) {
	env, service, breakageService := newFBOEnv(t)
	bookExpiringCredit(t, env, breakageService, 6)

	// given: expiring credit covers only part of credit-only consumption.
	err := transaction.RunWithNoValue(t.Context(), enttx.NewCreator(env.DB), func(ctx context.Context) error {
		input := consumeInput(env, "credit-only", 10)
		input.AdvancePolicy = fbo.AdvancePolicyOnShortfall

		// when: consumption is planned without posting individual accounting legs.
		inputs, err := service.Consume(ctx, input)
		require.NoError(t, err)
		require.Len(t, inputs, 4)

		var sourceEntry ledger.EntryInput
		for _, entry := range inputs[0].EntryInputs() {
			if entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO && entry.Amount().IsNegative() {
				sourceEntry = entry
			}
		}

		require.NotNil(t, sourceEntry)
		require.Equal(t, -6.0, sourceEntry.Amount().InexactFloat64())

		releases, err := breakageService.ListReleases(ctx, breakage.ListReleasesInput{
			CustomerID:    env.CustomerID,
			SourceEntryID: []string{sourceEntry.AssignedID()},
		})
		require.NoError(t, err)
		require.Len(t, releases, 1)
		require.Equal(t, inputs[0].AssignedID(), *releases[0].SourceTransactionID)
		require.Equal(t, input.TransactionGroupID, releases[0].BreakageTransactionGroupID)

		// then: consumption, release, and advance legs are committed in one caller-owned group.
		groupInput := ledger.WithGroupID(transactions.GroupInputs(env.Namespace, nil, inputs...), input.TransactionGroupID)
		group, err := env.Deps.HistoricalLedger.CommitGroup(ctx, groupInput)
		require.NoError(t, err)
		require.Len(t, group.Transactions(), 4)

		return nil
	})
	require.NoError(t, err)
	requireAccountAmount(t, env, env.CustomerAccounts.FBOAccount, 0)
	requireAccountAmount(t, env, env.CustomerAccounts.AccruedAccount, 10)
}

func TestConsumptionRollbackDiscardsJournalAndBreakageRelease(t *testing.T) {
	env, service, breakageService := newFBOEnv(t)
	bookExpiringCredit(t, env, breakageService, 10)
	failure := errors.New("caller failed")
	var groupID string

	// given: an expiring consume and its bookkeeping share the caller transaction.
	err := transaction.RunWithNoValue(t.Context(), enttx.NewCreator(env.DB), func(ctx context.Context) error {
		input := consumeInput(env, "rollback", 4)
		groupID = input.TransactionGroupID
		inputs, err := service.Consume(ctx, input)
		require.NoError(t, err)
		groupInput := ledger.WithGroupID(transactions.GroupInputs(env.Namespace, nil, inputs...), input.TransactionGroupID)
		_, err = env.Deps.HistoricalLedger.CommitGroup(ctx, groupInput)
		require.NoError(t, err)

		// when: the caller fails after the nested ledger commit.
		return failure
	})
	require.ErrorIs(t, err, failure)

	// then: neither the group nor its breakage bookkeeping survives.
	count, err := env.DB.LedgerTransactionGroup.Query().Count(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, count)
	releases, err := breakageService.ListReleases(t.Context(), breakage.ListReleasesInput{
		CustomerID:               env.CustomerID,
		SourceTransactionGroupID: []string{groupID},
	})
	require.NoError(t, err)
	require.Empty(t, releases)
	plans, err := breakageService.ListPlans(t.Context(), breakage.ListPlansInput{
		CustomerID: env.CustomerID,
		Currency:   env.Currency,
		AsOf:       env.Now(),
	})
	require.NoError(t, err)
	require.Len(t, plans, 1)
	require.Equal(t, 10.0, plans[0].OpenAmount.InexactFloat64())
}

func newFBOEnv(t *testing.T) (*ledgertestutils.IntegrationEnv, fbo.Service, breakage.Service) {
	t.Helper()
	env := ledgertestutils.NewIntegrationEnv(t, "fbo-consumption")
	adapter, err := breakageadapter.New(breakageadapter.Config{Client: env.DB})
	require.NoError(t, err)
	dependencies := transactions.ResolverDependencies{
		AccountService: env.Deps.ResolversService,
		AccountCatalog: env.Deps.AccountService,
		BalanceQuerier: env.Deps.HistoricalLedger,
	}
	breakageService, err := breakage.NewService(breakage.Config{Adapter: adapter, Dependencies: dependencies})
	require.NoError(t, err)
	service, err := fbo.NewService(fbo.Config{
		Logger:        omtestutils.NewDiscardLogger(t),
		Dependencies:  dependencies,
		Advance:       advancetestutils.NewService(t, env.Deps, breakageService),
		Breakage:      breakageService,
		AccountLocker: env.Deps.AccountService,
	})
	require.NoError(t, err)

	return env, service, breakageService
}

func consumeInput(env *ledgertestutils.IntegrationEnv, chargeID string, amount int64) fbo.ConsumeInput {
	return fbo.ConsumeInput{
		CustomerID:    env.CustomerID,
		PostingInput:  breakage.PostingInput{TransactionGroupID: ulid.Make().String()},
		ChargeID:      chargeID,
		BookedAt:      env.Now(),
		Currency:      env.CurrencyReference(),
		Amount:        alpacadecimal.NewFromInt(amount),
		Destination:   fbo.ConsumeDestinationAccrued,
		AdvancePolicy: fbo.AdvancePolicyNever,
	}
}

func fundCredit(t *testing.T, env *ledgertestutils.IntegrationEnv, amount int64) {
	t.Helper()
	inputs, err := transactions.ResolveTransactions(t.Context(), transactions.ResolverDependencies{
		AccountService: env.Deps.ResolversService,
		AccountCatalog: env.Deps.AccountService,
		BalanceQuerier: env.Deps.HistoricalLedger,
	}, transactions.ResolutionScope{CustomerID: env.CustomerID, Namespace: env.Namespace}, transactions.IssueCustomerReceivableTemplate{
		At:       env.Now(),
		Amount:   alpacadecimal.NewFromInt(amount),
		Currency: env.CurrencyReference(),
	})
	require.NoError(t, err)
	_, err = env.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(env.Namespace, nil, inputs...))
	require.NoError(t, err)
}

func bookExpiringCredit(t *testing.T, env *ledgertestutils.IntegrationEnv, breakageService breakage.Service, amount int64) {
	t.Helper()
	groupID := ulid.Make().String()
	err := transaction.RunWithNoValue(t.Context(), enttx.NewCreator(env.DB), func(ctx context.Context) error {
		inputs, err := transactions.ResolveTransactions(ctx, transactions.ResolverDependencies{
			AccountService: env.Deps.ResolversService,
			AccountCatalog: env.Deps.AccountService,
			BalanceQuerier: env.Deps.HistoricalLedger,
		}, transactions.ResolutionScope{CustomerID: env.CustomerID, Namespace: env.Namespace}, transactions.IssueCustomerReceivableTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(amount),
			Currency: env.CurrencyReference(),
		})
		if err != nil {
			return err
		}

		breakageInputs, err := breakageService.PlanIssuance(ctx, breakage.PlanIssuanceInput{
			PostingInput: breakage.PostingInput{TransactionGroupID: groupID},
			CustomerID:   env.CustomerID,
			Amount:       alpacadecimal.NewFromInt(amount),
			Currency:     env.CurrencyReference(),
			ExpiresAt:    env.Now().Add(10 * 24 * time.Hour),
		})
		if err != nil {
			return err
		}

		inputs = append(inputs, breakageInputs...)
		_, err = env.Deps.HistoricalLedger.CommitGroup(ctx, ledger.WithGroupID(transactions.GroupInputs(env.Namespace, nil, inputs...), groupID))

		return err
	})
	require.NoError(t, err)
}

func requireAccountAmount(t *testing.T, env *ledgertestutils.IntegrationEnv, account ledger.Account, expected float64) {
	t.Helper()
	balance, err := env.Deps.HistoricalLedger.GetAccountBalance(t.Context(), account, ledger.RouteFilter{Currency: env.CurrencyReference()}, ledger.BalanceQuery{})
	require.NoError(t, err)
	require.Equal(t, expected, balance.InexactFloat64())
}
