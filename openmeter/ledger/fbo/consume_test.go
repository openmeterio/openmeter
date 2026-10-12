package fbo_test

import (
	"context"
	"errors"
	"slices"
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

func TestPlanReservesSourcesAcrossPostingTimes(t *testing.T) {
	env, service, _ := newFBOEnv(t)
	fundCredit(t, env, 10)

	// given: both consumptions select the same historical balance but post later.
	err := transaction.RunWithNoValue(t.Context(), enttx.NewCreator(env.DB), func(ctx context.Context) error {
		plan, err := service.NewPlan(fbo.NewPlanInput{CustomerID: env.CustomerID})
		require.NoError(t, err)
		firstInput := consumeInput(env, "first", 7)
		firstInput.BookedAt = env.Now().Add(24 * time.Hour)
		first, err := plan.Consume(ctx, firstInput)
		require.NoError(t, err)
		require.Equal(t, 7.0, first.CoveredAmount.InexactFloat64())

		// when: a second consume is planned before the first is posted.
		secondInput := consumeInput(env, "second", 7)
		secondInput.BookedAt = env.Now().Add(48 * time.Hour)
		second, err := plan.Consume(ctx, secondInput)
		require.NoError(t, err)

		// then: only the unreserved slice is available in the shared group.
		require.Equal(t, 3.0, second.CoveredAmount.InexactFloat64())
		require.Equal(t, 4.0, second.UncoveredAmount.InexactFloat64())
		require.NotEqual(t, first.Sources[0].EntryID, second.Sources[0].EntryID)
		companion, err := transactions.ResolveTransactions(ctx, transactions.ResolverDependencies{
			AccountService: env.Deps.ResolversService,
			AccountCatalog: env.Deps.AccountService,
			BalanceQuerier: env.Deps.HistoricalLedger,
		}, transactions.ResolutionScope{CustomerID: env.CustomerID, Namespace: env.Namespace}, transactions.TransferCustomerReceivableToAccruedTemplate{
			At:            secondInput.BookedAt,
			Amount:        second.UncoveredAmount,
			Currency:      env.CurrencyReference(),
			CostBasis:     lo.ToPtr(alpacadecimal.NewFromInt(1)),
			SpendChargeID: &secondInput.ChargeID,
		})
		require.NoError(t, err)
		groupInput, err := plan.Finalize(nil, companion...)
		require.NoError(t, err)
		require.Len(t, groupInput.Transactions(), 3)
		_, err = plan.Consume(ctx, consumeInput(env, "after-finalization", 1))
		require.ErrorContains(t, err, "finalized")
		_, err = plan.Finalize(nil)
		require.ErrorContains(t, err, "finalized")
		group, err := env.Deps.HistoricalLedger.CommitGroup(ctx, groupInput)
		require.NoError(t, err)
		firstPosted, err := first.Transactions(group)
		require.NoError(t, err)
		require.Len(t, firstPosted, 1)
		require.Equal(t, first.Sources[0].TransactionID, firstPosted[0].ID().ID)
		secondPosted, err := second.Transactions(group)
		require.NoError(t, err)
		require.Len(t, secondPosted, 1)
		require.Equal(t, second.Sources[0].TransactionID, secondPosted[0].ID().ID)
		missing := fbo.Consumption{TransactionIDs: []string{ulid.Make().String()}}
		_, err = missing.Transactions(group)
		require.ErrorContains(t, err, "is missing from group")

		return nil
	})
	require.NoError(t, err)
	requireAccountAmount(t, env, env.CustomerAccounts.FBOAccount, 0)
	requireAccountAmount(t, env, env.CustomerAccounts.AccruedAccount, 14)
	requireAccountAmount(t, env, env.CustomerAccounts.ReceivableAccount, -14)
}

func TestPlanReservesExpiringSourcesBeforePosting(t *testing.T) {
	env, service, breakageService := newFBOEnv(t)
	bookExpiringCredit(t, env, breakageService, 10)

	// given: two consumes share one expiring source and the same unposted group.
	err := transaction.RunWithNoValue(t.Context(), enttx.NewCreator(env.DB), func(ctx context.Context) error {
		plan, err := service.NewPlan(fbo.NewPlanInput{CustomerID: env.CustomerID})
		require.NoError(t, err)
		first, err := plan.Consume(ctx, consumeInput(env, "first-expiring", 6))
		require.NoError(t, err)
		require.Equal(t, 6.0, first.CoveredAmount.InexactFloat64())

		// when: the next consume sees the reservation and the staged breakage release.
		plans, err := breakageService.ListPlans(ctx, breakage.ListPlansInput{
			CustomerID: env.CustomerID,
			Currency:   env.Currency,
			AsOf:       env.Now(),
		})
		require.NoError(t, err)
		require.Len(t, plans, 1)
		require.Equal(t, 4.0, plans[0].OpenAmount.InexactFloat64())
		second, err := plan.Consume(ctx, consumeInput(env, "second-expiring", 6))
		require.NoError(t, err)

		// then: each release names its own consumption entry, without reusing credit.
		require.Equal(t, 4.0, second.CoveredAmount.InexactFloat64())
		require.Equal(t, 2.0, second.UncoveredAmount.InexactFloat64())
		require.Len(t, second.Sources, 1)
		require.Equal(t, first.Sources[0].BreakagePlanID, second.Sources[0].BreakagePlanID)
		releases, err := breakageService.ListReleases(ctx, breakage.ListReleasesInput{
			CustomerID:    env.CustomerID,
			SourceEntryID: []string{first.Sources[0].EntryID, second.Sources[0].EntryID},
		})
		require.NoError(t, err)
		require.Len(t, releases, 2)
		groupInput, err := plan.Finalize(nil)
		require.NoError(t, err)
		require.Len(t, groupInput.Transactions(), 4)
		_, err = env.Deps.HistoricalLedger.CommitGroup(ctx, groupInput)

		return err
	})
	require.NoError(t, err)
	requireAccountAmount(t, env, env.CustomerAccounts.FBOAccount, 0)
	requireAccountAmount(t, env, env.CustomerAccounts.AccruedAccount, 10)
}

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
		plan, err := service.NewPlan(fbo.NewPlanInput{CustomerID: env.CustomerID})
		require.NoError(t, err)

		// when: alpha is consumed through the FBO service.
		consumption, err := plan.Consume(ctx, consumeInput(env, "alpha-consumption", 6))
		require.NoError(t, err)

		// then: beta's breakage plan is neither attached nor released.
		require.Equal(t, 6.0, consumption.CoveredAmount.InexactFloat64())
		require.Len(t, consumption.Sources, 1)
		require.Nil(t, consumption.Sources[0].BreakagePlanID)
		plans, err := breakageService.ListPlans(ctx, breakage.ListPlansInput{
			CustomerID: env.CustomerID,
			Currency:   env.Currency,
			AsOf:       env.Now(),
		})
		require.NoError(t, err)
		require.Len(t, plans, 1)
		require.Equal(t, 10.0, plans[0].OpenAmount.InexactFloat64())
		groupInput, err := plan.Finalize(nil)
		require.NoError(t, err)
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
		plan, err := service.NewPlan(fbo.NewPlanInput{CustomerID: env.CustomerID})
		require.NoError(t, err)
		input := consumeInput(env, "credit-only", 10)
		input.AdvancePolicy = fbo.AdvancePolicyOnShortfall

		// when: consumption is planned without posting individual accounting legs.
		consumption, err := plan.Consume(ctx, input)
		require.NoError(t, err)
		require.Equal(t, 6.0, consumption.CoveredAmount.InexactFloat64())
		require.Equal(t, 4.0, consumption.AdvanceAmount.InexactFloat64())
		require.Equal(t, 0.0, consumption.UncoveredAmount.InexactFloat64())
		require.Len(t, consumption.TransactionIDs, 4)
		releases, err := breakageService.ListReleases(ctx, breakage.ListReleasesInput{
			CustomerID:    env.CustomerID,
			SourceEntryID: []string{consumption.Sources[0].EntryID},
		})
		require.NoError(t, err)
		require.Len(t, releases, 1)
		require.Equal(t, consumption.Sources[0].TransactionID, *releases[0].SourceTransactionID)
		require.Equal(t, plan.GroupID(), releases[0].BreakageTransactionGroupID)

		// then: all legs share a group, while the result retains planning order
		// even when the caller enumerates the group's transactions differently.
		groupInput, err := plan.Finalize(nil)
		require.NoError(t, err)
		inputs := groupInput.Transactions()
		slices.Reverse(inputs)
		groupInput = ledger.WithGroupID(transactions.GroupInputs(env.Namespace, nil, inputs...), plan.GroupID())
		group, err := env.Deps.HistoricalLedger.CommitGroup(ctx, groupInput)
		require.NoError(t, err)
		require.Len(t, group.Transactions(), 4)
		posted, err := consumption.Transactions(group)
		require.NoError(t, err)
		require.Len(t, posted, 4)
		for idx, transaction := range posted {
			require.Equal(t, consumption.TransactionIDs[idx], transaction.ID().ID)
		}

		return nil
	})
	require.NoError(t, err)
	requireAccountAmount(t, env, env.CustomerAccounts.FBOAccount, 0)
	requireAccountAmount(t, env, env.CustomerAccounts.AccruedAccount, 10)
}

