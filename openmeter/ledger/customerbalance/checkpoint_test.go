package customerbalance

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func TestJournalCheckpointRecalculatesCurrentCharges(t *testing.T) {
	// Given funded credits and a charge that can use at most their spendable amount.
	env := newTestEnv(t)
	ctx := t.Context()
	env.bookFBOBalance(t, alpacadecimal.NewFromInt(100))
	env.fundOpenReceivable(t, alpacadecimal.NewFromInt(100))
	env.createUsageBasedCharge(t, alpacadecimal.NewFromInt(1), productcatalog.CreditThenInvoiceSettlementMode, env.sp())
	env.addUsage(25, env.Now().Add(-20*time.Minute))

	input := GetIncrementalBalanceInput{GetBalanceServiceInput: GetBalanceServiceInput{
		CustomerID: env.CustomerID, Currency: env.CurrencyReference(),
	}}
	first, err := env.Service.GetIncrementalBalance(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, first.Checkpoint)
	require.Equal(t, float64(75), first.Balance.Live().InexactFloat64())

	// When additional usage predates the last read and a new pending grant is created.
	data, err := json.Marshal(first.Checkpoint)
	require.NoError(t, err)
	var stored JournalCheckpoint
	require.NoError(t, json.Unmarshal(data, &stored))
	input.Checkpoints = []JournalCheckpoint{stored}
	env.addUsage(30, env.Now().Add(-25*time.Minute))
	env.createPendingInvoiceCreditGrant(t, alpacadecimal.NewFromInt(10), env.Currency)

	// Then the journal is reusable across serialization. Charge inputs are read again.
	result, err := env.Service.GetIncrementalBalance(ctx, input)
	require.NoError(t, err)
	require.True(t, result.Reused)
	require.Nil(t, result.Checkpoint, "a reuse keeps the original journal boundary")
	require.Equal(t, float64(45), result.Balance.Live().InexactFloat64())
	require.Equal(t, float64(10), result.Balance.Pending().InexactFloat64())

	full, err := env.Service.GetBalance(ctx, input.GetBalanceServiceInput)
	require.NoError(t, err)
	require.Equal(t, full, result.Balance)

	// A sibling key on the same accounts needs its own eligible-source projection.
	input.FeatureFilter = NewUnrestrictedFeatureFilter()
	other, err := env.Service.GetIncrementalBalance(ctx, input)
	require.NoError(t, err)
	require.False(t, other.Reused)
}

func TestJournalCheckpointExpiresAtFutureBooking(t *testing.T) {
	// Given an already recorded funding group which becomes effective an hour later.
	env := newTestEnv(t)
	ctx := t.Context()
	now := env.Now()
	future := now.Add(time.Hour)
	clock.FreezeTime(future)
	defer clock.UnFreeze()
	env.bookFBOBalance(t, alpacadecimal.NewFromInt(50))
	env.fundOpenReceivable(t, alpacadecimal.NewFromInt(50))

	clock.FreezeTime(now)
	defer clock.UnFreeze()
	input := GetIncrementalBalanceInput{GetBalanceServiceInput: GetBalanceServiceInput{
		CustomerID: env.CustomerID, Currency: env.CurrencyReference(),
	}}
	first, err := env.Service.GetIncrementalBalance(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, first.Checkpoint)
	require.Equal(t, float64(0), first.Balance.Booked().InexactFloat64())
	require.NotNil(t, first.Checkpoint.NextBooking)
	require.True(t, future.Equal(*first.Checkpoint.NextBooking))

	input.Checkpoints = []JournalCheckpoint{*first.Checkpoint}

	// When the clock reaches the booking, entry counts still match but balances change.
	clock.FreezeTime(future)
	defer clock.UnFreeze()
	current, err := env.Service.GetIncrementalBalance(ctx, input)
	require.NoError(t, err)
	require.True(t, current.Reused, "the opening amounts advance through the scheduled booking")
	require.NotNil(t, current.Checkpoint)
	require.Equal(t, float64(50), current.Balance.Booked().InexactFloat64())
	full, err := env.Service.GetBalance(ctx, input.GetBalanceServiceInput)
	require.NoError(t, err)
	require.Equal(t, full, current.Balance)
}

func TestJournalCheckpointUsesCalculationStart(t *testing.T) {
	env := newTestEnv(t)
	start := env.Now()

	// The posting is already recorded when the ledger read runs, but becomes
	// effective after this calculation's cutoff. Live charges still count.
	clock.FreezeTime(start.Add(time.Minute))
	defer clock.UnFreeze()
	env.bookFBOBalance(t, alpacadecimal.NewFromInt(50))
	env.fundOpenReceivable(t, alpacadecimal.NewFromInt(50))
	env.createFlatFeeCharge(t, alpacadecimal.NewFromInt(10), productcatalog.CreditOnlySettlementMode, env.sp())

	result, err := env.Service.GetIncrementalBalance(t.Context(), GetIncrementalBalanceInput{
		GetBalanceServiceInput: GetBalanceServiceInput{CustomerID: env.CustomerID, Currency: env.CurrencyReference()},
		StartedAt:              start,
	})
	require.NoError(t, err)
	require.NotNil(t, result.Checkpoint)
	require.True(t, start.Equal(result.Checkpoint.AsOf))
	require.Equal(t, float64(0), result.Balance.Booked().InexactFloat64())
	require.Equal(t, float64(-10), result.Balance.Live().InexactFloat64())
}

