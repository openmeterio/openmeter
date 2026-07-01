package currencyx

import (
	"cmp"
	"errors"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"
)

// WeightedAllocationItem defines one key that can receive a proportional
// allocation from a currency amount. Weight is dimensionless; it does not need
// to be a currency amount.
type WeightedAllocationItem[T any] struct {
	Key    T
	Weight alpacadecimal.Decimal
}

// WeightedAllocation is the allocated currency amount for one key.
type WeightedAllocation[T any] struct {
	Key    T
	Amount alpacadecimal.Decimal
}

// AmountAllocationItem defines one currency amount bucket that can receive a
// proportional allocation. The amount is both the allocation weight and the
// maximum amount that can be allocated to the key.
type AmountAllocationItem[T any] struct {
	Key    T
	Amount alpacadecimal.Decimal
}

// AmountAllocation is the allocated currency amount for one key.
type AmountAllocation[T any] struct {
	Key    T
	Amount alpacadecimal.Decimal
}

// WeightedAllocationInput defines a proportional currency allocation.
type WeightedAllocationInput[T any] struct {
	Amount alpacadecimal.Decimal
	Items  []WeightedAllocationItem[T]

	// CompareKey is used as a deterministic tie-breaker when two items have
	// the same fractional remainder. If nil, the original item order is used.
	CompareKey func(left, right T) int
}

// AmountAllocationInput defines a proportional allocation across currency
// amount buckets.
type AmountAllocationInput[T any] struct {
	Amount alpacadecimal.Decimal
	Items  []AmountAllocationItem[T]

	// CompareKey is used as a deterministic tie-breaker when two buckets have
	// the same fractional remainder. If nil, the original item order is used.
	CompareKey func(left, right T) int
}

// AllocateByWeight allocates a currency amount across keys using their
// weights and the largest remainder quota method at the currency precision.
func AllocateByWeight[T any](calculator Calculator, input WeightedAllocationInput[T]) ([]WeightedAllocation[T], error) {
	if err := validateWeightedAllocationInput(calculator, input); err != nil {
		return nil, err
	}

	if input.Amount.IsZero() {
		return nil, nil
	}

	totalWeight := alpacadecimal.Zero
	for _, item := range input.Items {
		totalWeight = totalWeight.Add(item.Weight)
	}

	type allocationCandidate struct {
		index     int
		key       T
		amount    alpacadecimal.Decimal
		remainder alpacadecimal.Decimal
	}

	candidates := make([]allocationCandidate, 0, len(input.Items))
	allocated := alpacadecimal.Zero
	for i, item := range input.Items {
		share := input.Amount.Mul(item.Weight).Div(totalWeight)
		amount := calculator.RoundDown(share)

		candidates = append(candidates, allocationCandidate{
			index:     i,
			key:       item.Key,
			amount:    amount,
			remainder: share.Sub(amount),
		})
		allocated = allocated.Add(amount)
	}

	slices.SortStableFunc(candidates, func(left, right allocationCandidate) int {
		if remainderCmp := right.remainder.Cmp(left.remainder); remainderCmp != 0 {
			return remainderCmp
		}

		if input.CompareKey != nil {
			if keyCmp := input.CompareKey(left.key, right.key); keyCmp != 0 {
				return keyCmp
			}
		}

		return cmp.Compare(left.index, right.index)
	})

	unit := currencyUnit(calculator)
	remaining := input.Amount.Sub(allocated)
	for i := range candidates {
		if remaining.LessThan(unit) {
			break
		}

		candidates[i].amount = candidates[i].amount.Add(unit)
		remaining = remaining.Sub(unit)
	}

	slices.SortFunc(candidates, func(left, right allocationCandidate) int {
		return cmp.Compare(left.index, right.index)
	})

	allocations := make([]WeightedAllocation[T], 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.amount.IsZero() {
			continue
		}

		allocations = append(allocations, WeightedAllocation[T]{
			Key:    candidate.key,
			Amount: candidate.amount,
		})
	}

	return allocations, nil
}

