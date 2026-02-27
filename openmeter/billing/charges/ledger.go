package charges

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

// LedgerTransactionGroupReference is a reference to a ledger transaction group.
// It is used to track payment settlement transactions.
type LedgerTransactionGroupReference struct {
	TransactionGroupID models.NamespacedID `json:"transactionGroupID"`
}

func (r LedgerTransactionGroupReference) Validate() error {
	var errs []error

	if err := r.TransactionGroupID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("transaction group ID: %w", err))
	}

	return errors.Join(errs...)
}
