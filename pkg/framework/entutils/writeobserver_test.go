package entutils_test

import (
	"context"
	"errors"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	db1 "github.com/openmeterio/openmeter/pkg/framework/entutils/testutils/ent1/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/testutils/ent1/db/example1"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	transactiontestutils "github.com/openmeterio/openmeter/pkg/framework/transaction/testutils"
)

func TestWriteObserverDetection(t *testing.T) {
	database := testutils.InitPostgresDB(t, testutils.PostgresDBStateEmpty)
	t.Cleanup(func() { database.Close(t) })

	// Neither the pooled client nor the transactional ones register anything:
	// the generated code installs the write observers on every client it builds.
	client := db1.NewClient(db1.Driver(database.EntDriver.Driver()))
	require.NoError(t, client.Schema.Create(t.Context()))

	adapter := &db1Adapter{db: client}

	mutation := func(operation string) transaction.WriteAttempt {
		return transaction.WriteAttempt{Kind: transaction.WriteKindMutation, Target: "Example1", Operation: operation}
	}

	rowLock := func(operation string) transaction.WriteAttempt {
		return transaction.WriteAttempt{Kind: transaction.WriteKindRowLock, Target: "Example1", Operation: operation}
	}

	tests := []struct {
		name     string
		run      func(ctx context.Context, client *db1.Client, id string) error
		expected []transaction.WriteAttempt
	}{
		{
			name: "create",
			run: func(ctx context.Context, client *db1.Client, id string) error {
				return client.Example1.Create().SetID(id).SetExampleValue1("v").Exec(ctx)
			},
			expected: []transaction.WriteAttempt{mutation("OpCreate")},
		},
		{
			name: "bulk create reports every row",
			run: func(ctx context.Context, client *db1.Client, id string) error {
				return client.Example1.CreateBulk(
					client.Example1.Create().SetID(id+"-1").SetExampleValue1("v"),
					client.Example1.Create().SetID(id+"-2").SetExampleValue1("v"),
				).Exec(ctx)
			},
			expected: []transaction.WriteAttempt{mutation("OpCreate"), mutation("OpCreate")},
		},
		{
			name: "upsert",
			run: func(ctx context.Context, client *db1.Client, id string) error {
				return client.Example1.Create().SetID(id).SetExampleValue1("v").
					OnConflictColumns(example1.FieldID).
					UpdateNewValues().
					Exec(ctx)
			},
			expected: []transaction.WriteAttempt{mutation("OpCreate")},
		},
		{
			name: "update",
			run: func(ctx context.Context, client *db1.Client, id string) error {
				return client.Example1.Update().Where(example1.ID(id)).SetExampleValue1("v").Exec(ctx)
			},
			expected: []transaction.WriteAttempt{mutation("OpUpdate")},
		},
		{
			name: "update one",
			run: func(ctx context.Context, client *db1.Client, id string) error {
				if err := client.Example1.Create().SetID(id).SetExampleValue1("v").Exec(ctx); err != nil {
					return err
				}

				return client.Example1.UpdateOneID(id).SetExampleValue1("w").Exec(ctx)
			},
			expected: []transaction.WriteAttempt{mutation("OpCreate"), mutation("OpUpdateOne")},
		},
		{
			name: "delete",
			run: func(ctx context.Context, client *db1.Client, id string) error {
				_, err := client.Example1.Delete().Where(example1.ID(id)).Exec(ctx)

				return err
			},
			expected: []transaction.WriteAttempt{mutation("OpDelete")},
		},
		{
			name: "delete one",
			run: func(ctx context.Context, client *db1.Client, id string) error {
				if err := client.Example1.Create().SetID(id).SetExampleValue1("v").Exec(ctx); err != nil {
					return err
				}

				return client.Example1.DeleteOneID(id).Exec(ctx)
			},
			expected: []transaction.WriteAttempt{mutation("OpCreate"), mutation("OpDeleteOne")},
		},
		{
			name: "select for update",
			run: func(ctx context.Context, client *db1.Client, id string) error {
				_, err := client.Example1.Query().Where(example1.ID(id)).ForUpdate().All(ctx)

				return err
			},
			expected: []transaction.WriteAttempt{rowLock("All")},
		},
		{
			// IDs goes through the Select/Scan path instead of the node query path.
			name: "select for share through a field selection",
			run: func(ctx context.Context, client *db1.Client, id string) error {
				_, err := client.Example1.Query().Where(example1.ID(id)).ForShare().IDs(ctx)

				return err
			},
			expected: []transaction.WriteAttempt{rowLock("IDs")},
		},
		{
			name: "plain reads are not reported",
			run: func(ctx context.Context, client *db1.Client, id string) error {
				if _, err := client.Example1.Query().Where(example1.ID(id)).All(ctx); err != nil {
					return err
				}

				if _, err := client.Example1.Query().Where(example1.ID(id)).IDs(ctx); err != nil {
					return err
				}

				_, err := client.Example1.Query().Where(example1.ID(id)).Count(ctx)

				return err
			},
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("pooled client", func(t *testing.T) {
				// given a context with a write observer
				recorder := &transactiontestutils.WriteRecorder{}
				ctx, err := transaction.WithWriteObserver(t.Context(), recorder)
				require.NoError(t, err)

				// when the statements run outside a transaction
				require.NoError(t, tc.run(ctx, client, ulid.Make().String()))

				// then every write is reported, and still executed
				require.Equal(t, tc.expected, recorder.Attempts())
			})

			t.Run("transaction", func(t *testing.T) {
				// given a context with a write observer
				recorder := &transactiontestutils.WriteRecorder{}
				ctx, err := transaction.WithWriteObserver(t.Context(), recorder)
				require.NoError(t, err)

				// when the statements run through a transactional client
				require.NoError(t, transaction.RunWithNoValue(ctx, adapter, func(ctx context.Context) error {
					tx, err := entutils.GetDriverFromContext(ctx)
					if err != nil {
						return err
					}

					return tc.run(ctx, db1.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client(), ulid.Make().String())
				}))

				// then every write is reported, and still executed
				require.Equal(t, tc.expected, recorder.Attempts())
			})
		})
	}

	t.Run("a rejecting observer stops the write before it reaches the database", func(t *testing.T) {
		// given a context whose observer rejects writes
		rejected := errors.New("writes are not allowed")
		ctx, err := transaction.WithWriteObserver(t.Context(), &transactiontestutils.WriteRecorder{Reject: rejected})
		require.NoError(t, err)

		id := ulid.Make().String()

		// when a mutation and a locking read are attempted
		createErr := client.Example1.Create().SetID(id).SetExampleValue1("v").Exec(ctx)
		_, lockErr := client.Example1.Query().Where(example1.ID(id)).ForUpdate().All(ctx)

		// then both fail and nothing was written
		require.ErrorIs(t, createErr, rejected)
		require.ErrorIs(t, lockErr, rejected)

		exists, err := client.Example1.Query().Where(example1.ID(id)).Exist(t.Context())
		require.NoError(t, err)
		require.False(t, exists)
	})

	t.Run("writes without an observer are untouched", func(t *testing.T) {
		id := ulid.Make().String()

		require.NoError(t, client.Example1.Create().SetID(id).SetExampleValue1("v").Exec(t.Context()))

		_, err := client.Example1.Query().Where(example1.ID(id)).ForUpdate().All(t.Context())
		require.NoError(t, err)
	})
}
