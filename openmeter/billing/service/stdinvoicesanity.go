package billingservice

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	stdinvoicesanity "github.com/openmeterio/openmeter/openmeter/billing/stdinvoice/sanity"
)

// checkInvoiceSanity validates the current snapshot before an app consumes it.
// Validation issues are logged together and returned only when enforcement is
// enabled; system errors always propagate. Callers own issue persistence and
// state transitions, keeping read-only readiness probes free of mutations.
func (m *InvoiceStateMachine) checkInvoiceSanity(ctx context.Context) error {
	err := stdinvoicesanity.Validate(m.Invoice)
	if err == nil {
		return nil
	}

	issues, issueErr := billing.ToValidationIssues(err)
	if issueErr != nil {
		return fmt.Errorf("invoice sanity check[id=%s]: %w", m.Invoice.ID, issueErr)
	}

	if len(issues) > 0 {
		m.Logger.ErrorContext(ctx, "invoice sanity validation failed",
			"invoice_id", m.Invoice.ID,
			"namespace", m.Invoice.Namespace,
			"invoice_status", m.Invoice.Status,
			"enforced", m.EnforceInvoiceSanityChecks,
			"issues", issues,
		)
	}

	if !m.EnforceInvoiceSanityChecks {
		return nil
	}

	return err
}