// AllocateByAmount allocates a currency amount across currency amount buckets
// using the largest remainder quota method. Each item amount is both its
// proportional weight and its allocation cap.
func AllocateByAmount[T any](calculator Calculator, input AmountAllocationInput[T]) ([]AmountAllocation[T], error) {
	if err := validateAmountAllocationInput(calculator, input); err != nil {
		return nil, err
	}

	if input.Amount.IsZero() {
		return nil, nil
	}

	totalAmount := alpacadecimal.Zero
	for _, item := range input.Items {
		totalAmount = totalAmount.Add(item.Amount)
	}

	type allocationCandidate struct {
		index     int
		key       T
		amount    alpacadecimal.Decimal
		allocated alpacadecimal.Decimal
		remainder alpacadecimal.Decimal
	}

	candidates := make([]allocationCandidate, 0, len(input.Items))
	allocated := alpacadecimal.Zero
	for i, item := range input.Items {
		share := input.Amount.Mul(item.Amount).Div(totalAmount)
		floor := calculator.RoundDown(share)

		candidates = append(candidates, allocationCandidate{
			index:     i,
			key:       item.Key,
			amount:    item.Amount,
			allocated: floor,
			remainder: share.Sub(floor),
		})
		allocated = allocated.Add(floor)
	}

	slices.SortStableFunc(candidates, func(left, right allocationCandidate) int {
		if remainderCmp := right.remainder.Cmp(left.remainder); remainderCmp != 0 {
			return remainderCmp
		}

		if input.CompareKey != nil {
			if keyCmp := input.CompareKey(left.key, right.key); keyCmp != 0 {
				return keyCmp
			}
		}

		return cmp.Compare(left.index, right.index)
	})

	unit := currencyUnit(calculator)
	remaining := input.Amount.Sub(allocated)
	for remaining.GreaterThanOrEqual(unit) {
		distributed := false

		for i := range candidates {
			if remaining.LessThan(unit) {
				break
			}

			next := candidates[i].allocated.Add(unit)
			if next.GreaterThan(candidates[i].amount) {
				continue
			}

			candidates[i].allocated = next
			remaining = remaining.Sub(unit)
			distributed = true
		}

		if !distributed {
			return nil, errors.New("cannot distribute remaining allocation without exceeding item amounts")
		}
	}

	slices.SortFunc(candidates, func(left, right allocationCandidate) int {
		return cmp.Compare(left.index, right.index)
	})

	allocations := make([]AmountAllocation[T], 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.allocated.IsZero() {
			continue
		}

		allocations = append(allocations, AmountAllocation[T]{
			Key:    candidate.key,
			Amount: candidate.allocated,
		})
	}

	return allocations, nil
}

func validateWeightedAllocationInput[T any](calculator Calculator, input WeightedAllocationInput[T]) error {
	var errs []error

	if err := calculator.Validate(); err != nil {
		return err
	}

	if input.Amount.Sign() < 0 {
		errs = append(errs, errors.New("amount must be non-negative"))
	}

	if !calculator.IsRoundedToPrecision(input.Amount) {
		errs = append(errs, errors.New("amount must be rounded to currency precision"))
	}

	if len(input.Items) == 0 && !input.Amount.IsZero() {
		errs = append(errs, errors.New("items are required for a non-zero amount"))
	}

	totalWeight := alpacadecimal.Zero
	for i, item := range input.Items {
		if item.Weight.Sign() <= 0 {
			errs = append(errs, fmt.Errorf("items[%d].weight must be positive", i))
			continue
		}

		totalWeight = totalWeight.Add(item.Weight)
	}

	return errors.Join(errs...)
}

func validateAmountAllocationInput[T any](calculator Calculator, input AmountAllocationInput[T]) error {
	var errs []error

	if err := calculator.Validate(); err != nil {
		return err
	}

	if input.Amount.Sign() < 0 {
		errs = append(errs, errors.New("amount must be non-negative"))
	}

	if !calculator.IsRoundedToPrecision(input.Amount) {
		errs = append(errs, errors.New("amount must be rounded to currency precision"))
	}

	if len(input.Items) == 0 && !input.Amount.IsZero() {
		errs = append(errs, errors.New("items are required for a non-zero amount"))
	}

	totalAmount := alpacadecimal.Zero
	for i, item := range input.Items {
		if item.Amount.Sign() <= 0 {
			errs = append(errs, fmt.Errorf("items[%d].amount must be positive", i))
			continue
		}

		if !calculator.IsRoundedToPrecision(item.Amount) {
			errs = append(errs, fmt.Errorf("items[%d].amount must be rounded to currency precision", i))
		}

		totalAmount = totalAmount.Add(item.Amount)
	}

	if input.Amount.GreaterThan(totalAmount) {
		errs = append(errs, errors.New("amount must not exceed total item amount"))
	}

	return errors.Join(errs...)
}

func currencyUnit(calculator Calculator) alpacadecimal.Decimal {
	return calculator.Unit()
}
