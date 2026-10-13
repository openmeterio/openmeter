package fbo

import (
	"context"
	"errors"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/advance"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Service interface {
	Consume(ctx context.Context, input ConsumeInput) ([]ledger.TransactionInput, error)
	ListSources(ctx context.Context, input SourceQuery) ([]AvailableSource, error)
}

type Config struct {
	Logger        *slog.Logger
	Dependencies  transactions.ResolverDependencies
	Advance       advance.Service
	Breakage      breakage.Service
	AccountLocker ledger.AccountLocker
}

var _ models.Validator = Config{}

func (c Config) Validate() error {
	var errs []error
	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if c.Dependencies.AccountService == nil {
		errs = append(errs, errors.New("account service is required"))
	}

	if c.Dependencies.AccountCatalog == nil {
		errs = append(errs, errors.New("account catalog is required"))
	}

	if c.Dependencies.BalanceQuerier == nil {
		errs = append(errs, errors.New("balance querier is required"))
	}

	if c.Advance == nil {
		errs = append(errs, errors.New("advance service is required"))
	}

	if c.Breakage == nil {
		errs = append(errs, errors.New("breakage service is required"))
	}

	if c.AccountLocker == nil {
		errs = append(errs, errors.New("account locker is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type service struct {
	dependencies  transactions.ResolverDependencies
	advance       advance.Service
	breakage      breakage.Service
	accountLocker ledger.AccountLocker
}

var _ Service = (*service)(nil)

func NewService(config Config) (Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		dependencies:  config.Dependencies,
		advance:       config.Advance,
		breakage:      config.Breakage,
		accountLocker: config.AccountLocker,
	}, nil
}
