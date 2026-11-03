package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CreditBalanceCalculator interface {
	GetBalance(context.Context, customerbalance.GetBalanceServiceInput) (customerbalance.Balance, error)
}

type Config struct {
	Adapter    rte.Adapter
	Calculator CreditBalanceCalculator
	Logger     *slog.Logger
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter is required"))
	}

	if c.Calculator == nil {
		errs = append(errs, errors.New("calculator is required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func New(config Config) (rte.Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{adapter: config.Adapter, calculator: config.Calculator, logger: config.Logger}, nil
}

type service struct {
	adapter    rte.Adapter
	calculator CreditBalanceCalculator
	logger     *slog.Logger
}

func (s *service) CalculateCreditBalance(ctx context.Context, key rte.CreditBalanceKey) (rte.CreditBalance, error) {
	if err := key.Validate(); err != nil {
		return rte.CreditBalance{}, err
	}

	calculatedAt := clock.Now()
	// Leave AsOf unset: historical queries intentionally omit live usage.
	balance, err := s.calculator.GetBalance(ctx, customerbalance.GetBalanceServiceInput{
		CustomerID:    key.CustomerID,
		Currency:      key.Currency,
		FeatureFilter: key.FeatureFilter,
	})
	if err != nil {
		return rte.CreditBalance{}, fmt.Errorf("calculate credit balance: %w", err)
	}

	return s.adapter.StoreCreditBalance(ctx, rte.StoreCreditBalanceInput{
		Key: key,
		Balance: rte.CreditBalance{
			CalculatedAt: calculatedAt,
			Booked:       balance.Booked(),
			Spendable:    balance.Spendable(),
			Live:         balance.Live(),
			Pending:      balance.Pending(),
		},
	})
}

func (s *service) GetCreditBalance(ctx context.Context, key rte.CreditBalanceKey) (rte.CreditBalance, error) {
	if err := key.Validate(); err != nil {
		return rte.CreditBalance{}, err
	}

	return s.adapter.GetCreditBalance(ctx, key)
}
