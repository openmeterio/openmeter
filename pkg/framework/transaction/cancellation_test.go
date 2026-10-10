package transaction

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRunPreservesCancellationAfterSQLAutoRollback(t *testing.T) {
	// given a real database/sql transaction with a controllable driver
	rolledBack := make(chan struct{})
	db := sql.OpenDB(rollbackConnector{rolledBack})
	defer db.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	// when cancellation finishes the automatic rollback before Commit
	_, err := Run(ctx, sqlCreator{db}, func(context.Context) (struct{}, error) {
		cancel()
		select {
		case <-rolledBack:
		case <-time.After(5 * time.Second):
			t.Fatal("automatic rollback did not complete")
		}

		return struct{}{}, nil
	})

	// then both the cancellation and transaction state remain discoverable
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, err, sql.ErrTxDone)
}

func TestCommitFailureClassification(t *testing.T) {
	commitErr := errors.New("commit failed")
	for _, test := range []struct {
		name       string
		err        error
		contextErr error
	}{
		{"active transaction already done", sql.ErrTxDone, nil},
		{"canceled transaction already done", sql.ErrTxDone, context.Canceled},
		{"expired transaction already done", sql.ErrTxDone, context.DeadlineExceeded},
		{"unrelated commit failure", commitErr, context.Canceled},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			switch test.contextErr {
			case context.Canceled:
				canceled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = canceled
			case context.DeadlineExceeded:
				expired, cancel := context.WithDeadline(ctx, time.Now().Add(-time.Second))
				defer cancel()
				ctx = expired
			}

			_, err := Run(ctx, &noopCreator{driver: commitFailureDriver{test.err}}, func(context.Context) (struct{}, error) {
				return struct{}{}, nil
			})
			require.ErrorIs(t, err, test.err)
			if test.err == sql.ErrTxDone && test.contextErr != nil {
				require.ErrorIs(t, err, test.contextErr)
			} else {
				require.False(t, errors.Is(err, context.Canceled))
				require.False(t, errors.Is(err, context.DeadlineExceeded))
			}
		})
	}
}

type commitFailureDriver struct {
	err error
}

func (d commitFailureDriver) Commit() error    { return d.err }
func (d commitFailureDriver) Rollback() error  { return sql.ErrTxDone }
func (d commitFailureDriver) SavePoint() error { return nil }

type sqlCreator struct{ db *sql.DB }

func (c sqlCreator) Tx(ctx context.Context) (context.Context, Driver, error) {
	tx, err := c.db.BeginTx(ctx, nil)
	return ctx, sqlDriver{tx}, err
}

type sqlDriver struct{ *sql.Tx }

func (sqlDriver) SavePoint() error { return nil }

type rollbackConnector struct{ rolledBack chan struct{} }

func (c rollbackConnector) Connect(context.Context) (driver.Conn, error) {
	return rollbackConnection(c), nil
}
func (c rollbackConnector) Driver() driver.Driver { return rollbackDriver{c} }

type rollbackDriver struct{ rollbackConnector }

func (d rollbackDriver) Open(string) (driver.Conn, error) {
	return rollbackConnection{d.rolledBack}, nil
}

type rollbackConnection struct{ rolledBack chan struct{} }

func (c rollbackConnection) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unused") }
func (c rollbackConnection) Close() error                        { return nil }
func (c rollbackConnection) Begin() (driver.Tx, error) {
	return rollbackTransaction(c), nil
}

type rollbackTransaction struct{ rolledBack chan struct{} }

func (tx rollbackTransaction) Commit() error   { return nil }
func (tx rollbackTransaction) Rollback() error { close(tx.rolledBack); return nil }
