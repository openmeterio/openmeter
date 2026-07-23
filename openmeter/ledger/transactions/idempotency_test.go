package transactions

import (
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ent/db/ledgerentry"
	"github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransactiongroup"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/historical"
	historicaladapter "github.com/openmeterio/openmeter/openmeter/ledger/historical/adapter"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCommitGroupIdempotency(t *testing.T) {
	t.Run("nil key preserves one-shot behavior", func(t *testing.T) {
		// given:
		// - one resolved issuance transaction without an idempotency key
		// when:
		// - the same input is committed twice
		// then:
		// - both groups and both balance effects are retained
		env := newTransactionsTestEnv(t)
		inputs := env.resolveIdempotencyIssue(t, 50)

		first, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), GroupInputs(env.Namespace, nil, inputs...))
		require.NoError(t, err)
		second, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), GroupInputs(env.Namespace, nil, inputs...))
		require.NoError(t, err)

		require.NotEqual(t, first.ID(), second.ID())
		require.Equal(t, ledgerRowCounts{Groups: 2, Transactions: 2, Entries: 4}, queryLedgerRowCounts(t, env))
		require.Equal(t, float64(100), env.SumBalance(t, env.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority)).InexactFloat64())
	})

	t.Run("sequential replay returns original group", func(t *testing.T) {
		// given:
		// - one keyed issuance transaction
		// when:
		// - the exact financial input is committed twice
		// then:
		// - the second call returns the original group without another booking
		env := newTransactionsTestEnv(t)
		inputs := env.resolveIdempotencyIssue(t, 50)
		group := WithIdempotencyKey(
			"issue:sequential",
			GroupInputs(env.Namespace, nil, inputs...),
		)

		first, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), group)
		require.NoError(t, err)
		second, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), group)
		require.NoError(t, err)

		require.Equal(t, first.ID(), second.ID())
		require.Equal(t, ledgerRowCounts{Groups: 1, Transactions: 1, Entries: 2}, queryLedgerRowCounts(t, env))
		require.Equal(t, float64(50), env.SumBalance(t, env.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority)).InexactFloat64())
	})

	t.Run("lost response replay recovers committed group", func(t *testing.T) {
		// given:
		// - a keyed commit whose response is deliberately discarded
		// when:
		// - the caller retries with the same key and stable input
		// then:
		// - the durable group is returned with no duplicate effect
		env := newTransactionsTestEnv(t)
		inputs := env.resolveIdempotencyIssue(t, 35)
		const key = "issue:lost-response"

		_, err := env.Deps.HistoricalLedger.CommitGroup(
			t.Context(),
			WithIdempotencyKey(
				key,
				GroupInputs(env.Namespace, nil, inputs...),
			),
		)
		require.NoError(t, err)

		persisted, err := env.DB.LedgerTransactionGroup.Query().
			Where(
				ledgertransactiongroup.Namespace(env.Namespace),
				ledgertransactiongroup.IdempotencyKey(key),
			).
			Only(t.Context())
		require.NoError(t, err)

		recovered, err := env.Deps.HistoricalLedger.CommitGroup(
			t.Context(),
			WithIdempotencyKey(
				key,
				GroupInputs(env.Namespace, nil, inputs...),
			),
		)
		require.NoError(t, err)

		require.Equal(t, persisted.ID, recovered.ID().ID)
		require.Equal(t, ledgerRowCounts{Groups: 1, Transactions: 1, Entries: 2}, queryLedgerRowCounts(t, env))
	})

	t.Run("entry order and request annotations do not change the fingerprint", func(t *testing.T) {
		// given:
		// - equivalent financial entries in a different input order
		// - different free-form request annotations
		// when:
		// - both inputs use the same key
		// then:
		// - replay succeeds because neither difference changes financial semantics
		env := newTransactionsTestEnv(t)
		inputs := env.resolveIdempotencyIssue(t, 25)
		firstInput := WithAnnotations(inputs[0], models.Annotations{"request.id": "first"})

		reorderedEntries := slices.Clone(inputs[0].EntryInputs())
		slices.Reverse(reorderedEntries)
		secondInput := WithAnnotations(
			&transactionInputWithEntries{
				TransactionInput: inputs[0],
				entries:          reorderedEntries,
			},
			models.Annotations{"request.id": "second"},
		)

		first, err := env.Deps.HistoricalLedger.CommitGroup(
			t.Context(),
			WithIdempotencyKey(
				"issue:canonical",
				GroupInputs(env.Namespace, nil, firstInput),
			),
		)
		require.NoError(t, err)
		second, err := env.Deps.HistoricalLedger.CommitGroup(
			t.Context(),
			WithIdempotencyKey(
				"issue:canonical",
				GroupInputs(env.Namespace, nil, secondInput),
			),
		)
		require.NoError(t, err)

		require.Equal(t, first.ID(), second.ID())
		require.Equal(t, ledgerRowCounts{Groups: 1, Transactions: 1, Entries: 2}, queryLedgerRowCounts(t, env))
	})

	t.Run("concurrent replay commits one group", func(t *testing.T) {
		// given:
		// - two workers ready to commit the same keyed transaction
		// when:
		// - both calls cross the preflight boundary concurrently
		// then:
		// - the database fences duplicate effects and both callers observe one group
		env := newTransactionsTestEnv(t)
		inputs := env.resolveIdempotencyIssue(t, 45)
		group := WithIdempotencyKey(
			"issue:concurrent",
			GroupInputs(env.Namespace, nil, inputs...),
		)
		ctx := t.Context()
		start := make(chan struct{})
		results := make([]ledger.TransactionGroup, 2)
		errs := make([]error, 2)
		var workers sync.WaitGroup

		for index := range results {
			workers.Add(1)
			go func() {
				defer workers.Done()
				<-start
				results[index], errs[index] = env.Deps.HistoricalLedger.CommitGroup(ctx, group)
			}()
		}

		close(start)
		workers.Wait()

		require.NoError(t, errs[0])
		require.NoError(t, errs[1])
		require.Equal(t, results[0].ID(), results[1].ID())
		require.Equal(t, ledgerRowCounts{Groups: 1, Transactions: 1, Entries: 2}, queryLedgerRowCounts(t, env))
		require.Equal(t, float64(45), env.SumBalance(t, env.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority)).InexactFloat64())
	})
}

