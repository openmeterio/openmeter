package charges

import (
	"errors"
	"fmt"
	"time"
)

// LedgerTransactionGroupReference is a reference to a ledger transaction group.
// It is used to track payment settlement transactions.
type LedgerTransactionGroupReference struct {
	TransactionGroupID string `json:"transactionGroupID"`
}

func (r LedgerTransactionGroupReference) Validate() error {
	var errs []error

	if r.TransactionGroupID == "" {
		errs = append(errs, fmt.Errorf("transaction group ID is required"))
	}

	return errors.Join(errs...)
}

type TimedLedgerTransactionGroupReference struct {
	LedgerTransactionGroupReference
	Time time.Time `json:"time"`
}

func (r TimedLedgerTransactionGroupReference) Validate() error {
	var errs []error

	if err := r.LedgerTransactionGroupReference.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("ledger transaction group reference: %w", err))
	}

	if r.Time.IsZero() {
		errs = append(errs, fmt.Errorf("time is required"))
	}

	return errors.Join(errs...)
}
