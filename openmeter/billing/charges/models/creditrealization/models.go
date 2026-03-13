package creditrealization

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type CreateInput struct {
	Annotations   models.Annotations    `json:"annotations"`
	ServicePeriod timeutil.ClosedPeriod `json:"servicePeriod"`

	LedgerTransaction ledgertransaction.GroupReference `json:"ledgerTransaction"`

	Amount alpacadecimal.Decimal `json:"amount"`

	// LineID is the standard invoice line ID that the credit was allocated to.
	// If nil, the credit is not allocated to any invoice line (e.g. line is still in gathering,
	// credit_only mode without invoicing, etc.)
	LineID *string `json:"lineID"`
}

func (i CreateInput) Validate() error {
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

	if i.LineID != nil && *i.LineID == "" {
		errs = append(errs, fmt.Errorf("line ID must be non-empty"))
	}

	return errors.Join(errs...)
}

type CreateInputs []CreateInput

func (i CreateInputs) Validate() error {
	var errs []error

	for idx, input := range i {
		if err := input.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("credit realization input[%d]: %w", idx, err))
		}
	}

	return errors.Join(errs...)
}

type Realization struct {
	models.NamespacedID
	models.ManagedModel
	CreateInput
}

func (r Realization) Validate() error {
	var errs []error

	if err := r.CreateInput.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("credit realization input: %w", err))
	}

	return errors.Join(errs...)
}

type Realizations []Realization

func (r Realizations) Validate() error {
	var errs []error

	for idx, realization := range r {
		if err := realization.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("credit realization[%d]: %w", idx, err))
		}
	}

	return errors.Join(errs...)
}
