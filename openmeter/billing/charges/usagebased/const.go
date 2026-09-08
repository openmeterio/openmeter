package usagebased

import (
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

const (
	InternalCollectionPeriod time.Duration = time.Minute

	ValidationIssueCodeInvoiceAssignmentBlockedActiveRun = "usage_based_invoice_assignment_blocked_active_run"
	ValidationIssueComponentLineEngine                   = billing.ComponentName("charges.usagebased.lineengine")
)
