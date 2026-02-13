package lineservice

import (
	"fmt"

	"github.com/samber/lo"
)

type setQuantityToMeteredQuantity struct{}

var _ PreCalculationMutator = (*setQuantityToMeteredQuantity)(nil)

// setQuantityToMeteredQuantity is a mutator that resets the quantity of the line to the metered quantity before discount
// is applied. This is to ensure that the discount is applied on the metered quantity, not the original quantity.
func (m *setQuantityToMeteredQuantity) Mutate(l PricerCalculateInput) (PricerCalculateInput, error) {
	if l.line.PreLinePeriodQuantity != nil {
		if l.line.MeteredPreLinePeriodQuantity == nil {
			return l, fmt.Errorf("no metered pre line period quantity set for line[%s]", l.line.ID)
		}

		l.line.PreLinePeriodQuantity = lo.ToPtr(l.line.MeteredPreLinePeriodQuantity.Copy())
	}

	if l.line.Quantity == nil {
		return l, fmt.Errorf("no quantity set for line[%s]", l.line.ID)
	}

	if l.line.MeteredQuantity == nil {
		return l, fmt.Errorf("no metered quantity set for line[%s]", l.line.ID)
	}

	l.line.Quantity = lo.ToPtr(l.line.MeteredQuantity.Copy())

	return l, nil
}
