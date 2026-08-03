package transaction

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
)

// Driver is an interface for transaction drivers
type Driver interface {
	Commit() error
	Rollback() error
	SavePoint() error
}

// Able to start a new transaction
type Creator interface {
	Tx(ctx context.Context) (context.Context, Driver, error)
}

// RunWithNoValue the callback inside a transaction with no return value
func RunWithNoValue(ctx context.Context, creator Creator, cb func(ctx context.Context) error) error {
	_, err := Run(ctx, creator, func(ctx context.Context) (interface{}, error) {
		return nil, cb(ctx)
	})
	return err
}

// Runs the callback inside a transaction
func Run[R any](ctx context.Context, creator Creator, cb func(ctx context.Context) (R, error)) (R, error) {
	var def R
	// Make sure we have a transaction
	ctx, tx, err := getTx(ctx, creator)
	if err != nil {
		return def, err
	}

	// Make sure transaction is set on context
	ctx, err = SetDriverOnContext(ctx, tx)
	if _, ok := err.(*DriverConflictError); !ok && err != nil {
		return def, fmt.Errorf("unknown error %w", err)
	}

	// Execute the callback and manage the transaction
	return manage(ctx, tx, func(ctx context.Context, tx Driver) (R, error) {
		return cb(ctx)
	})
}

// RunInNewTransaction starts and commits a transaction independently of any
// transaction in ctx. For Ent-backed adapters, it acquires a separate database
// connection and shadows the caller's transaction in the callback.
//
// WARNING: The callback's writes are not atomic with the caller. They remain
// committed if the caller later rolls back, and the callback cannot observe the
// caller's uncommitted writes. Calling this while the caller holds locks needed
// by the callback can deadlock. Concurrent use can also exhaust the connection
// pool because an operation may hold one connection while acquiring another.
//
// Use only when the domain explicitly requires a durable side effect outside
// the caller's transaction and the visibility, locking, and connection-pool
// risks have been reviewed.
func RunInNewTransaction[R any](ctx context.Context, creator Creator, cb func(ctx context.Context) (R, error)) (R, error) {
	var def R

	ctx, tx, err := creator.Tx(ctx)
	if err != nil {
		return def, fmt.Errorf("failed to start transaction: %w", err)
	}

	ctx = withDriver(ctx, tx)

	return manage(ctx, tx, func(ctx context.Context, tx Driver) (R, error) {
		return cb(ctx)
	})
}

// Returns the current transaction from the context or creates a new one
func getTx(ctx context.Context, creator Creator) (context.Context, Driver, error) {
	if tx, err := GetDriverFromContext(ctx); err == nil {
		return ctx, tx, nil
	} else {
		if _, ok := err.(*DriverNotFoundError); !ok {
			slog.Debug("failed to get transaction from context", "transaction_error", err)
		}
		ctx, tx, err := creator.Tx(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to start transaction: %w", err)
		}
		return ctx, tx, err
	}
}

// Manages the transaction based on the behavior of the callback
func manage[R any](ctx context.Context, tx Driver, cb func(ctx context.Context, tx Driver) (R, error)) (R, error) {
	var def R
	defer func() {
		if r := recover(); r != nil {
			pMsg := fmt.Sprintf("%v:\n%s", r, debug.Stack())

			// roll back the tx for all downstream (WithTx) clients
			_ = tx.Rollback()
			panic(pMsg)
		}
	}()

	err := tx.SavePoint()
	if err != nil {
		return def, err
	}

	result, err := cb(ctx, tx)
	if err != nil {
		// roll back the tx for all downstream (WithTx) clients
		if rerr := tx.Rollback(); rerr != nil {
			err = errors.Join(err, rerr)
		}

		return def, err
	}

	// commit the transaction
	err = tx.Commit()
	if err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			err = errors.Join(err, rerr)
		}
		return def, err
	}

	return result, nil
}
