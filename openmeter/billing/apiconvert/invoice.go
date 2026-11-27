package apiconvert

import (
	"fmt"

	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	appconvert "github.com/openmeterio/openmeter/openmeter/app/apiconvert"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/externalid"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// ToAPIStandardInvoice maps a standard invoice. taxCodes must hold the invoice's default Tax
// Code and every Tax Code its lines' tax configs reference (see ResolveInvoiceTaxCodes) for
// their stripe.code to show.
func ToAPIStandardInvoice(invoice billing.StandardInvoice, taxCodes map[string]*taxcode.TaxCode) (apiv1models.Invoice, error) {
	var apps *apiv1models.BillingProfileAppsOrReference
	var err error
	collectionAt := invoice.CollectionAt

	// Preserve the historic API shape for standard invoices by emulating collectionAt when the
	// domain model intentionally leaves it nil (for example, flat-fee-only invoices).
	if collectionAt == nil {
		collectionAt = lo.ToPtr(invoice.CreatedAt)
	}

	// Sort the lines to make the response more consistent (internally we don't care about the order)
	invoice.SortLines()

	if invoice.Workflow.Apps != nil {
		apps, err = ToAPIBillingProfileApps(invoice.Workflow.Apps)
		if err != nil {
			return apiv1models.Invoice{}, fmt.Errorf("failed to map profile apps to API: %w", err)
		}
	} else {
		apps, err = ToAPIBillingProfileAppReferences(&invoice.Workflow.AppReferences)
		if err != nil {
			return apiv1models.Invoice{}, fmt.Errorf("failed to map profile app references to API: %w", err)
		}
	}

	out := apiv1models.Invoice{
		Id: invoice.ID,

		CreatedAt:            invoice.CreatedAt,
		UpdatedAt:            invoice.UpdatedAt,
		DeletedAt:            invoice.DeletedAt,
		IssuedAt:             invoice.IssuedAt,
		VoidedAt:             invoice.VoidedAt,
		DueAt:                invoice.DueAt,
		CollectionAt:         collectionAt,
		DraftUntil:           invoice.DraftUntil,
		SentToCustomerAt:     invoice.SentToCustomerAt,
		QuantitySnapshotedAt: invoice.QuantitySnapshotedAt,
		Period:               mapPeriodToAPI(invoice.Period),

		Currency: string(invoice.Currency),
		Customer: ToAPIBillingInvoiceCustomerExtendedDetails(invoice.Customer),

		Number:      invoice.Number,
		Description: invoice.Description,
		Metadata:    convert.MapToPointer(invoice.Metadata),

		Status: apiv1models.InvoiceStatus(invoice.Status.ShortStatus()),
		StatusDetails: apiv1models.InvoiceStatusDetails{
			Failed:         invoice.StatusDetails.Failed,
			Immutable:      invoice.StatusDetails.Immutable,
			ExtendedStatus: string(invoice.Status),

			AvailableActions: mapInvoiceAvailableActionsToAPI(invoice.StatusDetails.AvailableActions),
		},
		Supplier: ToAPIBillingParty(invoice.Supplier),
		Totals:   mapTotalsToAPI(invoice.Totals),
		// TODO[OM-943]: Implement
		Payment: nil,
		Type:    apiv1models.InvoiceType(invoice.Type),
		ValidationIssues: convert.SliceToPointer(
			lo.Map(invoice.ValidationIssues, func(v billing.ValidationIssue, _ int) apiv1models.ValidationIssue {
				return apiv1models.ValidationIssue{
					Id:        v.ID,
					CreatedAt: v.CreatedAt,
					UpdatedAt: v.UpdatedAt,
					DeletedAt: v.DeletedAt,

					Severity:  apiv1models.ValidationIssueSeverity(v.Severity),
					Message:   v.Message,
					Code:      lo.EmptyableToPtr(v.Code),
					Component: string(v.Component),
					Field:     lo.EmptyableToPtr(v.Path),
				}
			})),
		ExternalIds: mapInvoiceAppExternalIdsToAPI(invoice.ExternalIDs),
	}

	workflowConfig, err := ToAPIBillingWorkflow(invoice.Workflow.Config, taxCodes)
	if err != nil {
		return apiv1models.Invoice{}, fmt.Errorf("failed to map workflow config to API: %w", err)
	}

	out.Workflow = apiv1models.InvoiceWorkflowSettings{
		Apps:                   apps,
		SourceBillingProfileId: invoice.Workflow.SourceBillingProfileID,
		Workflow:               workflowConfig,
	}

	outLines, err := slicesx.MapWithErr(invoice.Lines.OrEmpty(), func(line *billing.StandardLine) (apiv1models.InvoiceLine, error) {
		mappedLine, err := mapInvoiceLineToAPI(line, taxCodes)
		if err != nil {
			return apiv1models.InvoiceLine{}, fmt.Errorf("failed to map billing line[%s] to API: %w", line.ID, err)
		}

		return mappedLine, nil
	})
	if err != nil {
		return apiv1models.Invoice{}, err
	}

	if len(outLines) > 0 {
		out.Lines = &outLines
	}

	return out, nil
}

// ToAPIEventInvoice maps an invoice event to the v1 invoice shape notifications carry. The event
// has no Stripe code for the invoice default; taxCodes (see ResolveInvoiceTaxCodes) supplies it
// and the lines' stripe.code.
func ToAPIEventInvoice(event billing.EventStandardInvoice, taxCodes map[string]*taxcode.TaxCode) (apiv1models.Invoice, error) {
	// Only the apps captured in the event belong in the notification snapshot.
	event.Invoice.Workflow.Apps = nil

	invoice, err := ToAPIStandardInvoice(event.Invoice, taxCodes)
	if err != nil {
		return apiv1models.Invoice{}, err
	}

	// Even an event without apps has the expanded apps shape in stored v1 payloads.
	apps := apiv1models.BillingProfileApps{}

	if event.Apps.Invoicing.Type != "" {
		apps.Invoicing, err = appconvert.ToAPIEventApp(event.Apps.Invoicing)
		if err != nil {
			return apiv1models.Invoice{}, err
		}
	}

	if event.Apps.Payment.Type != "" {
		apps.Payment, err = appconvert.ToAPIEventApp(event.Apps.Payment)
		if err != nil {
			return apiv1models.Invoice{}, err
		}
	}

	if event.Apps.Tax.Type != "" {
		apps.Tax, err = appconvert.ToAPIEventApp(event.Apps.Tax)
		if err != nil {
			return apiv1models.Invoice{}, err
		}
	}

	invoice.Workflow.Apps = &apiv1models.BillingProfileAppsOrReference{}
	if err := invoice.Workflow.Apps.FromBillingProfileApps(apps); err != nil {
		return apiv1models.Invoice{}, err
	}

	return invoice, nil
}

func ToAPIBillingInvoiceCustomerExtendedDetails(c billing.InvoiceCustomer) apiv1models.BillingInvoiceCustomerExtendedDetails {
	a := c.BillingAddress

	out := apiv1models.BillingInvoiceCustomerExtendedDetails{
		Id:   lo.ToPtr(c.CustomerID),
		Key:  c.Key,
		Name: lo.EmptyableToPtr(c.Name),
	}

	if c.UsageAttribution != nil {
		out.UsageAttribution = apiv1models.CustomerUsageAttribution{
			SubjectKeys: c.UsageAttribution.SubjectKeys,
		}
	}

	if a != nil && !lo.IsEmpty(*a) {
		out.Addresses = lo.ToPtr([]apiv1models.Address{
			{
				Country:     (*string)(a.Country),
				PostalCode:  a.PostalCode,
				State:       a.State,
				City:        a.City,
				Line1:       a.Line1,
				Line2:       a.Line2,
				PhoneNumber: a.PhoneNumber,
			},
		})
	}

	return out
}

func mapInvoiceAppExternalIdsToAPI(externalIds externalid.InvoiceExternalIDs) *apiv1models.InvoiceAppExternalIds {
	if lo.IsEmpty(externalIds) {
		return nil
	}

	return &apiv1models.InvoiceAppExternalIds{
		Invoicing: lo.EmptyableToPtr(externalIds.Invoicing),
		Payment:   lo.EmptyableToPtr(externalIds.Payment),
	}
}

func mapPeriodToAPI(p *timeutil.ClosedPeriod) *apiv1models.Period {
	if p == nil {
		return nil
	}

	// TODO[later]: let's use a common model for this
	return &apiv1models.Period{
		From: p.From,
		To:   p.To,
	}
}

func mapTotalsToAPI(t totals.Totals) apiv1models.InvoiceTotals {
	return apiv1models.InvoiceTotals{
		Amount:              t.Amount.String(),
		ChargesTotal:        t.ChargesTotal.String(),
		CreditsTotal:        t.CreditsTotal.String(),
		DiscountsTotal:      t.DiscountsTotal.String(),
		TaxesInclusiveTotal: t.TaxesInclusiveTotal.String(),
		TaxesExclusiveTotal: t.TaxesExclusiveTotal.String(),
		TaxesTotal:          t.TaxesTotal.String(),
		Total:               t.Total.String(),
	}
}

func mapInvoiceAvailableActionsToAPI(actions billing.StandardInvoiceAvailableActions) apiv1models.InvoiceAvailableActions {
	return apiv1models.InvoiceAvailableActions{
		Advance:            mapInvoiceAvailableActionDetailsToAPI(actions.Advance),
		Approve:            mapInvoiceAvailableActionDetailsToAPI(actions.Approve),
		Delete:             mapInvoiceAvailableActionDetailsToAPI(actions.Delete),
		Retry:              mapInvoiceAvailableActionDetailsToAPI(actions.Retry),
		Void:               mapInvoiceAvailableActionDetailsToAPI(actions.Void),
		SnapshotQuantities: mapInvoiceAvailableActionDetailsToAPI(actions.SnapshotQuantities),
		Invoice:            lo.If(actions.Invoice != nil, &apiv1models.InvoiceAvailableActionInvoiceDetails{}).Else(nil),
	}
}

func mapInvoiceAvailableActionDetailsToAPI(actions *billing.StandardInvoiceAvailableActionDetails) *apiv1models.InvoiceAvailableActionDetails {
	if actions == nil {
		return nil
	}

	return &apiv1models.InvoiceAvailableActionDetails{
		ResultingState: string(actions.ResultingState),
	}
}
