package mutator

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/rate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type MinAmountCommitment struct{}

var _ PostCalculationMutator = (*MinAmountCommitment)(nil)

func (m *MinAmountCommitment) Mutate(i rate.PricerCalculateInput, pricerResult rating.DetailedLines) (rating.DetailedLines, error) {
	commitments := i.GetPrice().GetCommitments()

	if commitments.MinimumAmount == nil {
		return pricerResult, nil
	}

	// Minimum amount commitments are always applied to the last line in period
	if !i.IsLastInPeriod() {
		return pricerResult, nil
	}

	previouslyBilledAmount, err := i.GetPreviouslyBilledAmount()
	if err != nil {
		return pricerResult, err
	}

	totalBilledAmount := pricerResult.Sum(i.CurrencyCalculator).Add(previouslyBilledAmount)

	if totalBilledAmount.LessThan(*commitments.MinimumAmount) {
		minimumSpendAmount := i.CurrencyCalculator.RoundToPrecision(commitments.MinimumAmount.Sub(totalBilledAmount))

		// Minimum spend is always billed for the whole period, this is a noop if we are not using progressive billing
		period := i.GetServicePeriod()
		if i.IsProgressivelyBilled() {
			period = i.FullProgressivelyBilledServicePeriod
		}

		pricerResult = append(pricerResult, rating.DetailedLine{
			Name:                   fmt.Sprintf("%s: minimum spend", i.GetName()),
			Quantity:               alpacadecimal.NewFromFloat(1),
			PerUnitAmount:          minimumSpendAmount,
			ChildUniqueReferenceID: rating.MinSpendChildUniqueReferenceID,
			PaymentTerm:            productcatalog.InArrearsPaymentTerm,
			Category:               billing.FlatFeeCategoryCommitment,
			Period:                 &period,
		})
	}

	return pricerResult, nil
}

type MaxAmountCommitment struct{}

var _ PostCalculationMutator = (*MaxAmountCommitment)(nil)

func (m *MaxAmountCommitment) Mutate(i rate.PricerCalculateInput, pricerResult rating.DetailedLines) (rating.DetailedLines, error) {
	commitments := i.GetPrice().GetCommitments()

	if commitments.MaximumAmount == nil {
		return pricerResult, nil
	}

	maxSpend := *commitments.MaximumAmount

	previouslyBilledAmount, err := i.GetPreviouslyBilledAmount()
	if err != nil {
		return pricerResult, err
	}

	totalBilled := previouslyBilledAmount

	// Let's start applying the max amount commitment to the new lines

	for idx := range pricerResult {
		// Total spends after adding the line's amount
		pricerResult[idx] = pricerResult[idx].AddDiscountForOverage(rating.AddDiscountInput{
			BilledAmountBeforeLine: totalBilled,
			MaxSpend:               maxSpend,
			Currency:               i.CurrencyCalculator,
		})

		totalBilled = totalBilled.Add(pricerResult[idx].TotalAmount(i.CurrencyCalculator))
	}

	return pricerResult, nil
}
