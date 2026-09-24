package service

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	billingfeaturemeter "github.com/openmeterio/openmeter/openmeter/billing/featuremeter"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	activeRunInvoiceAssignmentIssueMessage = "an ongoing realization run must be completed before this charge can be assigned to another invoice"
)

// clearInvoiceAssignmentIssueWithoutCurrentRun removes line-engine-owned assignment issues after
// the charge releases its current realization run. Those issues are valid only while that run blocks assignment.
func clearInvoiceAssignmentIssueWithoutCurrentRun(base usagebased.ChargeBase) usagebased.ChargeBase {
	if base.State.CurrentRealizationRunID == nil {
		base.ValidationIssues = base.ValidationIssues.WithoutComponent(usagebased.ValidationIssueComponentLineEngine)
	}

	return base
}

type invoiceAssignmentGateCheck struct {
	ValidationIssues   billing.ValidationIssues
	ExcludeFromInvoice bool
}

func checkFeatureMeterAvailability(featureMeters billingfeaturemeter.FeatureMeters, charge usagebased.Charge) (invoiceAssignmentGateCheck, error) {
	_, err := featureMeters.Get(charge)
	if err == nil {
		return invoiceAssignmentGateCheck{}, nil
	}

	err = billing.ValidationWithComponent(billing.ValidationComponentProductCatalog, err)
	issues, systemErr := billing.ToValidationIssues(err)
	if systemErr != nil {
		return invoiceAssignmentGateCheck{}, systemErr
	}

	return invoiceAssignmentGateCheck{
		ValidationIssues:   issues,
		ExcludeFromInvoice: true,
	}, nil
}

func checkCurrentRealizationRun(charge usagebased.Charge) (invoiceAssignmentGateCheck, error) {
	if charge.State.CurrentRealizationRunID == nil {
		return invoiceAssignmentGateCheck{}, nil
	}

	currentRun, err := charge.GetCurrentRealizationRun()
	if err != nil {
		return invoiceAssignmentGateCheck{}, err
	}

	issue, err := newActiveRunInvoiceAssignmentIssue(currentRun)
	if err != nil {
		return invoiceAssignmentGateCheck{}, err
	}

	return invoiceAssignmentGateCheck{
		ValidationIssues:   billing.ValidationIssues{issue},
		ExcludeFromInvoice: true,
	}, nil
}

func replaceValidationIssueComponent(existing billing.ValidationIssues, component billing.ComponentName, replacement billing.ValidationIssues) (billing.ValidationIssues, bool) {
	hasExisting := existing.HasComponent(component)
	if !hasExisting && len(replacement) == 0 {
		return existing, false
	}

	return append(existing.WithoutComponent(component), replacement...), true
}

// Only charge rating warnings are forwarded to the invoice; other charge issues retain their own ownership.
func ratingValidationIssues(charge usagebased.Charge) billing.ValidationIssues {
	return lo.Filter(charge.ValidationIssues, func(issue billing.ValidationIssue, _ int) bool {
		return issue.Component == billing.ValidationComponentBillingRating
	})
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
