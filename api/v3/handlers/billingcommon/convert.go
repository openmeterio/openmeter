// Package billingcommon holds converter helpers shared by the billing-related
// v3 handler packages (customer charges, invoices), so that neither handler
// package needs to import the other.
package billingcommon

import (
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func ToAPIBillingTotals(t totals.Totals) api.BillingTotals {
	return api.BillingTotals{
		Amount:              t.Amount.String(),
		ChargesTotal:        t.ChargesTotal.String(),
		CreditsTotal:        t.CreditsTotal.String(),
		DiscountsTotal:      t.DiscountsTotal.String(),
		TaxesExclusiveTotal: t.TaxesExclusiveTotal.String(),
		TaxesInclusiveTotal: t.TaxesInclusiveTotal.String(),
		TaxesTotal:          t.TaxesTotal.String(),
		Total:               t.Total.String(),
	}
}

func ToAPIValidationIssues(issues billing.ValidationIssues) *[]api.BillingValidationIssue {
	if len(issues) == 0 {
		return nil
	}

	out := make([]api.BillingValidationIssue, len(issues))
	for idx, issue := range issues {
		out[idx] = api.BillingValidationIssue{
			Attributes: lo.EmptyableToPtr(map[string]any(issue.Attributes)),
			Code:       issue.Code,
			Component:  lo.EmptyableToPtr(string(issue.Component)),
			Field:      lo.EmptyableToPtr(issue.Path),
			Message:    issue.Message,
			Severity:   api.BillingValidationIssueSeverity(issue.Severity),
		}
	}

	return &out
}

func ConvertClosedPeriodToAPI(p timeutil.ClosedPeriod) api.ClosedPeriod {
	return api.ClosedPeriod{From: p.From, To: p.To}
}

func ConvertSubscriptionRefToAPI(ref meta.SubscriptionReference) api.BillingSubscriptionReference {
	var out api.BillingSubscriptionReference
	out.Id = ref.SubscriptionID
	out.Phase.Id = ref.PhaseID
	out.Phase.Item.Id = ref.ItemID

	return out
}

type lifecycleControllerConfig struct {
	manualOverride bool
}

type LifecycleControllerOption func(*lifecycleControllerConfig)

// WithManualOverride marks the API lifecycle controller manual when a charge
// override exists even if the base intent remains subscription-owned for sync.
func WithManualOverride(manualOverride bool) LifecycleControllerOption {
	return func(config *lifecycleControllerConfig) {
		config.manualOverride = manualOverride
	}
}

func ConvertLifecycleControllerToAPI(mb billing.InvoiceLineManagedBy, options ...LifecycleControllerOption) api.BillingLifecycleController {
	config := lifecycleControllerConfig{}
	for _, option := range options {
		option(&config)
	}

	if config.manualOverride || mb == billing.ManuallyManagedLine {
		return api.BillingLifecycleControllerManual
	}

	return api.BillingLifecycleControllerSystem
}
