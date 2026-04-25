package lockr

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
)

var (
	ErrNoLockAcquired    = errors.New("lock could not be acquired")
	ErrNoLockReleased    = errors.New("lock could not be released")
	ErrSessionLockerDone = errors.New("session locker is already closed")
	ErrSessionLockerBusy = errors.New("session locker is blocked by another lock request")
)

type Releaser func(context.Context) error

type releaser struct {
	mu     sync.Mutex
	done   bool
	locker *SessionLocker
	key    Key
}

func (r *releaser) release(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.done {
		return nil
	}

	rErr := r.locker.release(ctx, r.key)
	if rErr != nil {
		if !errors.Is(rErr, ErrNoLockReleased) && !errors.Is(rErr, ErrSessionLockerDone) {
			return rErr
		}
	}

	r.done = true

	// Release references to locker and key so they can be GC'd
	r.locker = nil
	r.key = nil

	return rErr
}

type SessionLockerConfig struct {
	Logger         *slog.Logger
	PostgresDriver *pgdriver.Driver
}

// SessionLocker is a locker that uses PostgreSQL advisory locks to acquire locks.
// It requires a dedicated connection to acquire locks.
type SessionLocker struct {
	logger *slog.Logger
	conn   *sql.Conn

	closed atomic.Bool
	closer func()
	mu     sync.Mutex
}

func NewSessionLockr(ctx context.Context, config SessionLockerConfig) (*SessionLocker, error) {
	if config.Logger == nil {
		return nil, errors.New("logger is required")
	}

	if config.PostgresDriver == nil {
		return nil, errors.New("postgres driver is required")
	}

	conn, err := config.PostgresDriver.DB().Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get postgres connection: %w", err)
	}

	id := clock.Now().UTC().UnixNano()

	logger := config.Logger.With("component", "session-lockr", "id", id)

	closer := sync.OnceFunc(func() {
		if err := conn.Close(); err != nil {
			logger.Error("failed to close postgres connection", "error", err)
		}
	})

	return &SessionLocker{
		logger: logger,
		conn:   conn,
		closer: closer,
	}, nil
}

func (l *SessionLocker) lock(ctx context.Context, key Key, nonblocking bool) (Releaser, error) {
	if l.closed.Load() {
		return nil, ErrSessionLockerDone
	}

	lockFunc := "pg_advisory_lock"

	if nonblocking {
		lockFunc = "pg_try_advisory_lock"
	}

	q, args := entsql.Dialect(dialect.Postgres).
		SelectExpr(entsql.ExprFunc(func(b *entsql.Builder) {
			b.WriteString(lockFunc)
			b.WriteString("(")
			b.Arg(int64(key.Hash64()))
			b.WriteString(")")
		})).
		Query()

	rows, err := l.conn.QueryContext(ctx, q, args...)
	defer func() {
		if rows != nil {
			if err := rows.Close(); err != nil {
				l.logger.Warn("failed to close session-level advisory lock result", "error", err)
			}
		}
	}()

	if err != nil {
		return nil, fmt.Errorf("failed to acquire session-level advisory lock: %w", checkForTimeout(err))
	}

	var lockAcquired bool

	if nonblocking {
		for rows.Next() {
			if err := rows.Scan(&lockAcquired); err != nil {
				return nil, fmt.Errorf("failed to scan session-level advisory lock result: %w", err)
			}
		}
	} else {
		lockAcquired = true

		for rows.Next() {
		}
	}

	if err = rows.Err(); err != nil {
		return nil, checkForTimeout(err)
	}

	if !lockAcquired {
		return nil, ErrNoLockAcquired
	}

	r := &releaser{
		locker: l,
		key:    key,
	}

	return r.release, nil
}

// TryLock attempts to acquire a lock for the given key in a non-blocking way and returns a Releaser that can be used
// to release the lock if it is successfully acquired. The ErrNoLockAcquired is acquiring the lock is denied by the database server.
// It may return ErrSessionLockerBusy if the SessionLocker is blocked by another caller, indicating that the lock request may be retried.
// The ErrSessionLockerDone is returned if SessionLocker is closed, meaning it cannot be used for acquiring locks.
func (l *SessionLocker) TryLock(ctx context.Context, key Key) (Releaser, error) {
	mutexLocked := l.mu.TryLock()
	if !mutexLocked {
		return nil, ErrSessionLockerBusy
	}

	defer l.mu.Unlock()

	return l.lock(ctx, key, true)
}

// Lock blocks until a lock is acquired and returns a Releaser that can be used to release the lock if it is successfully acquired.
// The ErrNoLockAcquired is acquiring the lock is denied by the database server.
// The ErrSessionLockerDone is returned if SessionLocker is closed, meaning it cannot be used for acquiring locks.
func (l *SessionLocker) Lock(ctx context.Context, key Key) (Releaser, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.lock(ctx, key, false)
}

func (l *SessionLocker) release(ctx context.Context, key Key) error {
	if l.closed.Load() {
		return ErrSessionLockerDone
	}

	q, args := entsql.Dialect(dialect.Postgres).
		SelectExpr(entsql.ExprFunc(func(b *entsql.Builder) {
			b.WriteString("pg_advisory_unlock")
			b.WriteString("(")
			b.Arg(int64(key.Hash64()))
			b.WriteString(")")
		})).
		Query()

	rows, err := l.conn.QueryContext(ctx, q, args...)
	defer func() {
		if rows != nil {
			if err = rows.Close(); err != nil {
				l.logger.Warn("failed to close session-level advisory lock result", "error", err)
			}
		}
	}()

	if err != nil {
		return fmt.Errorf("failed to release session-level advisory lock: %w", checkForTimeout(err))
	}

	var lockReleased bool

	for rows.Next() {
		if err = rows.Scan(&lockReleased); err != nil {
			return fmt.Errorf("failed to scan session-level advisory lock release result: %w", err)
		}
	}

	if err = rows.Err(); err != nil {
		return checkForTimeout(err)
	}

	if !lockReleased {
		return ErrNoLockReleased
	}

	return nil
}

// Close releases all locks held by the SessionLocker and closes the underlying database connection.
func (l *SessionLocker) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed.Load() {
		return
	}

	l.closer()
	l.closed.Store(true)

	// Release references to conn so it can be GC'd
	l.conn = nil
}