func TestCommitGroupIdempotencyConflict(t *testing.T) {
	tests := []struct {
		name  string
		retry func(t *testing.T, env *transactionsTestEnv) []ledger.TransactionInput
	}{
		{
			name: "amount",
			retry: func(t *testing.T, env *transactionsTestEnv) []ledger.TransactionInput {
				return env.resolveIdempotencyIssue(t, 51)
			},
		},
		{
			name: "currency route",
			retry: func(t *testing.T, env *transactionsTestEnv) []ledger.TransactionInput {
				return env.resolve(
					t,
					IssueCustomerReceivableTemplate{
						At:       env.Now(),
						Amount:   alpacadecimal.NewFromInt(50),
						Currency: currencyx.Code("ACME"),
					},
				)
			},
		},
		{
			name: "booked at",
			retry: func(t *testing.T, env *transactionsTestEnv) []ledger.TransactionInput {
				return env.resolve(
					t,
					IssueCustomerReceivableTemplate{
						At:       env.Now().Add(time.Minute),
						Amount:   alpacadecimal.NewFromInt(50),
						Currency: env.Currency,
					},
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given:
			// - a committed keyed issuance
			// when:
			// - the key is reused for a different financial input
			// then:
			// - the caller receives a typed conflict and the ledger is unchanged
			env := newTransactionsTestEnv(t)
			key := "issue:mismatch:" + tt.name
			original := env.resolveIdempotencyIssue(t, 50)

			_, err := env.Deps.HistoricalLedger.CommitGroup(
				t.Context(),
				WithIdempotencyKey(
					key,
					GroupInputs(env.Namespace, nil, original...),
				),
			)
			require.NoError(t, err)

			_, err = env.Deps.HistoricalLedger.CommitGroup(
				t.Context(),
				WithIdempotencyKey(
					key,
					GroupInputs(env.Namespace, nil, tt.retry(t, env)...),
				),
			)
			require.Error(t, err)
			require.True(t, ledger.IsTransactionGroupIdempotencyConflict(err))
			require.Equal(t, ledgerRowCounts{Groups: 1, Transactions: 1, Entries: 2}, queryLedgerRowCounts(t, env))
			require.Equal(t, float64(50), env.SumBalance(t, env.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority)).InexactFloat64())
		})
	}
}