func TestPlanRollbackDiscardsJournalAndBreakageRelease(t *testing.T) {
	env, service, breakageService := newFBOEnv(t)
	bookExpiringCredit(t, env, breakageService, 10)
	failure := errors.New("caller failed")
	var groupID string

	// given: an expiring consume and its bookkeeping share the caller transaction.
	err := transaction.RunWithNoValue(t.Context(), enttx.NewCreator(env.DB), func(ctx context.Context) error {
		plan, err := service.NewPlan(fbo.NewPlanInput{CustomerID: env.CustomerID})
		require.NoError(t, err)
		groupID = plan.GroupID()
		_, err = plan.Consume(ctx, consumeInput(env, "rollback", 4))
		require.NoError(t, err)
		groupInput, err := plan.Finalize(nil)
		require.NoError(t, err)
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

func TestPlanFailureRejectsEarlierConsumptions(t *testing.T) {
	env, service, breakageService := newFBOEnv(t)
	bookExpiringCredit(t, env, breakageService, 10)
	var groupID string

	// given: an earlier consumption has already written its breakage bookkeeping.
	err := transaction.RunWithNoValue(t.Context(), enttx.NewCreator(env.DB), func(ctx context.Context) error {
		plan, err := service.NewPlan(fbo.NewPlanInput{CustomerID: env.CustomerID})
		require.NoError(t, err)
		groupID = plan.GroupID()
		_, err = plan.Consume(ctx, consumeInput(env, "valid", 4))
		require.NoError(t, err)

		// when: a later consume fails in the same plan.
		invalid := consumeInput(env, "invalid", -1)
		invalid.Destination = fbo.ConsumeDestination("unknown")
		invalid.AdvancePolicy = fbo.AdvancePolicy("unknown")
		_, err = plan.Consume(ctx, invalid)
		require.ErrorContains(t, err, "amount cannot be negative")
		require.ErrorContains(t, err, "destination:")
		require.ErrorContains(t, err, "advance policy:")

		// then: the caller cannot drop that failure and commit earlier consumption.
		_, err = plan.Finalize(nil)
		require.ErrorContains(t, err, "FBO plan failed")

		return err
	})
	require.ErrorContains(t, err, "FBO plan failed")
	releases, err := breakageService.ListReleases(t.Context(), breakage.ListReleasesInput{
		CustomerID:               env.CustomerID,
		SourceTransactionGroupID: []string{groupID},
	})
	require.NoError(t, err)
	require.Empty(t, releases)
}

func newFBOEnv(t *testing.T) (*ledgertestutils.IntegrationEnv, fbo.Service, breakage.Service) {
	t.Helper()
	env := ledgertestutils.NewIntegrationEnv(t, "fbo-planning")
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
		ChargeID:          chargeID,
		BookedAt:          env.Now(),
		SourceBalanceAsOf: env.Now(),
		Currency:          env.CurrencyReference(),
		Amount:            alpacadecimal.NewFromInt(amount),
		Destination:       fbo.ConsumeDestinationAccrued,
		AdvancePolicy:     fbo.AdvancePolicyNever,
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
