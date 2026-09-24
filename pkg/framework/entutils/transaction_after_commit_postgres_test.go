package entutils_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	db1 "github.com/openmeterio/openmeter/pkg/framework/entutils/testutils/ent1/db"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func TestAfterCommitWithPostgres(t *testing.T) {
	for _, outcome := range []string{"commit", "rollback", "commit failure"} {
		t.Run(outcome, func(t *testing.T) {
			// Given a real Ent adapter and a constraint checked only at outer commit.
			database := testutils.InitPostgresDB(t, testutils.PostgresDBStateEmpty)
			t.Cleanup(func() { database.Close(t) })
			client := db1.NewClient(db1.Driver(database.EntDriver.Driver()))
			require.NoError(t, client.Schema.Create(t.Context()))
			_, err := database.PGDriver.DB().ExecContext(t.Context(),
				"ALTER TABLE example1s ADD CONSTRAINT unique_value UNIQUE (example_value_1) DEFERRABLE INITIALLY DEFERRED")
			require.NoError(t, err)
			adapter := &db1Adapter{db: client}
			rollbackErr := errors.New("operation failed")
			var called []string
			writeAndRegister := func(ctx context.Context, id string) {
				tx, err := entutils.GetDriverFromContext(ctx)
				require.NoError(t, err)
				value := id
				if outcome == "commit failure" {
					value = "duplicate"
				}
				_, err = adapter.WithTx(ctx, tx).Save(ctx, &db1.Example1{ID: id, ExampleValue1: value})
				require.NoError(t, err)
				require.NoError(t, tx.AfterCommit(func() {
					// Read through the nontransactional client, on another connection.
					stored, err := client.Example1.Get(t.Context(), id)
					require.NoError(t, err)
					require.Equal(t, value, stored.ExampleValue1)
					called = append(called, id)
				}))
			}

			// When nested operations commit, roll back, and commit inside a rolled-back parent.
			err = transaction.RunWithNoValue(t.Context(), adapter, func(ctx context.Context) error {
				writeAndRegister(ctx, "outer")
				require.NoError(t, transaction.RunWithNoValue(ctx, adapter, func(ctx context.Context) error {
					writeAndRegister(ctx, "kept")
					return nil
				}))
				err := transaction.RunWithNoValue(ctx, adapter, func(ctx context.Context) error {
					writeAndRegister(ctx, "parent")
					require.NoError(t, transaction.RunWithNoValue(ctx, adapter, func(ctx context.Context) error {
						writeAndRegister(ctx, "child")
						return nil
					}))
					return rollbackErr
				})
				require.ErrorIs(t, err, rollbackErr)
				err = transaction.RunWithNoValue(ctx, adapter, func(ctx context.Context) error {
					writeAndRegister(ctx, "sibling")
					return rollbackErr
				})
				require.ErrorIs(t, err, rollbackErr)
				require.Empty(t, called)
				count, err := client.Example1.Query().Count(t.Context())
				require.NoError(t, err)
				require.Zero(t, count, "uncommitted writes are invisible to other connections")
				if outcome == "rollback" {
					return rollbackErr
				}
				return nil
			})

			// Then callbacks observe committed rows only; failed scopes leave neither behind.
			if outcome == "commit" {
				require.NoError(t, err)
				require.Equal(t, []string{"outer", "kept"}, called)
			} else {
				if outcome == "rollback" {
					require.ErrorIs(t, err, rollbackErr)
				} else {
					require.ErrorContains(t, err, "unique_value")
				}
				require.Empty(t, called)
			}
			ids, err := client.Example1.Query().IDs(t.Context())
			require.NoError(t, err)
			require.ElementsMatch(t, called, ids)
		})
	}
}
