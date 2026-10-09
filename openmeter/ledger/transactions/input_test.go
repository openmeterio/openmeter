package transactions

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCommitGroup_AssignedIDs(t *testing.T) {
	t.Run("generated at commit", func(t *testing.T) {
		env := newTransactionsTestEnv(t)

		// given: a resolved credit issuance without assigned IDs
		input := env.creditIssuanceInput(t, 10)

		// when/then: committing generates IDs without changing the original input
		env.requireCommittedIDs(t, input, "")
		requireUnassignedIDs(t, input)
	})

	t.Run("preassigned before commit", func(t *testing.T) {
		env := newTransactionsTestEnv(t)

		// given: every posting is identified before it is committed
		original := env.creditIssuanceInput(t, 10)
		input, err := ledger.PreassignIDs(original)
		require.NoError(t, err)
		requireStablePreassignedIDs(t, input)

		// when/then: committing preserves the preassigned references
		env.requireCommittedIDs(t, input, "")
		requireUnassignedIDs(t, original)
	})

	t.Run("mixed supplied and generated", func(t *testing.T) {
		env := newTransactionsTestEnv(t)

		// given: the group, transaction and one entry have caller-supplied IDs
		original := env.creditIssuanceInput(t, 10)
		input := ledger.WithTransactionID(original, ulid.Make().String())
		entries := input.EntryInputs()
		entries[0] = ledger.WithEntryID(entries[0], ulid.Make().String())
		input = ledger.WithEntryInputs(input, entries...)
		groupID := ulid.Make().String()

		// when/then: committing preserves supplied IDs and generates the missing one
		env.requireCommittedIDs(t, input, groupID)
		requireUnassignedIDs(t, original)
	})

	t.Run("supplied IDs survive preassignment", func(t *testing.T) {
		env := newTransactionsTestEnv(t)

		// given: preassignment fills only the missing posting IDs
		original := env.creditIssuanceInput(t, 10)
		transactionID := ulid.Make().String()
		entryID := ulid.Make().String()
		input := ledger.WithTransactionID(original, transactionID)
		entries := input.EntryInputs()
		entries[0] = ledger.WithEntryID(entries[0], entryID)
		input = ledger.WithEntryInputs(input, entries...)

		input, err := ledger.PreassignIDs(input)
		require.NoError(t, err)
		require.Equal(t, transactionID, input.AssignedID())
		require.Equal(t, entryID, input.EntryInputs()[0].AssignedID())
		requireStablePreassignedIDs(t, input)

		// when/then: the prepared posting retains every ID through persistence
		env.requireCommittedIDs(t, input, ulid.Make().String())
		requireUnassignedIDs(t, original)
	})
}

func (e *transactionsTestEnv) creditIssuanceInput(t *testing.T, amount int64) ledger.TransactionInput {
	t.Helper()

	return e.resolve(t, IssueCustomerReceivableTemplate{
		At:       e.Now(),
		Amount:   alpacadecimal.NewFromInt(amount),
		Currency: e.CurrencyReference(),
	})[0]
}

func (e *transactionsTestEnv) requireCommittedIDs(t *testing.T, input ledger.TransactionInput, groupID string) {
	t.Helper()

	input = WithAnnotations(input, models.Annotations{"test": "assigned IDs"})
	groupInput := ledger.WithGroupID(input.AsGroupInput(e.Namespace, models.Annotations{"group": "assigned IDs"}), groupID)

	group, err := e.Deps.HistoricalLedger.CommitGroup(t.Context(), groupInput)
	require.NoError(t, err)

	group, err = e.Deps.HistoricalLedger.GetTransactionGroup(t.Context(), group.ID())
	require.NoError(t, err)

	if groupID != "" {
		require.Equal(t, groupID, group.ID().ID)
	}

	require.NotEmpty(t, group.ID().ID)
	require.NoError(t, ledger.ValidateAssignedID(group.ID().ID))
	require.Equal(t, groupInput.Annotations(), group.Annotations())
	require.Len(t, group.Transactions(), 1)

	committed := group.Transactions()[0]
	if input.AssignedID() != "" {
		require.Equal(t, input.AssignedID(), committed.ID().ID)
	}

	require.NotEmpty(t, committed.ID().ID)
	require.NoError(t, ledger.ValidateAssignedID(committed.ID().ID))
	require.Equal(t, input.Annotations(), committed.Annotations())
	require.Len(t, committed.Entries(), len(input.EntryInputs()))

	for _, entryInput := range input.EntryInputs() {
		var matched ledger.Entry
		for _, entry := range committed.Entries() {
			if entry.PostingAddress().SubAccountID() == entryInput.PostingAddress().SubAccountID() {
				matched = entry
				break
			}
		}

		require.NotNil(t, matched)
		if entryInput.AssignedID() != "" {
			require.Equal(t, entryInput.AssignedID(), matched.ID().ID)
		}

		require.NotEmpty(t, matched.ID().ID)
		require.NoError(t, ledger.ValidateAssignedID(matched.ID().ID))
		require.Equal(t, entryInput.Amount().InexactFloat64(), matched.Amount().InexactFloat64())
		require.Equal(t, entryInput.IdentityKey(), matched.IdentityKey())
	}

	require.Equal(t, 10.0, e.SumBalance(t, e.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority)).InexactFloat64())
	require.Equal(t, -10.0, e.SumBalance(t, e.ReceivableSubAccount(t)).InexactFloat64())
}

