package creditrealization

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/google/uuid"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	"github.com/samber/lo"
)

type CreateAllocationInput struct {
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
}

func (i CreateAllocationInput) Validate() error {
	var errs []error

	if i.ID != "" {
		if err := uuid.Validate(i.ID); err != nil {
			errs = append(errs, fmt.Errorf("id must be a valid UUID: %w", err))
		}
	}

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

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CreateAllocationInputs []CreateAllocationInput

func (i CreateAllocationInputs) Validate() error {
	var errs []error

	for idx, input := range i {
		if err := input.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("create allocation input[%d]: %w", idx, err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (i CreateAllocationInputs) AsCreateInputs() CreateInputs {
	return lo.Map(i, func(input CreateAllocationInput, _ int) CreateInput {
		return CreateInput{
			ID:                input.ID,
			Annotations:       input.Annotations,
			ServicePeriod:     input.ServicePeriod,
			LedgerTransaction: input.LedgerTransaction,
			Amount:            input.Amount,
			Type:              TypeAllocation,
			LineID:            input.LineID,
		}
	})
}

func (i CreateAllocationInputs) Sum() alpacadecimal.Decimal {
	sum := alpacadecimal.Zero
	for _, input := range i {
		sum = sum.Add(input.Amount)
	}
	return sum
}
