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
	if l.line.UsageBased.PreLinePeriodQuantity != nil {
		if l.line.UsageBased.MeteredPreLinePeriodQuantity == nil {
			return l, fmt.Errorf("no metered pre line period quantity set for line[%s]", l.line.ID)
		}

		l.line.UsageBased.PreLinePeriodQuantity = lo.ToPtr(l.line.UsageBased.MeteredPreLinePeriodQuantity.Copy())
	}

	if l.line.UsageBased.Quantity == nil {
		return l, fmt.Errorf("no quantity set for line[%s]", l.line.ID)
	}

	if l.line.UsageBased.MeteredQuantity == nil {
		return l, fmt.Errorf("no metered quantity set for line[%s]", l.line.ID)
	}

	l.line.UsageBased.Quantity = lo.ToPtr(l.line.UsageBased.MeteredQuantity.Copy())

	return l, nil
}
