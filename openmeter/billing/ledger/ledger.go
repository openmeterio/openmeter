package ledger

import (
	"context"
	"errors"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type LedgerID models.NamespacedID

func (l LedgerID) Validate() error {
	var errs []error

	if l.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if l.ID == "" {
		errs = append(errs, errors.New("id is required"))
	}

	return errors.Join(errs...)
}

type LedgerRef struct {
	Customer customer.CustomerID
	Currency currencyx.Code
}

func (l LedgerRef) Validate() error {
	var errs []error

	if err := l.Customer.Validate(); err != nil {
		errs = append(errs, errors.New("customer is required"))
	}

	if err := l.Currency.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

type Ledger struct {
	models.NamespacedModel
	models.ManagedModel

	ID         string
	CustomerID string
	Currency   currencyx.Code
}

func (l Ledger) GetID() LedgerID {
	return LedgerID{
		Namespace: l.Namespace,
		ID:        l.ID,
	}
}

func (l *Ledger) Validate() error {
	var errs []error

	if err := l.NamespacedModel.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := l.ManagedModel.Validate(); err != nil {
		errs = append(errs, err)
	}

	if l.ID == "" {
		errs = append(errs, errors.New("id is required"))
	}

	if l.CustomerID == "" {
		errs = append(errs, errors.New("customer id is required"))
	}

	if l.Currency == "" {
		errs = append(errs, errors.New("currency is required"))
	}

	return errors.Join(errs...)
}

type WithLockedLedgerInput struct {
	LedgerRef
	Callback func(ctx context.Context, ledger LedgerMutationService) error
}

func (i WithLockedLedgerInput) Validate() error {
	var errs []error

	if err := i.LedgerRef.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.Callback == nil {
		errs = append(errs, errors.New("callback is required"))
	}

	return errors.Join(errs...)
}

type WithLockedLedgerAdapterInput struct {
	Customer customer.CustomerID
	Currency currencyx.Code

	Callback func(ctx context.Context, ledger Ledger) error
}

func (i WithLockedLedgerAdapterInput) Validate() error {
	var errs []error

	if err := i.Customer.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.Callback == nil {
		errs = append(errs, errors.New("callback is required"))
	}

	return errors.Join(errs...)
}

type GetBalanceInput = LedgerRef

type GetBalanceAdapterResult []SubledgerBalance

type GetBalanceResult struct {
	Balance alpacadecimal.Decimal

	SubledgerBalances []SubledgerBalance
}

type SubledgerBalance struct {
	Subledger Subledger

	Balance alpacadecimal.Decimal
}
