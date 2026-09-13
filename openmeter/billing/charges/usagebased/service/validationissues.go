package service

import (
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	activeRunInvoiceAssignmentIssueMessage = "an ongoing realization run must be completed before this charge can be assigned to another invoice"
)

// clearInvoiceAssignmentIssueWithoutCurrentRun removes the invoice-assignment issue after the
// charge releases its current realization run. The issue is valid only while that run blocks assignment.
func clearInvoiceAssignmentIssueWithoutCurrentRun(base usagebased.ChargeBase) usagebased.ChargeBase {
	if base.State.CurrentRealizationRunID == nil && base.ValidationIssues.HasWithComponentCode(
		usagebased.ValidationIssueComponentLineEngine,
		usagebased.ValidationIssueCodeInvoiceAssignmentBlockedActiveRun,
	) {
		base.ValidationIssues = base.ValidationIssues.Without(
			usagebased.ValidationIssueComponentLineEngine,
			usagebased.ValidationIssueCodeInvoiceAssignmentBlockedActiveRun,
		)
	}

	return base
}

func hasFeatureDependencyValidationIssue(issues billing.ValidationIssues) bool {
	return slices.ContainsFunc(issues, isFeatureDependencyValidationIssue)
}

func withoutFeatureDependencyValidationIssues(issues billing.ValidationIssues) billing.ValidationIssues {
	return slices.DeleteFunc(slices.Clone(issues), isFeatureDependencyValidationIssue)
}

func isFeatureDependencyValidationIssue(issue billing.ValidationIssue) bool {
	return issue.Code == billing.ErrInvoiceLineFeatureNotFound.Code ||
		issue.Code == billing.ErrInvoiceLineFeatureHasNoMeters.Code
}

func newActiveRunInvoiceAssignmentIssue(run usagebased.RealizationRun) (billing.ValidationIssue, error) {
	if run.InvoiceID == nil || *run.InvoiceID == "" {
		return billing.ValidationIssue{}, fmt.Errorf("current realization run[%s] invoice ID is required", run.ID.ID)
	}

	if run.LineID == nil || *run.LineID == "" {
		return billing.ValidationIssue{}, fmt.Errorf("current realization run[%s] line ID is required", run.ID.ID)
	}

	// TODO: add structured charge field context when billing validation issues support models.FieldDescriptor.
	return billing.ValidationIssue{
		Severity:  billing.ValidationIssueSeverityCritical,
		Message:   activeRunInvoiceAssignmentIssueMessage,
		Code:      usagebased.ValidationIssueCodeInvoiceAssignmentBlockedActiveRun,
		Component: usagebased.ValidationIssueComponentLineEngine,
		Attributes: models.Annotations{
			"invoice_id": *run.InvoiceID,
			"line_id":    *run.LineID,
		},
	}, nil
}
