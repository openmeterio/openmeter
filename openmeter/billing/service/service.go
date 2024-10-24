package billingservice

import (
	"context"
	"errors"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
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

func Transaction[R any](ctx context.Context, creator billing.Adapter, cb func(ctx context.Context, tx billing.Adapter) (R, error)) (R, error) {
	return transaction.Run(ctx, creator, func(ctx context.Context) (R, error) {
		return entutils.TransactingRepo[R, billing.Adapter](ctx, creator, cb)
	})
}

func TransactionWithNoValue(ctx context.Context, creator billing.Adapter, cb func(ctx context.Context, tx billing.Adapter) error) error {
	return transaction.RunWithNoValue(ctx, creator, func(ctx context.Context) error {
		_, err := entutils.TransactingRepo[interface{}, billing.Adapter](ctx, creator, func(ctx context.Context, rep billing.Adapter) (interface{}, error) {
			return nil, cb(ctx, rep)
		})
		return err
	})
}
