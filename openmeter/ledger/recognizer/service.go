package recognizer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type Service interface {
	RecognizeEarnings(ctx context.Context, in RecognizeEarningsInput) (RecognizeEarningsResult, error)
}

type Config struct {
	Ledger             ledger.Ledger
	Dependencies       transactions.ResolverDependencies
	Lineage            lineage.Service
	TransactionManager transaction.Creator
}

func (c Config) Validate() error {
	var errs []error

	if c.Ledger == nil {
		errs = append(errs, errors.New("ledger is required"))
	}
	if c.Dependencies.AccountService == nil {
		errs = append(errs, errors.New("account service is required"))
	}
	if c.Dependencies.SubAccountService == nil {
		errs = append(errs, errors.New("sub-account service is required"))
	}
	if c.Lineage == nil {
		errs = append(errs, errors.New("lineage service is required"))
	}
	if c.TransactionManager == nil {
		errs = append(errs, errors.New("transaction manager is required"))
	}

	return errors.Join(errs...)
}

func NewService(config Config) (Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		ledger:             config.Ledger,
		deps:               config.Dependencies,
		lnge:               config.Lineage,
		transactionManager: config.TransactionManager,
	}, nil
}

type service struct {
	ledger             ledger.Ledger
	deps               transactions.ResolverDependencies
	lnge               lineage.Service
	transactionManager transaction.Creator
}

// RecognizeEarningsInput is the input for RecognizeEarnings.
type RecognizeEarningsInput struct {
	CustomerID customer.CustomerID
	At         time.Time
	Currency   currencyx.Code
	Reason     string
}

func (i RecognizeEarningsInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer id: %w", err))
	}
	if i.At.IsZero() {
		errs = append(errs, errors.New("at is required"))
	}
	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	return errors.Join(errs...)
}

// RecognizeEarningsResult contains the result of a recognition run.
type RecognizeEarningsResult struct {
	RecognizedAmount alpacadecimal.Decimal
	LedgerGroupID    string
}
