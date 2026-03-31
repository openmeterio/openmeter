package creditrealization

import (
	"errors"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/google/uuid"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type AdapterCreateInput struct {
	// ID is the ID of the credit realization, if empty a new ID will be generated.
	// If set, the ID must be a valid UUID.
	ID            string                `json:"id"`
	Annotations   models.Annotations    `json:"annotations"`
	ServicePeriod timeutil.ClosedPeriod `json:"servicePeriod"`

	LedgerTransaction ledgertransaction.GroupReference `json:"ledgerTransaction"`

	Amount alpacadecimal.Decimal `json:"amount"`

	// LineID is the standard invoice line ID that the credit was allocated to.
	// If nil, the credit is not allocated to any invoice line (e.g. line is still in gathering,
	// credit_only mode without invoicing, etc.)
	LineID *string `json:"lineID"`

	Type                  Type    `json:"type"`
	CorrectsRealizationID *string `json:"correctsRealizationID"`
}

type Type string

const (
	TypeAllocation Type = "allocation"
	TypeCorrection Type = "correction"
)

func (t Type) Values() []string {
	return []string{
		string(TypeAllocation),
		string(TypeCorrection),
	}
}

func (t Type) Validate() error {
	if !slices.Contains(t.Values(), string(t)) {
		return fmt.Errorf("invalid credit realization type: %s", t)
	}
	return nil
}

func (i AdapterCreateInput) Validate() error {
	var errs []error

	if i.ID != "" {
		if err := uuid.Validate(i.ID); err != nil {
			errs = append(errs, fmt.Errorf("id must be a valid UUID: %w", err))
		}
	}

	if err := i.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if err := i.Type.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("type: %w", err))
	}

	switch i.Type {
	case TypeAllocation:
		if i.LineID == nil {
			errs = append(errs, fmt.Errorf("line ID is required"))
		}

		if !i.Amount.IsPositive() {
			errs = append(errs, fmt.Errorf("amount must be positive"))
		}
	case TypeCorrection:
		if i.CorrectsRealizationID == nil {
			errs = append(errs, fmt.Errorf("corrects realization ID is required"))
		}

		if !i.Amount.IsNegative() {
			errs = append(errs, fmt.Errorf("amount must be negative"))
		}
	}

	if err := i.LedgerTransaction.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("ledger transaction: %w", err))
	}

	if i.LineID != nil && *i.LineID == "" {
		errs = append(errs, fmt.Errorf("line ID must be non-empty"))
	}

	return errors.Join(errs...)
}

type AdapterCreateInputs []AdapterCreateInput

func (i AdapterCreateInputs) Validate() error {
	var errs []error

	for idx, input := range i {
		if err := input.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("credit realization input[%d]: %w", idx, err))
		}
	}

	return errors.Join(errs...)
}

func (i AdapterCreateInputs) Sum() alpacadecimal.Decimal {
	sum := alpacadecimal.Zero
	for _, input := range i {
		sum = sum.Add(input.Amount)
	}
	return sum
}

type Realization struct {
	models.NamespacedModel
	models.ManagedModel
	AdapterCreateInput

	// SortHint is the hint for the order of the credit realizations created in the same batch.
	// Given collection is in priority order, reverting any transaction group should happen in reverse order.
	SortHint int `json:"sortHint"`
}

func (r Realization) Validate() error {
	var errs []error

	if err := r.AdapterCreateInput.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("credit realization input: %w", err))
	}

	return errors.Join(errs...)
}
