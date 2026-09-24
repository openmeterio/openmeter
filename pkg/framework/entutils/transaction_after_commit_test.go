package entutils_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type afterCommitDriver struct {
	commitErr error
}

func (d *afterCommitDriver) Commit() error           { return d.commitErr }
func (d *afterCommitDriver) Rollback() error         { return nil }
func (d *afterCommitDriver) SavePoint(string) error  { return nil }
func (d *afterCommitDriver) RollbackTo(string) error { return nil }
func (d *afterCommitDriver) Release(string) error    { return nil }

func TestAfterCommitSavepointScope(t *testing.T) {
	tx := entutils.NewTxDriver(&afterCommitDriver{}, nil)
	require.NoError(t, tx.SavePoint())

	var called []string
	register := func(name string) {
		require.NoError(t, tx.AfterCommit(func() { called = append(called, name) }))
	}
	register("outer-before")

	// A successful nested operation keeps its callbacks for the outer commit.
	require.NoError(t, tx.SavePoint())
	register("nested-committed")
	require.NoError(t, tx.Commit())
	require.Empty(t, called)

	// A rolled-back nested operation discards only its own callbacks.
	require.NoError(t, tx.SavePoint())
	register("nested-rolled-back")
	require.NoError(t, tx.Rollback())
	register("outer-after")
	require.Empty(t, called)

	require.NoError(t, tx.Commit())
	require.Equal(t, []string{"outer-before", "nested-committed", "outer-after"}, called)
}

func TestAfterCommitDiscardedWithParentSavepoint(t *testing.T) {
	tx := entutils.NewTxDriver(&afterCommitDriver{}, nil)
	require.NoError(t, tx.SavePoint())
	var called []string
	require.NoError(t, tx.AfterCommit(func() { called = append(called, "outer") }))

	require.NoError(t, tx.SavePoint())
	require.NoError(t, tx.AfterCommit(func() { called = append(called, "parent") }))
	require.NoError(t, tx.SavePoint())
	require.NoError(t, tx.AfterCommit(func() { called = append(called, "child") }))
	require.NoError(t, tx.Commit())
	require.Empty(t, called)

	require.NoError(t, tx.Rollback())
	require.NoError(t, tx.Commit())
	require.Equal(t, []string{"outer"}, called)
}

func TestAfterCommitDiscardedOnOuterFailure(t *testing.T) {
	tests := []struct {
		name      string
		driverErr error
		finish    func(*entutils.TxDriver) error
	}{
		{name: "rollback", finish: (*entutils.TxDriver).Rollback},
		{name: "commit failure", driverErr: errors.New("commit failed"), finish: (*entutils.TxDriver).Commit},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tx := entutils.NewTxDriver(&afterCommitDriver{commitErr: test.driverErr}, nil)
			require.NoError(t, tx.SavePoint())
			called := false
			require.NoError(t, tx.AfterCommit(func() { called = true }))
			err := test.finish(tx)
			if test.driverErr != nil {
				require.ErrorIs(t, err, test.driverErr)
			} else {
				require.NoError(t, err)
			}
			require.False(t, called)
			require.Error(t, tx.AfterCommit(func() {}))
		})
	}
}

func TestAfterCommitRunsOutsideDriverLock(t *testing.T) {
	tx := entutils.NewTxDriver(&afterCommitDriver{}, nil)
	require.NoError(t, tx.SavePoint())
	callbackDone := make(chan error, 1)
	require.NoError(t, tx.AfterCommit(func() {
		callbackDone <- tx.AfterCommit(func() {})
	}))

	commitDone := make(chan error, 1)
	go func() { commitDone <- tx.Commit() }()

	select {
	case err := <-callbackDone:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("after-commit callback ran while holding the transaction lock")
	}
	require.NoError(t, <-commitDone)
}
