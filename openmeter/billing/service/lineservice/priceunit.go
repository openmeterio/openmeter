package lineservice

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type unitPricer struct {
	ProgressiveBillingPricer
}

var _ Pricer = (*unitPricer)(nil)

func (p unitPricer) Calculate(l PricerCalculateInput) (newDetailedLinesInput, error) {
	var out newDetailedLinesInput

	unitPrice, err := l.line.UsageBased.Price.AsUnit()
	if err != nil {
		return nil, fmt.Errorf("converting price to unit price: %w", err)
	}

	if l.linePeriodQty.IsPositive() {
		out = newDetailedLinesInput{
			{
				Name:                   fmt.Sprintf("%s: usage in period", l.line.Name),
				Quantity:               l.linePeriodQty,
				PerUnitAmount:          unitPrice.Amount,
				ChildUniqueReferenceID: UnitPriceUsageChildUniqueReferenceID,
				PaymentTerm:            productcatalog.InArrearsPaymentTerm,
			},
		}
	}

	amountBilledInPreviousPeriods := l.currency.RoundToPrecision(l.preLinePeriodQty.Mul(unitPrice.Amount))

	detailedLines, err := l.applyCommitments(applyCommitmentsInput{
		Commitments:                   unitPrice.Commitments,
		DetailedLines:                 out,
		AmountBilledInPreviousPeriods: amountBilledInPreviousPeriods,
		MinimumSpendReferenceID:       UnitPriceMinSpendChildUniqueReferenceID,
	})
	if err != nil {
		return nil, err
	}

	return detailedLines, nil
}
