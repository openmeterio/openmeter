package billingservice

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	stdinvoicesanity "github.com/openmeterio/openmeter/openmeter/billing/stdinvoice/sanity"
)

// InvoiceSanityEnforcement distinguishes lifecycle execution from read-only
// action prediction, which may operate on an invoice without expanded lines.
type InvoiceSanityEnforcement string

const (
	InvoiceSanityEnforcementDisabled      InvoiceSanityEnforcement = "enforcement_disabled"
	InvoiceSanityEnforcementReportingOnly InvoiceSanityEnforcement = "enforcement_reporting_only"
	InvoiceSanityEnforcementEnabled       InvoiceSanityEnforcement = "enforcement_enabled"
)

// checkInvoiceSanity validates the current snapshot before an app consumes it.
// Callers invoke it for draft and issuing handoffs that require consistent totals.
// Disabled mode skips validation entirely. Otherwise validation issues are logged
// together and returned only in enabled mode; system errors always propagate.
// Callers own issue persistence and state transitions, keeping read-only
// readiness probes free of mutations.
func (m *InvoiceStateMachine) checkInvoiceSanity(ctx context.Context) error {
	if m.InvoiceSanityEnforcement == InvoiceSanityEnforcementDisabled {
		return nil
	}

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
			"enforced", m.InvoiceSanityEnforcement == InvoiceSanityEnforcementEnabled,
			"issues", issues,
		)
	}

	if m.InvoiceSanityEnforcement != InvoiceSanityEnforcementEnabled {
		return nil
	}

	return err
}
