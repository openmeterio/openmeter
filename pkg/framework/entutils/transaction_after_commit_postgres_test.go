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

type afterCommitPostgresFixture struct {
	t       *testing.T
	client  *db1.Client
	adapter *db1Adapter
	called  []string
}

func newAfterCommitPostgresFixture(t *testing.T) *afterCommitPostgresFixture {
	t.Helper()
	database := testutils.InitPostgresDB(t, testutils.PostgresDBStateEmpty)
	t.Cleanup(func() { database.Close(t) })
	client := db1.NewClient(db1.Driver(database.EntDriver.Driver()))
	require.NoError(t, client.Schema.Create(t.Context()))
	return &afterCommitPostgresFixture{t: t, client: client, adapter: &db1Adapter{db: client}}
}

// writeAndRegister pairs a transactional write with a callback that verifies
// the row is committed and readable through a separate connection.
func (f *afterCommitPostgresFixture) writeAndRegister(ctx context.Context, id, value string) {
	f.t.Helper()
	tx, err := entutils.GetDriverFromContext(ctx)
	require.NoError(f.t, err)
	_, err = f.adapter.WithTx(ctx, tx).Save(ctx, &db1.Example1{ID: id, ExampleValue1: value})
	require.NoError(f.t, err)
	require.NoError(f.t, tx.AfterCommit(func() {
		stored, err := f.client.Example1.Get(f.t.Context(), id)
		require.NoError(f.t, err)
		require.Equal(f.t, value, stored.ExampleValue1)
		f.called = append(f.called, id)
	}))
}

func (f *afterCommitPostgresFixture) assertCommitted(ids ...string) {
	f.t.Helper()
	require.Equal(f.t, ids, f.called, "callbacks must match committed writes in registration order")
	stored, err := f.client.Example1.Query().IDs(f.t.Context())
	require.NoError(f.t, err)
	require.ElementsMatch(f.t, ids, stored)
}

func TestAfterCommitWithPostgres(t *testing.T) {
	t.Run("nested commit waits for outer commit", func(t *testing.T) {
		// Given an outer write and a successfully committed nested write.
		f := newAfterCommitPostgresFixture(t)

		// When the nested scope commits, neither writes nor callbacks escape the outer transaction.
		err := transaction.RunWithNoValue(t.Context(), f.adapter, func(ctx context.Context) error {
			f.writeAndRegister(ctx, "outer", "outer value")
			require.NoError(t, transaction.RunWithNoValue(ctx, f.adapter, func(ctx context.Context) error {
				f.writeAndRegister(ctx, "nested", "nested value")
				return nil
			}))
			f.assertCommitted()
			return nil
		})

		// Then both callbacks observe committed rows, in registration order.
		require.NoError(t, err)
		f.assertCommitted("outer", "nested")
	})

	t.Run("savepoint rollback preserves surrounding callbacks", func(t *testing.T) {
		// Given a nested operation that fails between two successful writes.
		f := newAfterCommitPostgresFixture(t)
		operationErr := errors.New("nested operation failed")

		// When the outer transaction recovers and commits.
		err := transaction.RunWithNoValue(t.Context(), f.adapter, func(ctx context.Context) error {
			f.writeAndRegister(ctx, "before", "before value")
			err := transaction.RunWithNoValue(ctx, f.adapter, func(ctx context.Context) error {
				f.writeAndRegister(ctx, "discarded", "discarded value")
				return operationErr
			})
			require.ErrorIs(t, err, operationErr)
			f.writeAndRegister(ctx, "after", "after value")
			return nil
		})

		// Then only the failed scope loses its write and callback.
		require.NoError(t, err)
		f.assertCommitted("before", "after")
	})

	t.Run("parent rollback discards committed child callbacks", func(t *testing.T) {
		// Given a child scope that succeeds inside a parent that later fails.
		f := newAfterCommitPostgresFixture(t)
		operationErr := errors.New("parent operation failed")

		// When the outer transaction recovers from the parent failure.
		err := transaction.RunWithNoValue(t.Context(), f.adapter, func(ctx context.Context) error {
			f.writeAndRegister(ctx, "outer", "outer value")
			err := transaction.RunWithNoValue(ctx, f.adapter, func(ctx context.Context) error {
				f.writeAndRegister(ctx, "parent", "parent value")
				require.NoError(t, transaction.RunWithNoValue(ctx, f.adapter, func(ctx context.Context) error {
					f.writeAndRegister(ctx, "child", "child value")
					return nil
				}))
				return operationErr
			})
			require.ErrorIs(t, err, operationErr)
			return nil
		})

		// Then the child is discarded with its parent despite its successful savepoint release.
		require.NoError(t, err)
		f.assertCommitted("outer")
	})

	t.Run("outer rollback discards all callbacks", func(t *testing.T) {
		// Given successful writes in outer and nested scopes.
		f := newAfterCommitPostgresFixture(t)
		operationErr := errors.New("outer operation failed")

		// When the outer operation fails after the nested scope succeeds.
		err := transaction.RunWithNoValue(t.Context(), f.adapter, func(ctx context.Context) error {
			f.writeAndRegister(ctx, "outer", "outer value")
			require.NoError(t, transaction.RunWithNoValue(ctx, f.adapter, func(ctx context.Context) error {
				f.writeAndRegister(ctx, "nested", "nested value")
				return nil
			}))
			return operationErr
		})

		// Then neither writes nor callbacks survive.
		require.ErrorIs(t, err, operationErr)
		f.assertCommitted()
	})

	t.Run("database commit failure discards all callbacks", func(t *testing.T) {
		// Given a uniqueness constraint checked only at outer commit.
		f := newAfterCommitPostgresFixture(t)
		_, err := f.client.ExecContext(t.Context(),
			"ALTER TABLE example1s ADD CONSTRAINT unique_value UNIQUE (example_value_1) DEFERRABLE INITIALLY DEFERRED")
		require.NoError(t, err)

		// When valid statements complete but their transaction violates the deferred constraint.
		err = transaction.RunWithNoValue(t.Context(), f.adapter, func(ctx context.Context) error {
			f.writeAndRegister(ctx, "first", "duplicate")
			f.writeAndRegister(ctx, "second", "duplicate")
			return nil
		})

		// Then the database rejects the commit and no callbacks run.
		require.ErrorContains(t, err, "unique_value")
		f.assertCommitted()
	})
}
