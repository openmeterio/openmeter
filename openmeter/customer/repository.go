package customer

import (
	"context"
	"fmt"

	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type TxRepository interface {
	CustomerRepository

	Commit() error
	Rollback() error
}

type Repository interface {
	CustomerRepository

	WithTx(context.Context) (TxRepository, error)
}

type CustomerRepository interface {
	ListCustomers(ctx context.Context, params customerentity.ListCustomersInput) (pagination.PagedResponse[customerentity.Customer], error)
	CreateCustomer(ctx context.Context, params customerentity.CreateCustomerInput) (*customerentity.Customer, error)
	DeleteCustomer(ctx context.Context, customer customerentity.DeleteCustomerInput) error
	GetCustomer(ctx context.Context, customer customerentity.GetCustomerInput) (*customerentity.Customer, error)
	UpdateCustomer(ctx context.Context, params customerentity.UpdateCustomerInput) (*customerentity.Customer, error)
}

func WithTxNoValue(ctx context.Context, repo Repository, fn func(ctx context.Context, repo TxRepository) error) error {
	var err error

	wrapped := func(ctx context.Context, repo TxRepository) (interface{}, error) {
		if err = fn(ctx, repo); err != nil {
			return nil, err
		}

		return nil, nil
	}

	_, err = WithTx[any](ctx, repo, wrapped)

	return err
}

func WithTx[T any](ctx context.Context, repo Repository, fn func(ctx context.Context, repo TxRepository) (T, error)) (resp T, err error) {
	var txRepo TxRepository

	txRepo, err = repo.WithTx(ctx)
	if err != nil {
		return resp, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from panic: %v: %w", r, err)

			if e := txRepo.Rollback(); e != nil {
				err = fmt.Errorf("failed to rollback transaction: %w: %w", e, err)
			}

			return
		}

		if err != nil {
			if e := txRepo.Rollback(); e != nil {
				err = fmt.Errorf("failed to rollback transaction: %w: %w", e, err)
			}

			return
		}

		if e := txRepo.Commit(); e != nil {
			err = fmt.Errorf("failed to commit transaction: %w", e)
		}
	}()

	resp, err = fn(ctx, txRepo)
	if err != nil {
		err = fmt.Errorf("failed to execute transaction: %w", err)
		return
	}

	return
}
