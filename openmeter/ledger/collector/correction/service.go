package correction

import (
	"errors"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/advance"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Config struct {
	Logger             *slog.Logger
	Ledger             ledger.Ledger
	Advance            advance.Service
	Dependencies       transactions.ResolverDependencies
	Breakage           breakage.Service
	TransactionManager transaction.Creator
}

func (c Config) Validate() error {
	var errs []error

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if c.Ledger == nil {
		errs = append(errs, errors.New("ledger is required"))
	}

	if c.Advance == nil {
		errs = append(errs, errors.New("advance service is required"))
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

	if c.Breakage == nil {
		errs = append(errs, errors.New("breakage service is required"))
	}

	if c.TransactionManager == nil {
		errs = append(errs, errors.New("transaction manager is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type Corrector struct {
	ledger             ledger.Ledger
	advance            advance.Service
	deps               transactions.ResolverDependencies
	breakage           breakage.Service
	transactionManager transaction.Creator
}

func New(config Config) (*Corrector, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Corrector{
		ledger:             config.Ledger,
		advance:            config.Advance,
		deps:               config.Dependencies,
		breakage:           config.Breakage,
		transactionManager: config.TransactionManager,
	}, nil
}
