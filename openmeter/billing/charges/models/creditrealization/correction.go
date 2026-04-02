package creditrealization

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type CorrectionRequest []CorrectionRequestItem

func (c CorrectionRequest) ValidateWith(currency currencyx.Calculator) error {
	var errs []error

	for idx, item := range c {
		if err := item.ValidateWith(currency); err != nil {
			errs = append(errs, fmt.Errorf("correction request item[%d]: %w", idx, err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CorrectionRequestItem struct {
	Allocation Realization `json:"allocation"`
	// Amount is the amount of the correction request.
	// It is non-positive and rounded to the smallest denomination.
	Amount alpacadecimal.Decimal `json:"amount"`
}

func (i CorrectionRequestItem) ValidateWith(currency currencyx.Calculator) error {
	var errs []error

	if err := i.Allocation.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("allocation: %w", err))
	}

	if i.Amount.IsPositive() {
		errs = append(errs, fmt.Errorf("amount must not be positive"))
	}

	if !currency.IsRoundedToPrecision(i.Amount) {
		errs = append(errs, fmt.Errorf("amount must be a multiple of the smallest denomination"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (i CorrectionRequestItem) NormalizeWith(currency currencyx.Calculator) CorrectionRequestItem {
	i.Amount = currency.RoundToPrecision(i.Amount)
	return i
}

type CreateCorrectionInput struct {
	// ID is the ID of the correction, if empty a new ID will be generated.
	ID          string             `json:"id"`
	Annotations models.Annotations `json:"annotations"`

	LedgerTransaction ledgertransaction.GroupReference `json:"ledgerTransaction"`

	// Amount is the amount of the correction.
	// Expectations:
	// - It must be non-positive
	// - It must be rounded to the smallest denomination
	Amount alpacadecimal.Decimal `json:"amount"`

	// CorrectsRealizationID is the ID of the realization that this correction is correcting.
	CorrectsRealizationID string `json:"correctsRealizationID"`
}

func (i CreateCorrectionInput) NormalizeWith(currency currencyx.Calculator) CreateCorrectionInput {
	i.Amount = currency.RoundToPrecision(i.Amount)
	return i
}

func (i CreateCorrectionInput) ValidateWith(currency currencyx.Calculator) error {
	var errs []error

	if i.Amount.IsPositive() {
		errs = append(errs, fmt.Errorf("amount must not be positive"))
	}

	if !currency.IsRoundedToPrecision(i.Amount) {
		errs = append(errs, fmt.Errorf("amount must be rounded to currency precision"))
	}

	if i.CorrectsRealizationID == "" {
		errs = append(errs, fmt.Errorf("corrects realization id is required"))
	}

	if err := i.LedgerTransaction.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("ledger transaction: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CreateCorrectionInputs []CreateCorrectionInput

func (i CreateCorrectionInputs) NormalizeWith(currency currencyx.Calculator) CreateCorrectionInputs {
	return lo.Map(i, func(input CreateCorrectionInput, _ int) CreateCorrectionInput {
		return input.NormalizeWith(currency)
	})
}

func (i CreateCorrectionInputs) ValidateWith(existingRealizations Realizations, totalAmountToCorrect alpacadecimal.Decimal, currency currencyx.Calculator) error {
	var errs []error

	if totalAmountToCorrect.IsNegative() {
		errs = append(errs, fmt.Errorf("total amount to correct must not be negative"))
	}

	if !currency.IsRoundedToPrecision(totalAmountToCorrect) {
		errs = append(errs, fmt.Errorf("total amount to correct must be rounded to currency precision"))
	}

	for idx, input := range i {
		if err := input.ValidateWith(currency); err != nil {
			errs = append(errs, fmt.Errorf("correction input[%d]: %w", idx, err))
		}
	}

	realizationsWithRemainingAmount, err := existingRealizations.allocationsWithCorrections()
	if err != nil {
		errs = append(errs, fmt.Errorf("getting allocations with remaining amount: %w", err))
		return models.NewNillableGenericValidationError(errors.Join(errs...))
	}

	realizationsWithRemainingAmountByID := lo.KeyBy(realizationsWithRemainingAmount, func(allocationWithCorrections allocationWithCorrections) string {
		return allocationWithCorrections.Allocation.ID
	})

	correctionTotal := alpacadecimal.NewFromFloat(0)
	for _, input := range i {
		correctionTotal = correctionTotal.Add(input.Amount.Abs())
	}

	if !currency.RoundToPrecision(correctionTotal).Equal(totalAmountToCorrect) {
		errs = append(errs, fmt.Errorf("corrections total %s does not match the requested amount %s", correctionTotal, totalAmountToCorrect))
	}

	for idx, input := range i {
		correctsRealization, ok := realizationsWithRemainingAmountByID[input.CorrectsRealizationID]

		if !ok {
			errs = append(errs, fmt.Errorf("correction input[%d]: corrects realization id %s not found or is not a correction", idx, input.CorrectsRealizationID))
			break // let's stop validating we are depending on a corrupt state already
		}

		if input.Amount.Abs().GreaterThan(correctsRealization.RemainingAmount) {
			errs = append(errs, fmt.Errorf("correction input[%d]: amount to correct is greater than the remaining amount for allocation %s", idx, input.CorrectsRealizationID))
			break // let's stop validating we are depending on a corrupt state already
		}

		correctsRealization.RemainingAmount = correctsRealization.RemainingAmount.Add(input.Amount)
		realizationsWithRemainingAmountByID[input.CorrectsRealizationID] = correctsRealization
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (i CreateCorrectionInputs) AsCreateInputs(existingRealizations Realizations) ([]CreateInput, error) {
	existingAllocations := existingRealizations.AllocationsByID()

	return slicesx.MapWithErr(i, func(input CreateCorrectionInput) (CreateInput, error) {
		allocation, ok := existingAllocations[input.CorrectsRealizationID]
		if !ok {
			return CreateInput{}, fmt.Errorf("allocation %s not found", input.CorrectsRealizationID)
		}

		return CreateInput{
			ID:                    input.ID,
			Annotations:           input.Annotations,
			ServicePeriod:         allocation.ServicePeriod,
			LedgerTransaction:     input.LedgerTransaction,
			Amount:                input.Amount,
			Type:                  TypeCorrection,
			CorrectsRealizationID: lo.ToPtr(input.CorrectsRealizationID),
		}, nil
	})
}
