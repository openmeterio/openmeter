package mutator

import (
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/pricer"
	"github.com/openmeterio/openmeter/openmeter/billing/pricer/service/price"
)

type Credits struct{}

var _ PostCalculationMutator = (*Credits)(nil)

func (m *Credits) Mutate(i price.PricerCalculateInput, pricerResult pricer.DetailedLines) (pricer.DetailedLines, error) {
	for _, creditToApply := range i.GetCreditsApplied() {
		creditValueRemaining := i.CurrencyCalculator.RoundToPrecision(creditToApply.Amount)

		for idx := range pricerResult {
			if creditValueRemaining.IsZero() {
				break
			}

			totalAmount := pricerResult[idx].TotalAmount(i.CurrencyCalculator)
			if totalAmount.IsZero() {
				continue
			}

			if totalAmount.LessThanOrEqual(creditValueRemaining) {
				creditValueRemaining = creditValueRemaining.Sub(totalAmount)
				pricerResult[idx].CreditsApplied = append(pricerResult[idx].CreditsApplied, creditToApply.CloneWithAmount(totalAmount))
			} else {
				pricerResult[idx].CreditsApplied = append(pricerResult[idx].CreditsApplied, creditToApply.CloneWithAmount(creditValueRemaining))

				creditValueRemaining = alpacadecimal.Zero
				break
			}
		}

		if creditValueRemaining.IsPositive() {
			// TODO: Error code/validation error?
			// This is critical, as it means that charges/ledger has allocated more credits than the line is worth
			// thus we would charge the customer more credits that we actually have usage for.

			return pricerResult, billing.ErrInvoiceLineCreditsNotConsumedFully
		}
	}

	return pricerResult, nil
}
