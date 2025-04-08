package lineservice

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type dynamicPricer struct {
	ProgressiveBillingPricer
}

var _ Pricer = (*dynamicPricer)(nil)

func (p dynamicPricer) Calculate(l PricerCalculateInput) (newDetailedLinesInput, error) {
	var out newDetailedLinesInput

	dynamicPrice, err := l.line.UsageBased.Price.AsDynamic()
	if err != nil {
		return nil, fmt.Errorf("converting price to dynamic price: %w", err)
	}

	if l.linePeriodQty.IsPositive() {
		amountInPeriod := l.currency.RoundToPrecision(
			l.linePeriodQty.Mul(dynamicPrice.Multiplier),
		)

		out = newDetailedLinesInput{
			{
				Name:                   fmt.Sprintf("%s: usage in period", l.line.Name),
				Quantity:               alpacadecimal.NewFromInt(1),
				PerUnitAmount:          amountInPeriod,
				ChildUniqueReferenceID: UsageChildUniqueReferenceID,
				PaymentTerm:            productcatalog.InArrearsPaymentTerm,
			},
		}
	}

	amountBilledInPreviousPeriods := l.currency.RoundToPrecision(l.preLinePeriodQty.Mul(dynamicPrice.Multiplier))

	detailedLines, err := l.applyCommitments(applyCommitmentsInput{
		Commitments:                   dynamicPrice.Commitments,
		DetailedLines:                 out,
		AmountBilledInPreviousPeriods: amountBilledInPreviousPeriods,
		MinimumSpendReferenceID:       MinSpendChildUniqueReferenceID,
	})
	if err != nil {
		return nil, err
	}

	return detailedLines, nil
}
