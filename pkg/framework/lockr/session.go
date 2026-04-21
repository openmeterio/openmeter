package lockr

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	dbsql "database/sql"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
)

var (
	ErrNoLockAcquired         = errors.New("lock could not be acquired")
	ErrNoLockReleased         = errors.New("lock could not be released")
	ErrSessionLockerDone      = errors.New("session locker is already closed")
	ErrDatabaseConnectionDown = errors.New("database connection is down")
)

type Releaser func(context.Context) error

type SessionLockerConfig struct {
	Logger         *slog.Logger
	PostgresDriver *pgdriver.Driver
}

type SessionLocker struct {
	logger *slog.Logger
	conn   *dbsql.Conn

	closed atomic.Bool
	closer func()
}

func NewSessionLockr(config SessionLockerConfig) (*SessionLocker, error) {
	if config.Logger == nil {
		return nil, errors.New("logger is required")
	}

	if config.PostgresDriver == nil {
		return nil, errors.New("postgres driver is required")
	}

	conn, err := config.PostgresDriver.DB().Conn(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get postgres connection: %w", err)
	}

	closer := sync.OnceFunc(func() {
		if err := conn.Close(); err != nil {
			config.Logger.Error("failed to close postgres connection", "error", err)
		}
	})

	return &SessionLocker{
		logger: config.Logger,
		conn:   conn,
		closer: closer,
	}, nil
}

func (l *SessionLocker) lock(ctx context.Context, key Key, nonblocking bool) (Releaser, error) {
	if l.closed.Load() {
		return nil, ErrSessionLockerDone
	}

	if err := l.conn.PingContext(ctx); err != nil {
		return nil, ErrDatabaseConnectionDown
	}

	lockFunc := "pg_advisory_lock"

	if nonblocking {
		lockFunc = "pg_try_advisory_lock"
	}

	q, args := sql.Dialect(dialect.Postgres).
		SelectExpr(sql.ExprFunc(func(b *sql.Builder) {
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

	if nonblocking {
		var locked bool

		for rows.Next() {
			if err := rows.Scan(&locked); err != nil {
				return nil, fmt.Errorf("failed to scan session-level advisory lock result: %w", err)
			}
		}

		if !locked {
			return nil, ErrNoLockAcquired
		}
	} else {
		for rows.Next() {
		}
	}

	if err = rows.Err(); err != nil {
		return nil, checkForTimeout(err)
	}

	r := &struct {
		once sync.Once
	}{}

	return func(rCtx context.Context) error {
		var err error

		r.once.Do(func() {
			err = l.Release(rCtx, key)
		})

		return err
	}, nil
}

func (l *SessionLocker) TryLock(ctx context.Context, key Key) (Releaser, error) {
	return l.lock(ctx, key, true)
}

func (l *SessionLocker) Lock(ctx context.Context, key Key) (Releaser, error) {
	return l.lock(ctx, key, false)
}

func (l *SessionLocker) Release(ctx context.Context, key Key) error {
	q, args := sql.Dialect(dialect.Postgres).
		SelectExpr(sql.ExprFunc(func(b *sql.Builder) {
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

	var released bool

	for rows.Next() {
		if err = rows.Scan(&released); err != nil {
			return fmt.Errorf("failed to scan session-level advisory lock release result: %w", err)
		}
	}

	if err = rows.Err(); err != nil {
		return checkForTimeout(err)
	}

	return nil
}

func (l *SessionLocker) Close() {
	l.closer()
	l.closed.Store(true)
}
