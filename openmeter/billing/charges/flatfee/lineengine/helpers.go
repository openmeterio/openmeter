package lineengine

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
)

func convertCreditRealizations(creditRealizations creditrealization.Realizations) billing.CreditsApplied {
	return lo.Map(creditRealizations, func(creditRealization creditrealization.Realization, _ int) billing.CreditApplied {
		return billing.CreditApplied{
			Amount:              creditRealization.Amount,
			CreditRealizationID: creditRealization.ID,
		}
	})
}
