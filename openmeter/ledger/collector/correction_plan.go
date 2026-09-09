package collector

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/models"
)

// correctionPosition is a remaining economic slice. Readers supply original
// collection/backing order, never recognition or replacement-segment order.
// References used to write the correction stay with the reader, keyed by id.
type correctionPosition struct {
	id         string
	recordedAt time.Time
	orderKey   string
	order      int
	uncovered  bool
	accrued    alpacadecimal.Decimal
	earnings   alpacadecimal.Decimal
	coverage   alpacadecimal.Decimal
}

func (p correctionPosition) amount() alpacadecimal.Decimal {
	return p.accrued.Add(p.earnings).Add(p.coverage)
}

type collectionCorrectionInput struct {
	amount    alpacadecimal.Decimal
	positions []correctionPosition
}

func (i collectionCorrectionInput) Validate() error {
	var errs []error
	if i.amount.IsNegative() {
		errs = append(errs, errors.New("correction amount must not be negative"))
	}
	ids := make(map[string]bool)
	for _, p := range i.positions {
		if p.id == "" || ids[p.id] {
			errs = append(errs, errors.New("correction positions require distinct identifiers"))
		}
		ids[p.id] = true
		if p.accrued.IsNegative() || p.earnings.IsNegative() || p.coverage.IsNegative() {
			errs = append(errs, fmt.Errorf("negative remaining collection position %s", p.id))
		}
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type correctionSelection struct {
	id       string
	amount   alpacadecimal.Decimal
	earnings alpacadecimal.Decimal
}

// planCollectionCorrection selects sources before their downstream state. Both
// storage formats use this decision; recognition batching cannot change which
// funding is returned. Backed advance remains ahead of uncovered advance.
func planCollectionCorrection(input collectionCorrectionInput) ([]correctionSelection, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	positions := slices.Clone(input.positions)
	slices.SortStableFunc(positions, func(a, b correctionPosition) int {
		if a.uncovered != b.uncovered {
			if a.uncovered {
				return 1
			}
			return -1
		}
		if c := a.recordedAt.Compare(b.recordedAt); c != 0 {
			return -c
		}
		if c := cmp.Compare(a.order, b.order); c != 0 {
			return -c
		}
		if c := cmp.Compare(a.orderKey, b.orderKey); c != 0 {
			return -c
		}
		return -cmp.Compare(a.id, b.id)
	})
	remaining := input.amount
	var out []correctionSelection
	for _, position := range positions {
		take := minDecimal(remaining, position.amount())
		if !take.IsPositive() {
			continue
		}
		out = append(out, correctionSelection{id: position.id, amount: take, earnings: minDecimal(take, position.earnings)})
		remaining = remaining.Sub(take)
	}
	if remaining.IsPositive() {
		return nil, fmt.Errorf("correction exceeds remaining collection balance by %s", remaining)
	}
	return out, nil
}
