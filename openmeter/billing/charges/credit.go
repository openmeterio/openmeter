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

	LedgerTransaction LedgerTransactionGroupReference `json:"ledgerTransaction"`

	Amount alpacadecimal.Decimal `json:"amount"`
}

func (i CreditRealizationCreateInput) Validate() error {
	var errs []error

	if err := i.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if !i.Amount.IsPositive() {
		errs = append(errs, fmt.Errorf("amount must be positive"))
	}

	if err := i.LedgerTransaction.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("ledger transaction: %w", err))
	}

	return errors.Join(errs...)
}

type CreditRealization struct {
	models.NamespacedID
	models.ManagedModel
	CreditRealizationCreateInput

	// LineID is the standard invoice line ID that the credit was allocated to.
	// If nil, the credit is not allocated to any invoice line (e.g. line is still in gathering,
	// credit_only mode without invoicing, etc.)
	LineID *string `json:"lineID"`
}

func (r CreditRealization) Validate() error {
	var errs []error

	if err := r.CreditRealizationCreateInput.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("credit realization input: %w", err))
	}

	if r.LineID != nil && *r.LineID == "" {
		errs = append(errs, fmt.Errorf("line ID must be non-empty"))
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
