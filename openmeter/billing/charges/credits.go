package charges

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type CreditRealizationCreateInput struct {
	Annotations   models.Annotations    `json:"annotations"`
	ServicePeriod timeutil.ClosedPeriod `json:"servicePeriod"`

	// TODO: let's add ledger transaction id(s) here

	Amount alpacadecimal.Decimal `json:"amount"`
}

func (i CreditRealizationCreateInput) Validate() error {
	var errs []error

	if err := i.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if i.Amount.IsNegative() {
		errs = append(errs, fmt.Errorf("amount must be positive"))
	}

	return errors.Join(errs...)
}

type CreditRealization struct {
	models.NamespacedID
	models.ManagedModel
	CreditRealizationCreateInput

	// AllocatedToStandardInvoiceRealizationID is the standard invoice realization ID that the credit was allocated to.
	// If nil, the credit is not allocated to any invoice line (e.g. line is still in gathering,
	// credit_only mode without invoicing, etc.)
	AllocatedToStandardInvoiceRealizationID *string `json:"allocatedToStandardInvoiceRealizationID"`
}

func (r CreditRealization) Validate() error {
	var errs []error

	if err := r.CreditRealizationCreateInput.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("credit realization input: %w", err))
	}

	if r.AllocatedToStandardInvoiceRealizationID != nil && *r.AllocatedToStandardInvoiceRealizationID == "" {
		errs = append(errs, fmt.Errorf("allocated to standard invoice realization ID must be set"))
	}

	return errors.Join(errs...)
}

type CreditRealizations []CreditRealization

func (r CreditRealizations) Validate() error {
	var errs []error

	for idx, realization := range r {
		if err := realization.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("credit realization[%d]: %w", idx, err))
		}
	}

	return errors.Join(errs...)
}

func (r CreditRealizations) RealizedUnsettledAmount() alpacadecimal.Decimal {
	sum := alpacadecimal.Zero
	for _, realization := range r {
		sum = sum.Add(realization.Amount)
	}
	return sum
}

func (r CreditRealizations) LastRealizedPeriod() *timeutil.ClosedPeriod {
	// TODO: we might want to filter for period realizations only
	if len(r) == 0 {
		return nil
	}

	lastRealizedPeriod := r[0].ServicePeriod
	for _, realization := range r {
		if realization.ServicePeriod.To.After(lastRealizedPeriod.To) {
			lastRealizedPeriod = realization.ServicePeriod
		}
	}

	return &lastRealizedPeriod
}

type ReversedCreditRealization struct {
	RealizationID models.NamespacedID `json:"realizationID"`

	TransactionGroupReferences []LedgerTransactionGroupReference `json:"transactionGroupReference"`
}

func (r ReversedCreditRealization) Validate() error {
	var errs []error

	if err := r.RealizationID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("realization ID: %w", err))
	}

	for idx, transactionGroupReference := range r.TransactionGroupReferences {
		if err := transactionGroupReference.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("transaction group reference[%d]: %w", idx, err))
		}
	}

	return errors.Join(errs...)
}
