package billingservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ billing.Service = (*Service)(nil)

type Service struct {
	adapter         billing.Adapter
	customerService customer.CustomerService
	appService      app.Service
	logger          *slog.Logger
}

type Config struct {
	Adapter         billing.Adapter
	CustomerService customer.CustomerService
	AppService      app.Service
	Logger          *slog.Logger
}

func (c Config) Validate() error {
	if c.Adapter == nil {
		return errors.New("adapter cannot be null")
	}

	if c.CustomerService == nil {
		return errors.New("customer service cannot be null")
	}

	if c.AppService == nil {
		return errors.New("app service cannot be null")
	}

	if c.Logger == nil {
		return errors.New("logger cannot be null")
	}

	return nil
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		adapter:         config.Adapter,
		customerService: config.CustomerService,
		appService:      config.AppService,
		logger:          config.Logger,
	}, nil
}

// TransactingRepoForGatheringInvoiceManipulation is a helper function that wraps the given function in a transaction and ensures that
// an update lock is held on the customer record. This is useful when you need to manipulate the gathering invoices, as we cannot lock an
// invoice, that doesn't exist yet.
func TransactingRepoForGatheringInvoiceManipulation[T any](ctx context.Context, adapter billing.Adapter, customer customerentity.CustomerID, fn func(ctx context.Context, txAdapter billing.Adapter) (T, error)) (T, error) {
	err := adapter.UpsertCustomerOverrideIgnoringTrns(ctx, customer)
	if err != nil {
		var empty T
		return empty, fmt.Errorf("upserting customer override: %w", err)
	}

	return entutils.TransactingRepo(ctx, adapter, func(ctx context.Context, txAdapter billing.Adapter) (T, error) {
		if err := txAdapter.LockCustomerForUpdate(ctx, customer); err != nil {
			var empty T
			return empty, fmt.Errorf("locking customer for update: %w", err)
		}

		return fn(ctx, txAdapter)
	})
}
