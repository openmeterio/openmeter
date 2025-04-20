package ledger

import (
	"errors"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Transaction struct {
	models.ManagedResource

	SubledgerID string
	LedgerID    string
	// Amount is the amount of the transaction. Positive value means that the
	// customer received funds, negative value means that the customer lost funds.
	Amount   alpacadecimal.Decimal
	Owner    *OwnerReference
	Metadata models.Metadata
}

func (t *Transaction) Validate() error {
	var errs []error

	if err := t.ManagedResource.Validate(); err != nil {
		errs = append(errs, err)
	}

	if t.SubledgerID == "" {
		errs = append(errs, errors.New("subledger id is required"))
	}

	if t.LedgerID == "" {
		errs = append(errs, errors.New("ledger id is required"))
	}

	if t.Amount.IsZero() {
		errs = append(errs, errors.New("amount must be greater than 0"))
	}

	if t.Owner != nil {
		if err := t.Owner.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

type TransactionMeta struct {
	Metadata    models.Metadata
	Name        string
	Description *string
	Owner       *OwnerReference
}

func (m TransactionMeta) Validate() error {
	var errs []error

	if m.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	if m.Owner != nil {
		if err := m.Owner.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

type CreateTransactionInput struct {
	Subledger Subledger

	Amount alpacadecimal.Decimal

	TransactionMeta
}

func (i CreateTransactionInput) Validate() error {
	var errs []error

	if err := i.Subledger.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := i.TransactionMeta.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.Amount.IsZero() {
		errs = append(errs, errors.New("amount must not be zero"))
	}

	return errors.Join(errs...)
}

type WithdrawInput struct {
	Amount alpacadecimal.Decimal

	// Metadata for the created transactions
	TransactionMeta
}

func (i WithdrawInput) Validate() error {
	var errs []error

	if err := i.TransactionMeta.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.Amount.IsZero() {
		errs = append(errs, errors.New("amount must not be zero"))
	}

	return errors.Join(errs...)
}

type WithdrawalResults struct {
	Transactions   []Transaction
	TotalWithdrawn alpacadecimal.Decimal
}