func TestJournalCheckpointResumesBeforeLatePosting(t *testing.T) {
	env := newTestEnv(t)
	ctx := t.Context()
	start := env.Now()
	input := GetIncrementalBalanceInput{GetBalanceServiceInput: GetBalanceServiceInput{
		CustomerID: env.CustomerID, Currency: env.CurrencyReference(),
	}}
	fund := func(n int64) {
		env.bookFBOBalance(t, alpacadecimal.NewFromInt(n))
		env.fundOpenReceivable(t, alpacadecimal.NewFromInt(n))
	}
	assertBalance := func(want float64, reused bool) JournalCheckpoint {
		t.Helper()
		result, err := env.Service.GetIncrementalBalance(ctx, input)
		require.NoError(t, err)
		require.Equal(t, reused, result.Reused)
		require.NotNil(t, result.Checkpoint)
		require.Equal(t, want, result.Balance.Booked().InexactFloat64())
		full, err := env.Service.GetBalance(ctx, input.GetBalanceServiceInput)
		require.NoError(t, err)
		require.Equal(t, full, result.Balance)

		return *result.Checkpoint
	}

	// Given an opening balance of 100 and another 20 booked an hour later.
	fund(100)
	first := assertBalance(100, false)
	clock.FreezeTime(start.Add(time.Hour))
	defer clock.UnFreeze()
	fund(20)
	input.Checkpoints = []JournalCheckpoint{first}
	second := assertBalance(120, true)

	// A later commit backdates a posting into the interval between the two
	// checkpoints. Only the first opening is still valid.
	clock.FreezeTime(start.Add(30 * time.Minute))
	defer clock.UnFreeze()
	fund(5)
	clock.FreezeTime(start.Add(2 * time.Hour))
	defer clock.UnFreeze()
	input.Checkpoints = []JournalCheckpoint{second, first}
	assertBalance(125, true)

	// A new posting at the first boundary belongs to its prefix as well.
	clock.FreezeTime(start)
	defer clock.UnFreeze()
	fund(1)
	clock.FreezeTime(start.Add(2 * time.Hour))
	defer clock.UnFreeze()
	assertBalance(126, false)
}

func TestJournalCheckpointPreservesSpendableSources(t *testing.T) {
	env := newTestEnv(t)
	ctx := t.Context()
	start := env.Now()
	input := GetIncrementalBalanceInput{GetBalanceServiceInput: GetBalanceServiceInput{
		CustomerID: env.CustomerID, Currency: env.CurrencyReference(),
	}}
	fund := func(n int64, priority int, features ...string) {
		amount := alpacadecimal.NewFromInt(n)
		inputs, err := transactions.ResolveTransactions(ctx, transactions.ResolverDependencies{
			AccountService: env.Deps.ResolversService, AccountCatalog: env.Deps.AccountService, BalanceQuerier: env.Deps.HistoricalLedger,
		}, transactions.ResolutionScope{CustomerID: env.CustomerID, Namespace: env.Namespace}, transactions.IssueCustomerReceivableTemplate{
			At: env.Now(), Currency: env.CurrencyReference(), Amount: amount,
			CreditPriority: &priority, Filters: ledger.CreditFilters{Features: features},
		})
		require.NoError(t, err)
		_, err = env.Deps.HistoricalLedger.CommitGroup(ctx, transactions.GroupInputs(env.Namespace, nil, inputs...))
		require.NoError(t, err)
		env.fundOpenReceivableWithFeatures(t, amount, features)
	}

	// Given two sources. Only the unrestricted one will receive later bookings.
	fund(40, 200, "api")
	fund(20, ledger.DefaultCustomerFBOPriority)
	opening, err := env.Service.GetIncrementalBalance(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, opening.Checkpoint)
	input.Checkpoints = []JournalCheckpoint{*opening.Checkpoint}

	// A new restricted source is ahead of the unrestricted one in collection order.
	// Future usage reserves five now. The three-credit expiry counts only at its booking.
	clock.FreezeTime(start.Add(time.Hour))
	defer clock.UnFreeze()
	fund(10, 10, testFeatureKey)
	env.createFlatFeeCharge(t, alpacadecimal.NewFromInt(30), productcatalog.CreditThenInvoiceSettlementMode, env.sp(), testFeatureKey)
	env.bookFBOUsage(t, start.Add(2*time.Hour), env.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority).Address(), alpacadecimal.NewFromInt(5))
	env.bookBreakageIssuance(t, breakage.PlanIssuanceInput{
		CustomerID: env.CustomerID, Currency: env.CurrencyReference(),
		Amount: alpacadecimal.NewFromInt(3), ExpiresAt: start.Add(3 * time.Hour),
	})

	for _, test := range []struct {
		at                time.Duration
		booked, spendable float64
	}{
		{time.Hour, 70, 65},
		{2 * time.Hour, 65, 65},
		{3 * time.Hour, 62, 62},
	} {
		t.Run(test.at.String(), func(t *testing.T) {
			clock.FreezeTime(start.Add(test.at))
			defer clock.UnFreeze()
			next, err := env.Service.GetIncrementalBalance(ctx, input)
			require.NoError(t, err)
			require.True(t, next.Reused)
			require.NotNil(t, next.Checkpoint)
			require.Equal(t, test.booked, next.Balance.Booked().InexactFloat64())
			require.Equal(t, test.spendable, next.Balance.Spendable().InexactFloat64())
			require.Equal(t, float64(40), next.Balance.Live().InexactFloat64(), "the fee cannot spend the unchanged credits restricted to another feature")
			priorities := lo.Map(next.Checkpoint.Sources, func(source journalSource, _ int) int { return *source.Priority })
			require.Equal(t, []int{10, ledger.DefaultCustomerFBOPriority, 200}, priorities)

			cold, err := env.Service.GetIncrementalBalance(ctx, GetIncrementalBalanceInput{GetBalanceServiceInput: input.GetBalanceServiceInput})
			require.NoError(t, err)
			require.Equal(t, cold.Balance, next.Balance)
			require.Equal(t, cold.Checkpoint.Sources, next.Checkpoint.Sources)
			input.Checkpoints = []JournalCheckpoint{*next.Checkpoint}
		})
	}
}

