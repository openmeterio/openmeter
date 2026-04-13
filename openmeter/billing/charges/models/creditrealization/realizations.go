package creditrealization

import (
	"errors"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/lo/mutable"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type Realizations []Realization

func (r Realizations) Validate() error {
	var errs []error

	for idx, realization := range r {
		if err := realization.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("credit realization[%d]: %w", idx, err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (r Realizations) Sum() alpacadecimal.Decimal {
	sum := alpacadecimal.Zero
	for _, realization := range r {
		sum = sum.Add(realization.Amount)
	}
	return sum
}

func (r Realizations) AllocationsByID() map[string]Realization {
	return lo.KeyBy(
		lo.Filter(r, func(realization Realization, _ int) bool {
			return realization.Type == TypeAllocation
		}),
		func(realization Realization) string {
			return realization.ID
		},
	)
}

func (r Realizations) AsCreditsApplied() (billing.CreditsApplied, error) {
	allocationsWithCorrections, err := r.allocationsWithCorrections()
	if err != nil {
		return nil, err
	}

	creditsApplied := make(billing.CreditsApplied, 0, len(allocationsWithCorrections))
	for _, allocationWithCorrections := range allocationsWithCorrections {
		if !allocationWithCorrections.RemainingAmount.IsPositive() {
			continue
		}

		creditsApplied = append(creditsApplied, billing.CreditApplied{
			Amount:              allocationWithCorrections.RemainingAmount,
			CreditRealizationID: allocationWithCorrections.Allocation.ID,
		})
	}

	return creditsApplied, nil
}

var ErrInsufficientFunds = models.NewGenericValidationError(errors.New("insufficient funds"))

func (r Realizations) CreateCorrectionRequest(amount alpacadecimal.Decimal, currency currencyx.Calculator) (CorrectionRequest, error) {
	if amount.IsPositive() {
		return CorrectionRequest{}, models.NewGenericValidationError(errors.New("amount must not be positive"))
	}

	amount = currency.RoundToPrecision(amount)
	if amount.IsZero() {
		return nil, nil
	}

	allocationsWithCorrections, err := r.allocationsWithCorrections()
	if err != nil {
		return CorrectionRequest{}, err
	}

	mutable.Reverse(allocationsWithCorrections)

	out := make(CorrectionRequest, 0, len(allocationsWithCorrections))
	amountToCorrect := amount.Abs()
	for _, allocationWithCorrections := range allocationsWithCorrections {
		if allocationWithCorrections.RemainingAmount.IsZero() {
			continue
		}

		if allocationWithCorrections.RemainingAmount.GreaterThan(amountToCorrect) {
			out = append(out, CorrectionRequestItem{
				Allocation: allocationWithCorrections.Allocation,
				Amount:     amountToCorrect.Neg(),
			})

			amountToCorrect = alpacadecimal.Zero
			break
		}

		out = append(out, CorrectionRequestItem{
			Allocation: allocationWithCorrections.Allocation,
			Amount:     allocationWithCorrections.RemainingAmount.Neg(),
		})

		amountToCorrect = amountToCorrect.Sub(allocationWithCorrections.RemainingAmount)
	}

	if amountToCorrect.IsPositive() {
		return CorrectionRequest{}, ErrInsufficientFunds
	}

	return out, nil
}

func (r Realizations) Correct(amount alpacadecimal.Decimal, currency currencyx.Calculator, cb func(req CorrectionRequest) (CreateCorrectionInputs, error)) (CreateInputs, error) {
	req, err := r.CreateCorrectionRequest(amount, currency)
	if err != nil {
		return nil, err
	}

	if len(req) == 0 {
		return nil, nil
	}

	if err := req.ValidateWith(currency); err != nil {
		return nil, err
	}

	corrections, err := cb(req)
	if err != nil {
		return nil, err
	}
	corrections = corrections.NormalizeWith(currency)

	if err := corrections.ValidateWith(r, amount.Abs(), currency); err != nil {
		return nil, err
	}

	return corrections.AsCreateInputs(r)
}

func (r Realizations) CorrectAll(currency currencyx.Calculator, cb func(req CorrectionRequest) (CreateCorrectionInputs, error)) (CreateInputs, error) {
	total := r.Sum()
	if total.IsZero() {
		return nil, nil
	}

	return r.Correct(total.Neg(), currency, cb)
}

type allocationWithCorrections struct {
	Allocation      Realization
	Corrections     []Realization
	RemainingAmount alpacadecimal.Decimal
}

// allocationsWithCorrections returns the allocations with the corrections that can are applied to them,
// the return value is sorted by creation order (reverts should happen in reverse order).
func (r Realizations) allocationsWithCorrections() ([]allocationWithCorrections, error) {
	// let's collect the corrections by allocation ID

	corrections := lo.Filter(r, func(realization Realization, _ int) bool {
		return realization.Type == TypeCorrection
	})

	correctionsByAllocationID := lo.GroupBy(corrections, func(correction Realization) string {
		return lo.FromPtr(correction.CorrectsRealizationID)
	})
	if _, ok := correctionsByAllocationID[""]; ok {
		return nil, models.NewGenericValidationError(errors.New("correction for unknown allocation"))
	}

	// let's collect the allocations by allocation ID
	realizations := lo.Filter(r, func(realization Realization, _ int) bool {
		return realization.Type == TypeAllocation
	})

	// Let's sort the allocations by createdAt + sortHint
	slices.SortStableFunc(realizations, func(a, b Realization) int {
		cmpCreatedAt := a.CreatedAt.Compare(b.CreatedAt)
		if cmpCreatedAt != 0 {
			return cmpCreatedAt
		}

		return a.SortHint - b.SortHint
	})

	// let's assign the allocations to the corrections
	return slicesx.MapWithErr(realizations, func(allocation Realization) (allocationWithCorrections, error) {
		remainingAmount := allocation.Amount
		corrections, hasCorrections := correctionsByAllocationID[allocation.ID]
		if hasCorrections {
			remainingAmount = remainingAmount.Add(corrections.Sum())
		}

		if remainingAmount.IsNegative() {
			return allocationWithCorrections{}, models.NewGenericValidationError(fmt.Errorf("remaining amount is negative for allocation %s", allocation.ID))
		}

		return allocationWithCorrections{
			Allocation:      allocation,
			Corrections:     corrections,
			RemainingAmount: remainingAmount,
		}, nil
	})
}
