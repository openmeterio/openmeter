package invoicecalc

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

// CalculateDraftUntil calculates the draft until date
func CalculateDraftUntil(i *billing.Invoice, _ CalculatorDependencies) error {
	if !i.Workflow.Config.Invoicing.AutoAdvance {
		i.DraftUntil = nil
		return nil
	}

	collectionAt := lo.FromPtrOr(i.CollectionAt, i.CreatedAt)
	draftUntil, _ := i.Workflow.Config.Invoicing.DraftPeriod.AddTo(collectionAt)
	i.DraftUntil = &draftUntil

	return nil
}
