package lockr

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type LockerConfig struct {
	Logger *slog.Logger
}

func (c *LockerConfig) Validate() error {
	if c.Logger == nil {
		return fmt.Errorf("logger is required")
	}
	return nil
}

// Locker is the generic interface for distributed business level locks.
type Locker struct {
	cfg *LockerConfig
}

func NewLocker(cfg *LockerConfig) (*Locker, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid locker config: %w", err)
	}

	return &Locker{
		cfg: cfg,
	}, nil
}

// ErrLockTimeout is returned when a lock operation times out
var ErrLockTimeout = errors.New("lock operation timed out")

// LockForTX locks the key for the duration of the transaction.
func (l *Locker) LockForTX(ctx context.Context, key Key) error {
	l.cfg.Logger.DebugContext(ctx, "locking for tx", "key", key.String(), "hash", key.Hash64())
	client, err := l.getTxClient(ctx)
	if err != nil {
		return err
	}

	return l.lock(ctx, client, key)
}

func (l *Locker) LockForTXWithScopes(ctx context.Context, scopes ...string) error {
	k, err := NewKey(scopes...)
	if err != nil {
		return err
	}

	return l.LockForTX(ctx, k)
}

// lock executes the advisory lock query and handles the result set
func (l *Locker) lock(ctx context.Context, client *db.Tx, key Key) error {
	rows, err := client.QueryContext(ctx, "SELECT pg_advisory_xact_lock($1)", int64(key.Hash64()))
	defer func() {
		if rows != nil {
			if e := rows.Close(); e != nil {
				l.cfg.Logger.WarnContext(ctx, "failed to close result set", "error", e)
			}
		}
	}()

	if err != nil {
		return l.checkForTimeout(err)
	}

	// Consume the result set
	for rows.Next() {
		// pg_advisory_xact_lock returns void, but we still need to iterate through rows
	}

	if err := rows.Err(); err != nil {
		return l.checkForTimeout(err)
	}

	return nil
}

// TryLockForTX attempts to acquire the lock without blocking.
// Returns (true, nil) if the lock was acquired, (false, nil) if it is already held by another session,
// or (false, err) on error.
func (l *Locker) TryLockForTX(ctx context.Context, key Key) (bool, error) {
	l.cfg.Logger.DebugContext(ctx, "try locking for tx", "key", key.String(), "hash", key.Hash64())
	client, err := l.getTxClient(ctx)
	if err != nil {
		return false, err
	}

	return l.tryLock(ctx, client, key)
}

// TryLockForTXWithScopes is a convenience method that creates a key from the given scopes and calls TryLockForTX.
func (l *Locker) TryLockForTXWithScopes(ctx context.Context, scopes ...string) (bool, error) {
	k, err := NewKey(scopes...)
	if err != nil {
		return false, err
	}

	return l.TryLockForTX(ctx, k)
}

// tryLock executes the non-blocking advisory lock query and returns whether the lock was acquired.
func (l *Locker) tryLock(ctx context.Context, client *db.Tx, key Key) (bool, error) {
	rows, err := client.QueryContext(ctx, "SELECT pg_try_advisory_xact_lock($1)", int64(key.Hash64()))
	defer func() {
		if rows != nil {
			if e := rows.Close(); e != nil {
				l.cfg.Logger.WarnContext(ctx, "failed to close result set", "error", e)
			}
		}
	}()

	if err != nil {
		return false, err
	}

	var acquired bool
	for rows.Next() {
		if err := rows.Scan(&acquired); err != nil {
			return false, fmt.Errorf("failed to scan try lock result: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return false, err
	}

	return acquired, nil
}

// Note: it would be great to use in-process timeouts with context.WithTimeout
// Unfortunately, due to this https://github.com/jackc/pgx/issues/2100#issuecomment-2395092552 (context cancellation resulting in query cancellation resulting in errored tx states) we rely on the pg timeout which leaves the connection intact
func (l *Locker) checkForTimeout(err error) error {
	if strings.Contains(err.Error(), pgLockTimeoutErrCode) {
		return ErrLockTimeout
	}
	return err
}

func (l *Locker) getTxClient(ctx context.Context) (*db.Tx, error) {
	// If we're not in a transaction this method has to fail
	tx, err := entutils.GetDriverFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("lockr only works in a transaction, but driver not found: %w", err)
	}

	client := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())

	rows, err := client.QueryContext(ctx, "SELECT transaction_timestamp() != statement_timestamp()")
	if err != nil {
		return nil, fmt.Errorf("failed to check transaction status: %w", err)
	}

	defer func() {
		if rows != nil {
			if e := rows.Close(); e != nil {
				l.cfg.Logger.WarnContext(ctx, "failed to close result set", "error", e)
			}
		}
	}()

	var isInTransaction bool
	for rows.Next() {
		err = rows.Scan(&isInTransaction)
		if err != nil {
			return nil, fmt.Errorf("failed to check transaction status: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to check transaction status: %w", err)
	}

	if !isInTransaction {
		return nil, fmt.Errorf("lockr only works in a postgres transaction")
	}

	return client, nil
}

type noopTxCreator struct{}

var _ transaction.Creator = (*noopTxCreator)(nil)

func (n *noopTxCreator) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	return ctx, nil, fmt.Errorf("a transaction should already be accessible from the context")
}
