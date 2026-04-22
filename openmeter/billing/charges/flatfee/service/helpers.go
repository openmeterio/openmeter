package service

import (
	"context"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
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

func (s *service) persistDetailedLines(ctx context.Context, charge flatfee.Charge, line billing.StandardLine) error {
	return s.adapter.UpsertDetailedLines(ctx, charge.GetChargeID(), lo.Map(line.DetailedLines, func(detailedLine billing.DetailedLine, _ int) flatfee.DetailedLine {
		return detailedLine.Base.Clone()
	}))
}
