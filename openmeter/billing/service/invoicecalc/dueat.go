package invoicecalc

import (
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

func CalculateDueAt(i *billing.StandardInvoice) error {
	if !i.Workflow.Config.Invoicing.AutoAdvance {
		// In case of manual approval dueAt is started from the moment of the invoice is issued
		// to prevent the invoice from being overdue on issuance

		if i.IssuedAt == nil {
			// If we don't know when the invoice was issued we cannot calculate the dueAt
			return nil
		}

		dueAt, _ := i.Workflow.Config.Invoicing.DueAfter.Period.AddTo(*i.IssuedAt)

		i.DueAt = lo.ToPtr(dueAt.Truncate(time.Second))

		return nil
	}

	// Auto advance is enabled, so we can calculate the dueAt
	if i.DraftUntil == nil {
		// If we don't know the draftUntil we cannot yet calculate the dueAt
		return nil
	}

	dueAt, _ := i.Workflow.Config.Invoicing.DueAfter.Period.AddTo(*i.DraftUntil)

	// Note: Stripe uses seconds precision, so let's truncate for easier testing
	i.DueAt = lo.ToPtr(dueAt.Truncate(time.Second))

	return nil
}
