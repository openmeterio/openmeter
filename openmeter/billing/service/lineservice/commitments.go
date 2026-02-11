package lineservice

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type minAmountCommitmentMutator struct{}

var _ PostCalculationMutator = (*minAmountCommitmentMutator)(nil)

func (m *minAmountCommitmentMutator) Mutate(i PricerCalculateInput, pricerResult newDetailedLinesInput) (newDetailedLinesInput, error) {
	commitments := i.line.UsageBased.Price.GetCommitments()

	if commitments.MinimumAmount == nil {
		return pricerResult, nil
	}

	// Minimum amount commitments are always applied to the last line in period
	if !i.IsLastInPeriod() {
		return pricerResult, nil
	}

	previouslyBilledAmount, err := getPreviouslyBilledAmount(i)
	if err != nil {
		return pricerResult, err
	}

	totalBilledAmount := pricerResult.Sum(i.currency).Add(previouslyBilledAmount)

	if totalBilledAmount.LessThan(*commitments.MinimumAmount) {
		minimumSpendAmount := i.currency.RoundToPrecision(commitments.MinimumAmount.Sub(totalBilledAmount))

		// Minimum spend is always billed for the whole period, this is a noop if we are not using progressive billing
		period := i.line.Period
		if i.line.SplitLineGroupID != nil {
			if i.line.SplitLineHierarchy == nil {
				return pricerResult, fmt.Errorf("line[%s] does not have a split line hierarchy, but is a progressive billed line", i.line.ID)
			}

			period = i.line.SplitLineHierarchy.Group.ServicePeriod
		}

		pricerResult = append(pricerResult, newDetailedLineInput{
			Name:                   fmt.Sprintf("%s: minimum spend", i.line.Name),
			Quantity:               alpacadecimal.NewFromFloat(1),
			PerUnitAmount:          minimumSpendAmount,
			ChildUniqueReferenceID: MinSpendChildUniqueReferenceID,
			PaymentTerm:            productcatalog.InArrearsPaymentTerm,
			Category:               billing.FlatFeeCategoryCommitment,
			Period:                 &period,
		})
	}

	return pricerResult, nil
}

type maxAmountCommitmentMutator struct{}

var _ PostCalculationMutator = (*maxAmountCommitmentMutator)(nil)

func (m *maxAmountCommitmentMutator) Mutate(i PricerCalculateInput, pricerResult newDetailedLinesInput) (newDetailedLinesInput, error) {
	commitments := i.line.UsageBased.Price.GetCommitments()

	if commitments.MaximumAmount == nil {
		return pricerResult, nil
	}

	maxSpend := *commitments.MaximumAmount

	previouslyBilledAmount, err := getPreviouslyBilledAmount(i)
	if err != nil {
		return pricerResult, err
	}

	totalBilled := previouslyBilledAmount

	// Let's start applying the max amount commitment to the new lines

	for idx, line := range pricerResult {
		// Total spends after adding the line's amount
		pricerResult[idx] = pricerResult[idx].AddDiscountForOverage(addDiscountInput{
			BilledAmountBeforeLine: totalBilled,
			MaxSpend:               maxSpend,
			Currency:               i.currency,
		})

		totalBilled = totalBilled.Add(line.TotalAmount(i.currency))
	}

	return pricerResult, nil
}

func getPreviouslyBilledAmount(l PricerCalculateInput) (alpacadecimal.Decimal, error) {
	if l.line.SplitLineGroupID == nil {
		return alpacadecimal.Zero, nil
	}

	if l.line.SplitLineHierarchy == nil {
		return alpacadecimal.Zero, fmt.Errorf("line[%s] does not have a progressive line hierarchy, but is a progressive billed line", l.line.ID)
	}

	return l.line.SplitLineHierarchy.SumNetAmount(billing.SumNetAmountInput{
		PeriodEndLTE: l.line.Period.Start,
		// TODO[later]: Should we include charges here? For now it's fine to not include them, as only
		// minimum amount charges can happen, but if later we add more charge types, we will have to
		// include them here.
		IncludeCharges: false,
	})
}
