package invoicecalc

import "github.com/openmeterio/openmeter/openmeter/billing"

// CalculateDraftUntil calculates the draft until date
func CalculateDraftUntil(i *billing.Invoice, _ CalculatorDependencies) error {
	if !i.Workflow.Config.Invoicing.AutoAdvance {
		i.DraftUntil = nil
		return nil
	}

	draftUntil, _ := i.Workflow.Config.Invoicing.DraftPeriod.AddTo(i.CreatedAt)
	i.DraftUntil = &draftUntil

	return nil
}
