package service

import (
	"errors"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/advance"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Config struct {
	Logger          *slog.Logger
	Ledger          ledger.Ledger
	BalanceQuerier  ledger.BalanceQuerier
	AccountResolver ledger.AccountResolver
	AccountCatalog  ledger.AccountCatalog
	Breakage        breakage.Service
}

func (c Config) Validate() error {
	var errs []error

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if c.Ledger == nil {
		errs = append(errs, errors.New("ledger is required"))
	}

	if c.BalanceQuerier == nil {
		errs = append(errs, errors.New("balance querier is required"))
	}

	if c.AccountResolver == nil {
		errs = append(errs, errors.New("account resolver is required"))
	}

	if c.AccountCatalog == nil {
		errs = append(errs, errors.New("account catalog is required"))
	}

	if c.Breakage == nil {
		errs = append(errs, errors.New("breakage service is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func New(config Config) (advance.Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		logger:          config.Logger,
		ledger:          config.Ledger,
		balanceQuerier:  config.BalanceQuerier,
		accountResolver: config.AccountResolver,
		accountCatalog:  config.AccountCatalog,
		breakage:        config.Breakage,
	}, nil
}

type service struct {
	logger          *slog.Logger
	ledger          ledger.Ledger
	balanceQuerier  ledger.BalanceQuerier
	accountResolver ledger.AccountResolver
	accountCatalog  ledger.AccountCatalog
	breakage        breakage.Service
}

func (s *service) resolverDependencies() transactions.ResolverDependencies {
	return transactions.ResolverDependencies{
		AccountService: s.accountResolver,
		AccountCatalog: s.accountCatalog,
		BalanceQuerier: s.balanceQuerier,
	}
}

var _ advance.Service = (*service)(nil)
