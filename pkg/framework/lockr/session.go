package lockr

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
)

var (
	ErrNoLockAcquired = errors.New("lock could not be acquired")
	ErrNoLockReleased = errors.New("lock could not be released")
)

type Releaser func(context.Context) error

type SessionLockerConfig struct {
	Logger   *slog.Logger
	DBClient *db.Client
}

type SessionLocker struct {
	logger *slog.Logger
	db     *db.Client
}

func NewSessionLockr(config SessionLockerConfig) *SessionLocker {
	return &SessionLocker{
		logger: config.Logger,
		db:     config.DBClient,
	}
}

func (l *SessionLocker) lock(ctx context.Context, key Key, nonblocking bool) (Releaser, error) {
	lockFunc := "pg_advisory_lock"

	if nonblocking {
		lockFunc = "pg_try_advisory_lock"
	}

	q, args := sql.Dialect(l.db.GetConfig().Driver.Dialect()).
		SelectExpr(sql.ExprFunc(func(b *sql.Builder) {
			b.WriteString(lockFunc)
			b.WriteString("(")
			b.Arg(int64(key.Hash64()))
			b.WriteString(")")
		})).
		Query()

	rows, err := l.db.QueryContext(ctx, q, args...)
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

	return func(rCtx context.Context) error {
		return l.Release(rCtx, key)
	}, nil
}

func (l *SessionLocker) TryLock(ctx context.Context, key Key) (Releaser, error) {
	return l.lock(ctx, key, true)
}

func (l *SessionLocker) Lock(ctx context.Context, key Key) (Releaser, error) {
	return l.lock(ctx, key, false)
}

func (l *SessionLocker) Release(ctx context.Context, key Key) error {
	q, args := sql.Dialect(l.db.GetConfig().Driver.Dialect()).
		SelectExpr(sql.ExprFunc(func(b *sql.Builder) {
			b.WriteString("pg_advisory_unlock")
			b.WriteString("(")
			b.Arg(int64(key.Hash64()))
			b.WriteString(")")
		})).
		Query()

	rows, err := l.db.QueryContext(ctx, q, args...)
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
