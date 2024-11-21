package calculation

import (
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
)

// DraftUntilIfMissing calculates the draft until date if it is missing.
// If it's set we are not updating it as the user should update that instead of manipulating the
// workflow config.
func DraftUntilIfMissing(i *billingentity.Invoice) (bool, error) {
	if !i.ExpandedFields.Workflow || i.DraftUntil != nil || !i.Workflow.Config.Invoicing.AutoAdvance {
		return false, nil
	}

	draftUntil, _ := i.Workflow.Config.Invoicing.DraftPeriod.AddTo(i.CreatedAt)
	i.DraftUntil = &draftUntil

	return true, nil
}
