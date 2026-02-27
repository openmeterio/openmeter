package charges

import (
	"errors"
	"fmt"
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
