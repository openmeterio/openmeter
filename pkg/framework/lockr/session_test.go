package lockr

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
)

func newTestSessionLocker(t *testing.T, dbConn string, opts ...pgdriver.Option) *SessionLocker {
	t.Helper()

	postgresDriver, err := pgdriver.NewPostgresDriver(t.Context(), dbConn, opts...)
	if err != nil {
		t.Fatalf("failed to get postgres driver: %s", err)
	}

	t.Cleanup(func() {
		if err := postgresDriver.Close(); err != nil {
			t.Errorf("failed to close postgres driver: %v", err)
		}
	})

	locker, err := NewSessionLockr(SessionLockerConfig{
		Logger:         testutils.NewLogger(t),
		PostgresDriver: postgresDriver,
	})
	require.NoError(t, err)

	return locker
}

func Test_SessionLocker(t *testing.T) {
	testDB := testutils.InitPostgresDB(t)
	t.Cleanup(func() {
		testDB.Close(t)
	})

	t.Run("Lock and release", func(t *testing.T) {
		locker := newTestSessionLocker(t, testDB.URL)
		defer locker.Close()

		k, err := NewKey("test", "lock-release")
		require.NoError(t, err)

		releaser, err := locker.Lock(t.Context(), k)
		require.NoError(t, err)
		require.NotNil(t, releaser)

		err = releaser(t.Context())
		require.NoError(t, err)
	})

	t.Run("TryLock and release", func(t *testing.T) {
		locker := newTestSessionLocker(t, testDB.URL)
		defer locker.Close()

		k, err := NewKey("test", "trylock-release")
		require.NoError(t, err)

		releaser, err := locker.TryLock(t.Context(), k)
		require.NoError(t, err)
		require.NotNil(t, releaser)

		err = releaser(t.Context())
		require.NoError(t, err)
	})

	t.Run("Same session can acquire the same lock twice", func(t *testing.T) {
		locker := newTestSessionLocker(t, testDB.URL)
		defer locker.Close()

		k, err := NewKey("test", "reentrant")
		require.NoError(t, err)

		releaser1, err := locker.Lock(t.Context(), k)
		require.NoError(t, err)

		releaser2, err := locker.Lock(t.Context(), k)
		require.NoError(t, err)

		// PostgreSQL session-level advisory locks are reentrant: each acquisition
		// increments a counter and requires a matching unlock to fully release.
		require.NoError(t, releaser2(t.Context()))
		require.NoError(t, releaser1(t.Context()))
	})

	t.Run("TryLock fails when lock is held by another session", func(t *testing.T) {
		locker1 := newTestSessionLocker(t, testDB.URL)
		defer locker1.Close()

		locker2 := newTestSessionLocker(t, testDB.URL)
		defer locker2.Close()

		k, err := NewKey("test", "trylock-contention")
		require.NoError(t, err)

		// Session 1 acquires the lock
		releaser, err := locker1.Lock(t.Context(), k)
		require.NoError(t, err)

		// Session 2 tries to acquire the same lock non-blocking
		_, err = locker2.TryLock(t.Context(), k)
		require.ErrorIs(t, err, ErrNoLockAcquired)

		// After session 1 releases, session 2 can acquire
		require.NoError(t, releaser(t.Context()))

		releaser2, err := locker2.TryLock(t.Context(), k)
		require.NoError(t, err)
		require.NoError(t, releaser2(t.Context()))
	})

	t.Run("Different keys do not conflict", func(t *testing.T) {
		locker1 := newTestSessionLocker(t, testDB.URL)
		defer locker1.Close()

		locker2 := newTestSessionLocker(t, testDB.URL)
		defer locker2.Close()

		key1, err := NewKey("test", "key-a")
		require.NoError(t, err)

		key2, err := NewKey("test", "key-b")
		require.NoError(t, err)

		// Both sessions acquire different locks concurrently
		releaser1, err := locker1.Lock(t.Context(), key1)
		require.NoError(t, err)

		releaser2, err := locker2.Lock(t.Context(), key2)
		require.NoError(t, err)

		require.NoError(t, releaser1(t.Context()))
		require.NoError(t, releaser2(t.Context()))
	})

	t.Run("Lock blocks until released by another session", func(t *testing.T) {
		locker1 := newTestSessionLocker(t, testDB.URL)
		defer locker1.Close()

		locker2 := newTestSessionLocker(t, testDB.URL)
		defer locker2.Close()

		k, err := NewKey("test", "blocking")
		require.NoError(t, err)

		// Session 1 acquires the lock
		releaser1, err := locker1.Lock(t.Context(), k)
		require.NoError(t, err)

		// Track ordering of operations
		events := make(chan string, 4)

		var wg sync.WaitGroup
		wg.Add(1)

		waitCh := make(chan int)

		// Session 2 blocks trying to acquire the same lock
		go func() {
			defer wg.Done()

			events <- "s2 waiting"
			time.Sleep(50 * time.Millisecond)
			close(waitCh)

			releaser2, err := locker2.Lock(t.Context(), k)
			assert.NoError(t, err)
			events <- "s2 acquired"

			if releaser2 != nil {
				assert.NoError(t, releaser2(t.Context()))
			}
		}()

		// Wait until session 2 is blocked
		assert.Eventually(t, func() bool {
			select {
			case <-waitCh:
				return true
			default:
				t.Log("waiting for session 2 to block")
				return false
			}
		}, time.Second, 10*time.Millisecond)

		events <- "s1 releasing"
		require.NoError(t, releaser1(t.Context()))

		wg.Wait()
		close(events)

		var results []string
		for e := range events {
			results = append(results, e)
		}

		require.Equal(t, []string{"s2 waiting", "s1 releasing", "s2 acquired"}, results)
	})

	t.Run("Release without lock is a no-op", func(t *testing.T) {
		locker := newTestSessionLocker(t, testDB.URL)
		defer locker.Close()

		key, err := NewKey("test", "release-noop")
		require.NoError(t, err)

		// Releasing a lock that was never acquired should not error
		err = locker.Release(t.Context(), key)
		require.NoError(t, err)
	})

	t.Run("Lock respects context cancellation", func(t *testing.T) {
		locker1 := newTestSessionLocker(t, testDB.URL)
		defer locker1.Close()

		locker2 := newTestSessionLocker(t, testDB.URL)
		defer locker2.Close()

		k, err := NewKey("test", "ctx-cancel")
		require.NoError(t, err)

		// Session 1 holds the lock
		releaser, err := locker1.Lock(t.Context(), k)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = releaser(t.Context())
		})

		// Session 2 tries to acquire with a short-lived context
		ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
		defer cancel()

		_, err = locker2.Lock(ctx, k)
		require.Error(t, err)
	})

	t.Run("Lock timeout returns ErrLockTimeout", func(t *testing.T) {
		lockTimeout := 2 * time.Second
		opts := []pgdriver.Option{
			pgdriver.WithLockTimeout(lockTimeout),
		}

		locker1 := newTestSessionLocker(t, testDB.URL, opts...)
		defer locker1.Close()

		locker2 := newTestSessionLocker(t, testDB.URL, opts...)
		defer locker2.Close()

		k, err := NewKey("test", "timeout")
		require.NoError(t, err)

		// Session 1 holds the lock
		releaser, err := locker1.Lock(t.Context(), k)
		require.NoError(t, err)

		done := make(chan struct{})

		go func() {
			defer close(done)

			// Session 2 blocks and eventually times out via PostgreSQL lock_timeout
			_, err := locker2.Lock(t.Context(), k)
			assert.ErrorIs(t, err, ErrLockTimeout)
		}()

		// Wait for session 2 to time out, then release
		assert.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				t.Log("waiting for session 2 to block")
				return false
			}
		}, 3*lockTimeout, lockTimeout)
		require.NoError(t, releaser(t.Context()))
	})

	t.Run("Multiple locks held and released independently", func(t *testing.T) {
		locker := newTestSessionLocker(t, testDB.URL)
		defer locker.Close()

		key1, err := NewKey("test", "multi-a")
		require.NoError(t, err)

		key2, err := NewKey("test", "multi-b")
		require.NoError(t, err)

		key3, err := NewKey("test", "multi-c")
		require.NoError(t, err)

		releaser1, err := locker.Lock(t.Context(), key1)
		require.NoError(t, err)

		releaser2, err := locker.Lock(t.Context(), key2)
		require.NoError(t, err)

		releaser3, err := locker.Lock(t.Context(), key3)
		require.NoError(t, err)

		// Release in different order than acquired
		require.NoError(t, releaser2(t.Context()))
		require.NoError(t, releaser1(t.Context()))
		require.NoError(t, releaser3(t.Context()))
	})

	t.Run("Releaser only releases lock once", func(t *testing.T) {
		locker1 := newTestSessionLocker(t, testDB.URL)
		defer locker1.Close()

		locker2 := newTestSessionLocker(t, testDB.URL)
		defer locker2.Close()

		k, err := NewKey("test", "release-once")
		require.NoError(t, err)

		// Session 1 acquires the lock twice (reentrant)
		releaser1a, err := locker1.Lock(t.Context(), k)
		require.NoError(t, err)

		releaser1b, err := locker1.Lock(t.Context(), k)
		require.NoError(t, err)

		// Release the second acquisition
		require.NoError(t, releaser1b(t.Context()))

		// Call releaser1b again — should be a no-op (sync.Once), so the first
		// acquisition still holds the lock and session 2 cannot acquire it.
		require.NoError(t, releaser1b(t.Context()))

		// Session 2 should still be unable to acquire because session 1's first
		// lock acquisition has not been released yet.
		_, err = locker2.TryLock(t.Context(), k)
		require.ErrorIs(t, err, ErrNoLockAcquired)

		// Now release the first acquisition
		require.NoError(t, releaser1a(t.Context()))

		// Session 2 can now acquire the lock
		releaser2, err := locker2.TryLock(t.Context(), k)
		require.NoError(t, err)
		require.NoError(t, releaser2(t.Context()))
	})

	t.Run("TryLock succeeds after blocking Lock is released", func(t *testing.T) {
		locker1 := newTestSessionLocker(t, testDB.URL)
		defer locker1.Close()

		locker2 := newTestSessionLocker(t, testDB.URL)
		defer locker2.Close()

		k, err := NewKey("test", "trylock-after-release")
		require.NoError(t, err)

		// Session 1 acquires with blocking Lock
		releaser, err := locker1.Lock(t.Context(), k)
		require.NoError(t, err)

		// Session 2 can't TryLock while held
		_, err = locker2.TryLock(t.Context(), k)
		require.ErrorIs(t, err, ErrNoLockAcquired)

		// Release and retry
		require.NoError(t, releaser(t.Context()))

		releaser2, err := locker2.TryLock(t.Context(), k)
		require.NoError(t, err)
		require.NoError(t, releaser2(t.Context()))
	})
}
