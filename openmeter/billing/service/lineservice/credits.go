package lineservice

import (
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type creditsMutator struct{}

var _ PostCalculationMutator = (*creditsMutator)(nil)

func (m *creditsMutator) Mutate(i PricerCalculateInput, pricerResult newDetailedLinesInput) (newDetailedLinesInput, error) {
	for _, creditToApply := range i.line.CreditsApplied {
		creditValueRemaining := i.currency.RoundToPrecision(creditToApply.Amount)

		for idx := range pricerResult {
			totalAmount := pricerResult[idx].TotalAmount(i.currency)

			if totalAmount.LessThanOrEqual(creditValueRemaining) {
				creditValueRemaining = creditValueRemaining.Sub(totalAmount)
				pricerResult[idx].CreditsApplied = append(pricerResult[idx].CreditsApplied, billing.CreditApplied{
					Amount:      totalAmount,
					Description: creditToApply.Description,
				})
			} else {
				pricerResult[idx].CreditsApplied = append(pricerResult[idx].CreditsApplied, billing.CreditApplied{
					Amount:      creditValueRemaining,
					Description: creditToApply.Description,
				})

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
