package httperrors_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type sqlCreator struct{ db *sql.DB }

func (c sqlCreator) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	tx, err := c.db.BeginTx(ctx, nil)
	return ctx, sqlDriver{tx}, err
}

type sqlDriver struct{ *sql.Tx }

func (sqlDriver) SavePoint() error { return nil }

func TestPostgresAutoRollbackCancellation(t *testing.T) {
	dsn := os.Getenv("OPENMETER_E2E_POSTGRES_URL")
	if dsn == "" {
		t.Skip("OPENMETER_E2E_POSTGRES_URL not set")
	}
	for _, http2 := range []bool{false, true} {
		t.Run(fmt.Sprintf("http2=%t", http2), func(t *testing.T) {
			// given an HTTP operation using a real PostgreSQL transaction
			db, err := sql.Open("pgx", dsn)
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })
			db.SetMaxOpenConns(1)
			require.NoError(t, db.PingContext(t.Context()))
			started := make(chan struct{})
			server, completed := serveErrors(t, http2, func(diagnostics httptransport.ErrorHandler) http.Handler {
				return httptransport.NewHandler(
					func(context.Context, *http.Request) (struct{}, error) { return struct{}{}, nil },
					func(ctx context.Context, _ struct{}) (struct{}, error) {
						return transaction.Run(ctx, sqlCreator{db}, func(ctx context.Context) (struct{}, error) {
							driver, err := transaction.GetDriverFromContext(ctx)
							if err != nil {
								return struct{}{}, err
							}
							if _, err := driver.(sqlDriver).ExecContext(ctx, "SELECT 1"); err != nil {
								return struct{}{}, err
							}
							close(started)
							<-ctx.Done()
							// Returning the sole connection to the pool proves automatic rollback
							// completed before transaction.Run calls Commit. No fake driver error.
							timeout := time.NewTimer(5 * time.Second)
							defer timeout.Stop()
							tick := time.NewTicker(time.Millisecond)
							defer tick.Stop()
							for db.Stats().InUse != 0 {
								select {
								case <-tick.C:
								case <-timeout.C:
									return struct{}{}, errors.New("automatic rollback did not complete")
								}
							}
							return struct{}{}, nil
						})
					}, commonhttp.JSONResponseEncoder[struct{}],
					httptransport.WithErrorHandler(diagnostics),
				)
			})

			// when the caller disconnects before the operation commits
			cancelRequest(t, server, http2, started, false)

			// then ErrTxDone retains its cancellation cause through the HTTP boundary
			observed := receive(t, completed)
			require.Equal(t, 499, observed.status)
			require.Empty(t, observed.body)
			require.ErrorIs(t, observed.diagnostic, context.Canceled)
			require.ErrorIs(t, observed.diagnostic, sql.ErrTxDone)
			requireLogLevel(t, observed, "WARN")
		})
	}
}