func TestCommitGroupIdempotencyReplayAfterBalanceChanges(t *testing.T) {
	// given:
	// - a keyed issuance followed by another independent balance mutation
	// when:
	// - the original issuance is replayed
	// then:
	// - replay bypasses balance-dependent work and returns the original group
	env := newTransactionsTestEnv(t)
	original := env.resolveIdempotencyIssue(t, 50)
	const key = "issue:after-balance-change"

	first, err := env.Deps.HistoricalLedger.CommitGroup(
		t.Context(),
		WithIdempotencyKey(
			key,
			GroupInputs(env.Namespace, nil, original...),
		),
	)
	require.NoError(t, err)

	other := env.resolve(
		t,
		IssueCustomerReceivableTemplate{
			At:       env.Now().Add(time.Minute),
			Amount:   alpacadecimal.NewFromInt(20),
			Currency: env.Currency,
		},
	)
	_, err = env.Deps.HistoricalLedger.CommitGroup(t.Context(), GroupInputs(env.Namespace, nil, other...))
	require.NoError(t, err)

	replayed, err := env.Deps.HistoricalLedger.CommitGroup(
		t.Context(),
		WithIdempotencyKey(
			key,
			GroupInputs(env.Namespace, nil, original...),
		),
	)
	require.NoError(t, err)

	require.Equal(t, first.ID(), replayed.ID())
	require.Equal(t, ledgerRowCounts{Groups: 2, Transactions: 2, Entries: 4}, queryLedgerRowCounts(t, env))
	require.Equal(t, float64(70), env.SumBalance(t, env.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority)).InexactFloat64())
}

func TestTransactionGroupIdempotencyKeyValidation(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "empty",
			key:     "",
			wantErr: true,
		},
		{
			name:    "maximum length",
			key:     strings.Repeat("a", ledger.TransactionGroupIdempotencyKeyMaxLength),
			wantErr: false,
		},
		{
			name:    "over maximum length",
			key:     strings.Repeat("a", ledger.TransactionGroupIdempotencyKeyMaxLength+1),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTransactionsTestEnv(t)
			inputs := env.resolveIdempotencyIssue(t, 10)

			_, err := env.Deps.HistoricalLedger.CommitGroup(
				t.Context(),
				WithIdempotencyKey(
					tt.key,
					GroupInputs(env.Namespace, nil, inputs...),
				),
			)
			if tt.wantErr {
				require.Error(t, err)
				require.ErrorIs(t, err, ledger.ErrTransactionGroupIdempotencyKeyInvalid)
				require.Equal(t, ledgerRowCounts{}, queryLedgerRowCounts(t, env))
			} else {
				require.NoError(t, err)
				require.Equal(t, ledgerRowCounts{Groups: 1, Transactions: 1, Entries: 2}, queryLedgerRowCounts(t, env))
			}
		})
	}
}

func TestTransactionGroupIdempotencyKeyIsNamespaceScoped(t *testing.T) {
	// given:
	// - two direct group records in different namespaces
	// when:
	// - both use the same key and financial fingerprint
	// then:
	// - the durable uniqueness boundary accepts both records
	env := newTransactionsTestEnv(t)
	repo := historicaladapter.NewRepo(env.DB)
	key := "shared-key"
	fingerprint := "v1:" + strings.Repeat("a", 64)

	first, err := repo.CreateTransactionGroup(t.Context(), historical.CreateTransactionGroupInput{
		Namespace:        env.Namespace,
		IdempotencyKey:   &key,
		InputFingerprint: &fingerprint,
	})
	require.NoError(t, err)
	second, err := repo.CreateTransactionGroup(t.Context(), historical.CreateTransactionGroupInput{
		Namespace:        "other-" + env.Namespace,
		IdempotencyKey:   &key,
		InputFingerprint: &fingerprint,
	})
	require.NoError(t, err)

	require.NotEqual(t, first.ID, second.ID)
}

type transactionInputWithEntries struct {
	ledger.TransactionInput
	entries []ledger.EntryInput
}

func (i *transactionInputWithEntries) EntryInputs() []ledger.EntryInput {
	return i.entries
}

type ledgerRowCounts struct {
	Groups       int
	Transactions int
	Entries      int
}

func queryLedgerRowCounts(t *testing.T, env *transactionsTestEnv) ledgerRowCounts {
	t.Helper()

	groups, err := env.DB.LedgerTransactionGroup.Query().
		Where(ledgertransactiongroup.Namespace(env.Namespace)).
		Count(t.Context())
	require.NoError(t, err)
	transactions, err := env.DB.LedgerTransaction.Query().
		Where(ledgertransaction.Namespace(env.Namespace)).
		Count(t.Context())
	require.NoError(t, err)
	entries, err := env.DB.LedgerEntry.Query().
		Where(ledgerentry.Namespace(env.Namespace)).
		Count(t.Context())
	require.NoError(t, err)

	return ledgerRowCounts{
		Groups:       groups,
		Transactions: transactions,
		Entries:      entries,
	}
}

func (e *transactionsTestEnv) resolveIdempotencyIssue(t *testing.T, amount int64) []ledger.TransactionInput {
	t.Helper()

	return e.resolve(
		t,
		IssueCustomerReceivableTemplate{
			At:       e.Now(),
			Amount:   alpacadecimal.NewFromInt(amount),
			Currency: e.Currency,
		},
	)
}
