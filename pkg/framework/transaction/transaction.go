package transaction

import (
	"context"
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
			err = fmt.Errorf("%w: %v", err, rerr)
		}
		return def, err
	}

	// commit the transaction
	err = tx.Commit()
	if err != nil {
		return def, err
	}

	return result, nil
}