func requireUnassignedIDs(t *testing.T, input ledger.TransactionInput) {
	t.Helper()

	require.Empty(t, input.AssignedID())
	for _, entry := range input.EntryInputs() {
		require.Empty(t, entry.AssignedID())
	}
}

func requireStablePreassignedIDs(t *testing.T, input ledger.TransactionInput) {
	t.Helper()

	again, err := ledger.PreassignIDs(input)
	require.NoError(t, err)
	require.NotEmpty(t, input.AssignedID())
	require.Equal(t, input.AssignedID(), again.AssignedID())

	for idx, entry := range input.EntryInputs() {
		require.NotEmpty(t, entry.AssignedID())
		require.Equal(t, entry.AssignedID(), again.EntryInputs()[idx].AssignedID())
	}
}

func TestCommitGroup_RejectsDuplicateEntryIDs(t *testing.T) {
	env := newTransactionsTestEnv(t)

	// given: two balanced transactions reuse the same entry primary key
	first := env.creditIssuanceInput(t, 10)
	second := env.creditIssuanceInput(t, 5)
	id := ulid.Make().String()
	firstEntries := first.EntryInputs()
	firstEntries[0] = ledger.WithEntryID(firstEntries[0], id)
	first = ledger.WithEntryInputs(first, firstEntries...)
	secondEntries := second.EntryInputs()
	secondEntries[0] = ledger.WithEntryID(secondEntries[0], id)
	second = ledger.WithEntryInputs(second, secondEntries...)

	// when: the group is submitted to the ledger
	_, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), GroupInputs(env.Namespace, nil, first, second))
	require.ErrorContains(t, err, "duplicate entry ID")

	// then: group validation prevents any accounting rows from being persisted
	groups, err := env.DB.LedgerTransactionGroup.Query().Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, groups)

	entries, err := env.DB.LedgerEntry.Query().Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, entries)
}

func TestCommitGroup_IDCollisionRollsBack(t *testing.T) {
	setup := func(t *testing.T) (*transactionsTestEnv, ledger.TransactionGroup, []ledger.TransactionInput) {
		t.Helper()

		env := newTransactionsTestEnv(t)
		original := env.creditIssuanceInput(t, 10)
		committed, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), original.AsGroupInput(env.Namespace, nil))
		require.NoError(t, err)

		inputs := []ledger.TransactionInput{
			env.creditIssuanceInput(t, 7),
			env.creditIssuanceInput(t, 7),
		}

		return env, committed, inputs
	}

	requireRollback := func(t *testing.T, env *transactionsTestEnv, group ledger.TransactionGroupInput) {
		t.Helper()

		_, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), group)
		require.Error(t, err)

		groups, err := env.DB.LedgerTransactionGroup.Query().Count(t.Context())
		require.NoError(t, err)
		require.Equal(t, 1, groups)

		txs, err := env.DB.LedgerTransaction.Query().Count(t.Context())
		require.NoError(t, err)
		require.Equal(t, 1, txs)

		entries, err := env.DB.LedgerEntry.Query().Count(t.Context())
		require.NoError(t, err)
		require.Equal(t, 2, entries)

		require.Equal(t, 10.0, env.SumBalance(t, env.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority)).InexactFloat64())
		require.Equal(t, -10.0, env.SumBalance(t, env.ReceivableSubAccount(t)).InexactFloat64())
	}

	t.Run("group collision", func(t *testing.T) {
		// given: an existing posting owns the requested group ID
		env, committed, inputs := setup(t)
		group := ledger.WithGroupID(GroupInputs(env.Namespace, nil, inputs...), committed.ID().ID)

		// when/then: the collision leaves only the original posting
		requireRollback(t, env, group)
	})

	t.Run("transaction collision after an earlier transaction posts", func(t *testing.T) {
		// given: the second transaction reuses an existing transaction ID
		env, committed, inputs := setup(t)
		inputs[1] = ledger.WithTransactionID(inputs[1], committed.Transactions()[0].ID().ID)
		group := ledger.WithGroupID(GroupInputs(env.Namespace, nil, inputs...), ulid.Make().String())

		// when/then: the collision rolls back the first transaction as well
		requireRollback(t, env, group)
	})

	t.Run("entry collision after an earlier transaction posts", func(t *testing.T) {
		// given: the second transaction reuses an existing entry ID
		env, committed, inputs := setup(t)
		entries := inputs[1].EntryInputs()
		entries[0] = ledger.WithEntryID(entries[0], committed.Transactions()[0].Entries()[0].ID().ID)
		inputs[1] = ledger.WithEntryInputs(inputs[1], entries...)
		group := ledger.WithGroupID(GroupInputs(env.Namespace, nil, inputs...), ulid.Make().String())

		// when/then: the collision rolls back all entries from the new group
		requireRollback(t, env, group)
	})
}

func TestPreassignIDs_RejectsInvalidInput(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input ledger.TransactionInput
	}{
		{
			name: "missing transaction",
		},
		{
			name:  "invalid transaction ID",
			input: &TransactionInput{id: "invalid"},
		},
		{
			name: "invalid entry ID",
			input: &TransactionInput{
				entryInputs: []*EntryInput{{id: "invalid"}},
			},
		},
		{
			name:  "missing entry",
			input: ledger.WithEntryInputs(&TransactionInput{}, nil),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input, err := ledger.PreassignIDs(tc.input)
			require.Error(t, err)
			require.Nil(t, input)
		})
	}
}
