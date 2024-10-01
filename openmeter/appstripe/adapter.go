package appstripe

import (
	"context"
	"fmt"

	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
)

type TxAdapter interface {
	AppStripeAdapter

	Commit() error
	Rollback() error
}

type Adapter interface {
	AppStripeAdapter

	WithTx(context.Context) (context.Context, error)
	Rollback(ctx context.Context) error
	Commit(ctx context.Context) error
}

type AppStripeAdapter interface {
	CreateStripeApp(ctx context.Context, input appstripeentity.CreateAppStripeInput) (appstripeentity.App, error)
	UpsertStripeCustomerData(ctx context.Context, input appstripeentity.UpsertStripeCustomerDataInput) error
	DeleteStripeCustomerData(ctx context.Context, input appstripeentity.DeleteStripeCustomerDataInput) error
}

func WithTxNoValue(ctx context.Context, adapter Adapter, fn func(ctx context.Context) error) error {
	var err error

	wrapped := func(ctx context.Context) (interface{}, error) {
		if err = fn(ctx); err != nil {
			return nil, err
		}

		return nil, nil
	}

	_, err = WithTx(ctx, adapter, wrapped)

	return err
}

func WithTx[T any](ctx context.Context, adapter Adapter, fn func(ctx context.Context) (T, error)) (resp T, err error) {
	if entdb.TxFromContext(ctx) != nil {
		return fn(ctx)
	}

	ctx, err = adapter.WithTx(ctx)
	if err != nil {
		return resp, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from panic: %v: %w", r, err)

			if e := adapter.Rollback(ctx); e != nil {
				err = fmt.Errorf("failed to rollback transaction: %w: %w", e, err)
			}

			return
		}

		if err != nil {
			if e := adapter.Rollback(ctx); e != nil {
				err = fmt.Errorf("failed to rollback transaction: %w: %w", e, err)
			}

			return
		}

		if e := adapter.Commit(ctx); e != nil {
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
