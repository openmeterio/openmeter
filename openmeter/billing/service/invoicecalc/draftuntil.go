package invoicecalc

import "github.com/openmeterio/openmeter/openmeter/billing"

// DraftUntilIfMissing calculates the draft until date if it is missing.
// If it's set we are not updating it as the user should update that instead of manipulating the
// workflow config.
func DraftUntilIfMissing(i *billing.Invoice, _ CalculatorDependencies) error {
	if i.DraftUntil != nil || !i.Workflow.Config.Invoicing.AutoAdvance {
		return nil
	}

	draftUntil, _ := i.Workflow.Config.Invoicing.DraftPeriod.AddTo(i.CreatedAt)
	i.DraftUntil = &draftUntil

	return nil
}
