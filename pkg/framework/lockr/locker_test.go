package lockr_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/peterldowns/pgtestdb"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entdriver"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

// Lets set up a dummy tx creator

type creator struct {
	db *db.Client
}

var _ transaction.Creator = &creator{}

func (c *creator) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := c.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func TestLockerLockForTx(t *testing.T) {
	// Lets write some testrunning utilities
	var m sync.Mutex

	withDBClient := func(fn func(t *testing.T, client *db.Client)) func(t *testing.T) {
		return func(t *testing.T) {
			// Let's set up a test postgres
			testdb := testutils.InitPostgresDB(t)
			dbClient := testdb.EntDriver.Client()
			pgDriver := testdb.PGDriver
			entDriver := testdb.EntDriver

			defer func() {
				_ = dbClient.Close()
				_ = entDriver.Close()
				_ = pgDriver.Close()
			}()

			// migration tooling is not concurrency safe
			func() {
				m.Lock()
				defer m.Unlock()

				if err := dbClient.Schema.Create(context.Background()); err != nil {
					t.Fatalf("failed to create schema: %v", err)
				}
			}()

			fn(t, dbClient)
		}
	}

	t.Run("Should error if not in a transaction", withDBClient(func(t *testing.T, client *db.Client) {
		locker, err := lockr.NewLocker(&lockr.LockerConfig{
			Logger: testutils.NewLogger(t),
		})
		require.NoError(t, err)

		key, err := lockr.NewKey("test")
		require.NoError(t, err)

		err = locker.LockForTX(context.Background(), key)
		require.Error(t, err)
		require.ErrorContains(t, err, "lockr only works in a transaction, but driver not found")
	}))

	t.Run("Should acquire a lock", withDBClient(func(t *testing.T, client *db.Client) {
		txCreator := &creator{db: client}

		locker, err := lockr.NewLocker(&lockr.LockerConfig{
			Logger: testutils.NewLogger(t),
		})
		require.NoError(t, err)

		require.NoError(t, transaction.RunWithNoValue(context.Background(), txCreator, func(ctx context.Context) error {
			key, err := lockr.NewKey("test")
			if err != nil {
				t.Fatalf("failed to create key: %v", err)
			}

			require.NoError(t, locker.LockForTX(ctx, key))

			return nil
		}))
	}))

	t.Run("Should be able to acquire same lock twice if in same transaction", withDBClient(func(t *testing.T, client *db.Client) {
		txCreator := &creator{db: client}

		locker, err := lockr.NewLocker(&lockr.LockerConfig{
			Logger: testutils.NewLogger(t),
		})
		require.NoError(t, err)

		require.NoError(t, transaction.RunWithNoValue(context.Background(), txCreator, func(ctx context.Context) error {
			key, err := lockr.NewKey("test")
			if err != nil {
				t.Fatalf("failed to create key: %v", err)
			}

			require.NoError(t, locker.LockForTX(ctx, key))

			require.NoError(t, locker.LockForTX(ctx, key))

			return nil
		}))
	}))

	t.Run("Should be able to acquire same lock in sub-transaction", withDBClient(func(t *testing.T, client *db.Client) {
		txCreator := &creator{db: client}

		locker, err := lockr.NewLocker(&lockr.LockerConfig{
			Logger: testutils.NewLogger(t),
		})
		require.NoError(t, err)

		require.NoError(t, transaction.RunWithNoValue(context.Background(), txCreator, func(ctx context.Context) error {
			key, err := lockr.NewKey("test")
			if err != nil {
				t.Fatalf("failed to create key: %v", err)
			}

			require.NoError(t, locker.LockForTX(ctx, key))

			require.NoError(t, transaction.RunWithNoValue(ctx, txCreator, func(ctx context.Context) error {
				require.NoError(t, locker.LockForTX(ctx, key))

				return nil
			}))

			return nil
		}))
	}))

	t.Run("Should wait while acquiring lock from parallel transactions", withDBClient(func(t *testing.T, client *db.Client) {
		txCreator := &creator{db: client}

		locker, err := lockr.NewLocker(&lockr.LockerConfig{
			Logger: testutils.NewLogger(t),
		})
		require.NoError(t, err)

		key, err := lockr.NewKey("test")
		require.NoError(t, err)

		// We run two parallel go routines, each with a transaction, with different delays

		wg := sync.WaitGroup{}
		wg.Add(2)

		finCh := make(chan string, 4)

		go func() {
			defer wg.Done()

			require.NoError(t, transaction.RunWithNoValue(context.Background(), txCreator, func(ctx context.Context) error {
				finCh <- "1 start"

				require.NoError(t, locker.LockForTX(ctx, key))

				// non-blocking sleep for 1 second (we keep the lock for 1 second)
				time.Sleep(1 * time.Second)

				finCh <- "1 done"

				return nil
			}))
		}()

		go func() {
			defer wg.Done()

			require.NoError(t, transaction.RunWithNoValue(context.Background(), txCreator, func(ctx context.Context) error {
				// scheduler works in magical ways, so adding a small delay here to break execution and switch to different go routine
				time.Sleep(10 * time.Millisecond)

				finCh <- "2 start"

				require.NoError(t, locker.LockForTX(ctx, key))

				finCh <- "2 done"

				return nil
			}))
		}()

		wg.Wait()

		// Let's read the contents of the chan to make sure things finished in the correct order
		close(finCh)

		results := []string{}

		for fin := range finCh {
			results = append(results, fin)
		}

		require.Equal(t, []string{"1 start", "2 start", "1 done", "2 done"}, results)
	}))

	t.Run("Should error if acquiring lock takes longer than timeout", func(t *testing.T) {
		// We'll need a custom db setup to configure the timeout
		lockTimeout := time.Second * 3

		host := os.Getenv("POSTGRES_HOST")
		if host == "" {
			t.Skip("POSTGRES_HOST not set")
		}

		// TODO: fix migrations
		dbConf := pgtestdb.Custom(t, pgtestdb.Config{
			DriverName: "pgx",
			User:       "postgres",
			Password:   "postgres",
			Host:       host,
			Port:       "5432",
			Options:    "sslmode=disable",
		}, &testutils.NoopMigrator{})

		pgdrv, err := pgdriver.NewPostgresDriver(
			context.TODO(),
			dbConf.URL(),
			pgdriver.WithLockTimeout(lockTimeout),
		)
		if err != nil {
			t.Fatalf("failed to get pg driver: %s", err)
		}

		client := entdriver.NewEntPostgresDriver(pgdrv.DB()).Client()

		defer func() {
			_ = client.Close()
			_ = pgdrv.Close()

			time.Sleep(1 * time.Second)
		}()

		txCreator := &creator{db: client}

		locker, err := lockr.NewLocker(&lockr.LockerConfig{
			Logger: testutils.NewLogger(t),
		})
		require.NoError(t, err)

		key, err := lockr.NewKey("test")
		require.NoError(t, err)

		// We run two parallel go routines, each with a transaction, with different delays

		wg := sync.WaitGroup{}
		wg.Add(2)

		go func() {
			defer wg.Done()

			require.NoError(t, transaction.RunWithNoValue(context.Background(), txCreator, func(ctx context.Context) error {
				require.NoError(t, locker.LockForTX(ctx, key))

				// non-blocking sleep for 4 seconds (more than 3)
				time.Sleep(lockTimeout + time.Second)

				return nil
			}))
		}()

		go func() {
			defer wg.Done()

			// This will fail as the timeout cancels the context and the client connection
			require.Error(t, transaction.RunWithNoValue(context.Background(), txCreator, func(ctx context.Context) error {
				// scheduler works in magical ways, so adding a small delay here to break execution and switch to different go routine
				time.Sleep(10 * time.Millisecond)

				// We should get a timeout error as we've been trying to get the lock for over 3 second

				err := locker.LockForTX(ctx, key)
				require.Error(t, err)
				require.ErrorIs(t, err, lockr.ErrLockTimeout)

				return err
			}))
		}()

		wg.Wait()
	})
}