type journalWithConcurrentPosting struct {
	ledger.BalanceQuerier
	ledger.JournalReader
	post func()
}

func (j *journalWithConcurrentPosting) BalanceJournal() ledger.JournalReader { return j }

func (j *journalWithConcurrentPosting) GetAccountBookingCoverage(ctx context.Context, input ledger.AccountBookingCoverageInput) (ledger.AccountBookingCoverage, error) {
	result, err := j.JournalReader.GetAccountBookingCoverage(ctx, input)
	j.post()

	return result, err
}

func TestConcurrentPostingPreventsJournalCheckpoint(t *testing.T) {
	env := newTestEnv(t)
	env.Service.BalanceQuerier = &journalWithConcurrentPosting{
		BalanceQuerier: env.Deps.HistoricalLedger,
		JournalReader:  env.Deps.HistoricalLedger.BalanceJournal(),
		post:           func() { env.bookFBOBalance(t, alpacadecimal.NewFromInt(25)) },
	}

	// The posting commits after coverage was read. Storing the projection under
	// those old coverage counts would make this checkpoint unusable.
	result, err := env.Service.GetIncrementalBalance(t.Context(), GetIncrementalBalanceInput{
		GetBalanceServiceInput: GetBalanceServiceInput{CustomerID: env.CustomerID, Currency: env.CurrencyReference()},
	})
	require.NoError(t, err)
	require.Nil(t, result.Checkpoint)
}

func TestBackdatedPostingDuringAdvanceRetriesFullCalculation(t *testing.T) {
	env := newTestEnv(t)
	ctx := t.Context()
	start := env.Now()
	input := GetIncrementalBalanceInput{GetBalanceServiceInput: GetBalanceServiceInput{
		CustomerID: env.CustomerID, Currency: env.CurrencyReference(),
	}}
	fund := func(n int64) {
		env.bookFBOBalance(t, alpacadecimal.NewFromInt(n))
		env.fundOpenReceivable(t, alpacadecimal.NewFromInt(n))
	}

	fund(10)
	first, err := env.Service.GetIncrementalBalance(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, first.Checkpoint)
	input.Checkpoints = []JournalCheckpoint{*first.Checkpoint}

	// Given a later booking, the saved opening is eligible for a forward read.
	clock.FreezeTime(start.Add(time.Hour))
	defer clock.UnFreeze()
	fund(20)
	env.Service.BalanceQuerier = &journalWithConcurrentPosting{
		BalanceQuerier: env.Deps.HistoricalLedger,
		JournalReader:  env.Deps.HistoricalLedger.BalanceJournal(),
		post: func() {
			// This commit invalidates the prefix after it was checked. The
			// interval query cannot see this amount in its range.
			clock.FreezeTime(start)
			defer clock.FreezeTime(start.Add(time.Hour))
			fund(5)
		},
	}

	result, err := env.Service.GetIncrementalBalance(ctx, input)
	require.NoError(t, err)
	require.False(t, result.Reused)
	require.Nil(t, result.Checkpoint)
	require.Equal(t, float64(35), result.Balance.Booked().InexactFloat64())
	full, err := env.Service.GetBalance(ctx, input.GetBalanceServiceInput)
	require.NoError(t, err)
	require.Equal(t, full, result.Balance)
}
