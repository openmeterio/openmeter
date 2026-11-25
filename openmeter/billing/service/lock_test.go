package billingservice

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	billingadapter "github.com/openmeterio/openmeter/openmeter/billing/adapter"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func TestCustomerLockReusesCallerConnection(t *testing.T) {
	// given: a missing lock row and a pool with only one connection.
	svc, testDB := newCustomerLockTestService(t)
	testDB.PGDriver.DB().SetMaxOpenConns(1)
	customerID := customer.CustomerID{Namespace: "default", ID: ulid.Make().String()}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	// when: billing locks the same customer twice inside a caller-owned transaction.
	err := transaction.RunWithNoValue(ctx, svc.adapter, func(ctx context.Context) error {
		return svc.WithLock(ctx, customerID, func(ctx context.Context) error {
			return transactionForInvoiceManipulationNoValue(ctx, svc, customerID, func(context.Context) error {
				return nil
			})
		})
	})

	// then: both acquisitions complete on that connection and commit one lock row.
	require.NoError(t, err)
	count, err := testDB.EntDriver.Client().BillingCustomerLock.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Zero(t, testDB.PGDriver.DB().Stats().WaitCount)
}

func TestCustomerLockCreationRollsBack(t *testing.T) {
	// given: the customer's first billing operation will fail after taking its lock.
	svc, testDB := newCustomerLockTestService(t)
	customerID := customer.CustomerID{Namespace: "default", ID: ulid.Make().String()}
	operationErr := errors.New("billing operation failed")

	// when: the operation aborts the billing transaction.
	err := transactionForInvoiceManipulationNoValue(t.Context(), svc, customerID, func(context.Context) error {
		return operationErr
	})

	// then: creating the synchronization row rolls back with the operation.
	require.ErrorIs(t, err, operationErr)
	count, err := testDB.EntDriver.Client().BillingCustomerLock.Query().Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestCustomerLockSerializesConcurrentTransactions(t *testing.T) {
	operationErr := errors.New("first billing operation failed")
	cases := []struct {
		name     string
		setup    func(*testing.T, *testutils.TestDB, customer.CustomerID)
		firstErr error
		// blockedOn is the statement the second transaction waits in. Only a waiter on the
		// row lock keeps its place in line; a waiter on the insert has to lock afterwards.
		blockedOn string
	}{
		{
			name:      "concurrent first use commits",
			blockedOn: "INSERT",
		},
		{
			name:      "concurrent first use rolls back",
			firstErr:  operationErr,
			blockedOn: "INSERT",
		},
		{
			name:      "existing lock row",
			blockedOn: "SELECT",
			setup: func(t *testing.T, testDB *testutils.TestDB, customerID customer.CustomerID) {
				err := testDB.EntDriver.Client().BillingCustomerLock.Create().
					SetNamespace(customerID.Namespace).
					SetCustomerID(customerID.ID).
					Exec(t.Context())
				require.NoError(t, err)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given: two billing operations target the same customer.
			svc, testDB := newCustomerLockTestService(t)
			customerID := customer.CustomerID{Namespace: "default", ID: ulid.Make().String()}
			if tc.setup != nil {
				tc.setup(t, testDB, customerID)
			}

			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			var workers sync.WaitGroup
			t.Cleanup(func() {
				cancel()
				workers.Wait()
			})

			firstAcquired := make(chan struct{})
			releaseFirst := make(chan struct{})
			firstResult := make(chan error, 1)
			workers.Go(func() {
				firstResult <- transactionForInvoiceManipulationNoValue(ctx, svc, customerID, func(ctx context.Context) error {
					close(firstAcquired)
					select {
					case <-releaseFirst:
						return tc.firstErr
					case <-ctx.Done():
						return ctx.Err()
					}
				})
			})

			select {
			case <-firstAcquired:
			case err := <-firstResult:
				t.Fatalf("first transaction did not acquire the lock: %v", err)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}

			// when: another transaction tries to lock the customer before the first finishes.
			secondAcquired := make(chan struct{})
			secondResult := make(chan error, 1)
			workers.Go(func() {
				secondResult <- transactionForInvoiceManipulationNoValue(ctx, svc, customerID, func(context.Context) error {
					close(secondAcquired)
					return nil
				})
			})

			// then: PostgreSQL reports a blocked transaction, not just an unscheduled goroutine.
			require.EventuallyWithT(t, func(c *assert.CollectT) {
				rows, err := testDB.PGDriver.DB().QueryContext(ctx, `
					SELECT query FROM pg_stat_activity
					WHERE datname = current_database()
					  AND wait_event_type = 'Lock'
					  AND cardinality(pg_blocking_pids(pid)) > 0
				`)
				if !assert.NoError(c, err) {
					return
				}

				defer rows.Close()

				var blocked []string
				for rows.Next() {
					var query string
					if !assert.NoError(c, rows.Scan(&query)) {
						return
					}

					blocked = append(blocked, query)
				}

				assert.NoError(c, rows.Err())

				if assert.Len(c, blocked, 1) {
					assert.True(c, strings.HasPrefix(blocked[0], tc.blockedOn), "blocked on %q", blocked[0])
				}
			}, 5*time.Second, 10*time.Millisecond)

			select {
			case <-secondAcquired:
				t.Fatal("second transaction acquired the customer lock before the first finished")
			default:
			}

			close(releaseFirst)
			require.ErrorIs(t, <-firstResult, tc.firstErr)
			require.NoError(t, <-secondResult)
			select {
			case <-secondAcquired:
			default:
				t.Fatal("second transaction did not acquire the customer lock")
			}

			count, err := testDB.EntDriver.Client().BillingCustomerLock.Query().Count(ctx)
			require.NoError(t, err)
			require.Equal(t, 1, count)
		})
	}
}

func TestCustomerLockPropagatesInsertFailure(t *testing.T) {
	// given: PostgreSQL rejects creation of the synchronization row.
	svc, testDB := newCustomerLockTestService(t)
	_, err := testDB.PGDriver.DB().ExecContext(t.Context(), `
		ALTER TABLE billing_customer_locks
		ADD CONSTRAINT reject_lock_creation CHECK (namespace <> 'rejected')
	`)
	require.NoError(t, err)
	customerID := customer.CustomerID{Namespace: "rejected", ID: ulid.Make().String()}

	// when: billing attempts to acquire the customer lock.
	called := false
	err = transactionForInvoiceManipulationNoValue(t.Context(), svc, customerID, func(context.Context) error {
		called = true
		return nil
	})

	// then: the original constraint error reaches the caller and billing never starts.
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, "23514", pgErr.Code)
	require.Equal(t, "reject_lock_creation", pgErr.ConstraintName)
	require.False(t, called)
}

func newCustomerLockTestService(t *testing.T) (*Service, *testutils.TestDB) {
	t.Helper()

	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	t.Cleanup(func() {
		testDB.Close(t)
	})
	logger := testutils.NewLogger(t)
	adapter, err := billingadapter.New(billingadapter.Config{
		Client: testDB.EntDriver.Client(),
		Logger: logger,
	})
	require.NoError(t, err)

	return &Service{
		adapter: adapter,
		logger:  logger,
	}, testDB
}
