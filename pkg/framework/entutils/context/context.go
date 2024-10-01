package context

import (
	"context"
	"fmt"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
)

type DB struct {
	db *entdb.Client
}

func NewClient(db *entdb.Client) DB {
	return DB{db: db}
}

func (r DB) Client(ctx context.Context) *entdb.Client {
	client := entdb.FromContext(ctx)
	if client != nil {
		return client
	}

	return r.db
}

func (r DB) ClientNoTx() *entdb.Client {
	return r.db
}

func (r DB) Tx(ctx context.Context) (context.Context, error) {
	// If there is already a transaction in the context, we don't need to create a new one
	tx := entdb.TxFromContext(ctx)
	if tx != nil {
		return ctx, nil
	}

	tx, err := r.db.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	ctx = entdb.NewTxContext(ctx, tx)

	return ctx, nil
}

func (r DB) Commit(ctx context.Context) error {
	tx := entdb.TxFromContext(ctx)
	if tx != nil {
		return tx.Commit()
	}

	return nil
}

func (r DB) Rollback(ctx context.Context) error {
	tx := entdb.TxFromContext(ctx)
	if tx != nil {
		return tx.Rollback()
	}

	return nil
}

func WithTxNoValue(ctx context.Context, client DB, fn func(ctx context.Context) error) error {
	var err error

	wrapped := func(ctx context.Context) (interface{}, error) {
		if err = fn(ctx); err != nil {
			return nil, err
		}

		return nil, nil
	}

	_, err = WithTx(ctx, client, wrapped)

	return err
}

func WithTx[T any](ctx context.Context, client DB, fn func(ctx context.Context) (T, error)) (resp T, err error) {
	if entdb.TxFromContext(ctx) != nil {
		return fn(ctx)
	}

	ctx, err = client.Tx(ctx)
	if err != nil {
		return resp, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from panic: %v: %w", r, err)

			if e := client.Rollback(ctx); e != nil {
				err = fmt.Errorf("failed to rollback transaction: %w: %w", e, err)
			}

			return
		}

		if err != nil {
			if e := client.Rollback(ctx); e != nil {
				err = fmt.Errorf("failed to rollback transaction: %w: %w", e, err)
			}

			return
		}

		if e := client.Commit(ctx); e != nil {
			err = fmt.Errorf("failed to commit transaction: %w", e)
		}
	}()

	resp, err = fn(ctx)
	if err != nil {
		err = fmt.Errorf("failed to execute transaction: %w", err)
		return
	}

	return
}
