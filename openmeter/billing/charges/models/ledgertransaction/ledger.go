package ledgertransaction

import (
	"errors"
	"fmt"
	"time"
)

// LedgerTransactionGroupReference is a reference to a ledger transaction group.
// It is used to track payment settlement transactions.
type GroupReference struct {
	TransactionGroupID string `json:"transactionGroupID"`
}

func (r GroupReference) Validate() error {
	var errs []error

	if r.TransactionGroupID == "" {
		errs = append(errs, fmt.Errorf("transaction group ID is required"))
	}

	return errors.Join(errs...)
}

func (r *GroupReference) GetIDOrNull() *string {
	if r == nil || r.TransactionGroupID == "" {
		return nil
	}

	return &r.TransactionGroupID
}

type TimedGroupReference struct {
	GroupReference
	Time time.Time `json:"time"`
}

func (r TimedGroupReference) Validate() error {
	var errs []error

	if err := r.GroupReference.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("ledger transaction group reference: %w", err))
	}

	if r.Time.IsZero() {
		errs = append(errs, fmt.Errorf("time is required"))
	}

	return errors.Join(errs...)
}

func (r *TimedGroupReference) GetIDOrNull() *string {
	if r == nil || r.TransactionGroupID == "" {
		return nil
	}

	return r.GroupReference.GetIDOrNull()
}

func (r *TimedGroupReference) GetTimeOrNull() *time.Time {
	if r == nil {
		return nil
	}

	return &r.Time
}
