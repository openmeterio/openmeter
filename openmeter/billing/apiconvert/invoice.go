package apiconvert

import (
	"fmt"

	"github.com/samber/lo"

	apimodels "github.com/openmeterio/openmeter/api/models"
	appconvert "github.com/openmeterio/openmeter/openmeter/app/apiconvert"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/externalid"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func ToAPIStandardInvoice(invoice billing.StandardInvoice) (apimodels.Invoice, error) {
	var apps *apimodels.BillingProfileAppsOrReference
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
			return apimodels.Invoice{}, fmt.Errorf("failed to map profile apps to API: %w", err)
		}
	} else {
		apps, err = ToAPIBillingProfileAppReferences(&invoice.Workflow.AppReferences)
		if err != nil {
			return apimodels.Invoice{}, fmt.Errorf("failed to map profile app references to API: %w", err)
		}
	}

	out := apimodels.Invoice{
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

		Status: apimodels.InvoiceStatus(invoice.Status.ShortStatus()),
		StatusDetails: apimodels.InvoiceStatusDetails{
			Failed:         invoice.StatusDetails.Failed,
			Immutable:      invoice.StatusDetails.Immutable,
			ExtendedStatus: string(invoice.Status),

			AvailableActions: mapInvoiceAvailableActionsToAPI(invoice.StatusDetails.AvailableActions),
		},
		Supplier: ToAPIBillingParty(invoice.Supplier),
		Totals:   mapTotalsToAPI(invoice.Totals),
		// TODO[OM-943]: Implement
		Payment: nil,
		Type:    apimodels.InvoiceType(invoice.Type),
		ValidationIssues: convert.SliceToPointer(
			lo.Map(invoice.ValidationIssues, func(v billing.ValidationIssue, _ int) apimodels.ValidationIssue {
				return apimodels.ValidationIssue{
					Id:        v.ID,
					CreatedAt: v.CreatedAt,
					UpdatedAt: v.UpdatedAt,
					DeletedAt: v.DeletedAt,

					Severity:  apimodels.ValidationIssueSeverity(v.Severity),
					Message:   v.Message,
					Code:      lo.EmptyableToPtr(v.Code),
					Component: string(v.Component),
					Field:     lo.EmptyableToPtr(v.Path),
				}
			})),
		ExternalIds: mapInvoiceAppExternalIdsToAPI(invoice.ExternalIDs),
	}

	workflowConfig, err := ToAPIBillingWorkflow(invoice.Workflow.Config)
	if err != nil {
		return apimodels.Invoice{}, fmt.Errorf("failed to map workflow config to API: %w", err)
	}

	out.Workflow = apimodels.InvoiceWorkflowSettings{
		Apps:                   apps,
		SourceBillingProfileId: invoice.Workflow.SourceBillingProfileID,
		Workflow:               workflowConfig,
	}

	outLines, err := slicesx.MapWithErr(invoice.Lines.OrEmpty(), func(line *billing.StandardLine) (apimodels.InvoiceLine, error) {
		mappedLine, err := mapInvoiceLineToAPI(line)
		if err != nil {
			return apimodels.InvoiceLine{}, fmt.Errorf("failed to map billing line[%s] to API: %w", line.ID, err)
		}

		return mappedLine, nil
	})
	if err != nil {
		return apimodels.Invoice{}, err
	}

	if len(outLines) > 0 {
		out.Lines = &outLines
	}

	return out, nil
}

func ToAPIEventInvoice(event billing.EventStandardInvoice) (apimodels.Invoice, error) {
	// Only the apps captured in the event belong in the notification snapshot.
	event.Invoice.Workflow.Apps = nil

	invoice, err := ToAPIStandardInvoice(event.Invoice)
	if err != nil {
		return apimodels.Invoice{}, err
	}

	// Even an event without apps has the expanded apps shape in stored v1 payloads.
	apps := apimodels.BillingProfileApps{}

	if event.Apps.Invoicing.Type != "" {
		apps.Invoicing, err = appconvert.ToAPIEventApp(event.Apps.Invoicing)
		if err != nil {
			return apimodels.Invoice{}, err
		}
	}

	if event.Apps.Payment.Type != "" {
		apps.Payment, err = appconvert.ToAPIEventApp(event.Apps.Payment)
		if err != nil {
			return apimodels.Invoice{}, err
		}
	}

	if event.Apps.Tax.Type != "" {
		apps.Tax, err = appconvert.ToAPIEventApp(event.Apps.Tax)
		if err != nil {
			return apimodels.Invoice{}, err
		}
	}

	invoice.Workflow.Apps = &apimodels.BillingProfileAppsOrReference{}
	if err := invoice.Workflow.Apps.FromBillingProfileApps(apps); err != nil {
		return apimodels.Invoice{}, err
	}

	return invoice, nil
}

func ToAPIBillingInvoiceCustomerExtendedDetails(c billing.InvoiceCustomer) apimodels.BillingInvoiceCustomerExtendedDetails {
	a := c.BillingAddress

	out := apimodels.BillingInvoiceCustomerExtendedDetails{
		Id:   lo.ToPtr(c.CustomerID),
		Key:  c.Key,
		Name: lo.EmptyableToPtr(c.Name),
	}

	if c.UsageAttribution != nil {
		out.UsageAttribution = apimodels.CustomerUsageAttribution{
			SubjectKeys: c.UsageAttribution.SubjectKeys,
		}
	}

	if a != nil && !lo.IsEmpty(*a) {
		out.Addresses = lo.ToPtr([]apimodels.Address{
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

func mapInvoiceAppExternalIdsToAPI(externalIds externalid.InvoiceExternalIDs) *apimodels.InvoiceAppExternalIds {
	if lo.IsEmpty(externalIds) {
		return nil
	}

	return &apimodels.InvoiceAppExternalIds{
		Invoicing: lo.EmptyableToPtr(externalIds.Invoicing),
		Payment:   lo.EmptyableToPtr(externalIds.Payment),
	}
}

func mapPeriodToAPI(p *timeutil.ClosedPeriod) *apimodels.Period {
	if p == nil {
		return nil
	}

	// TODO[later]: let's use a common model for this
	return &apimodels.Period{
		From: p.From,
		To:   p.To,
	}
}

func mapTotalsToAPI(t totals.Totals) apimodels.InvoiceTotals {
	return apimodels.InvoiceTotals{
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

func mapInvoiceAvailableActionsToAPI(actions billing.StandardInvoiceAvailableActions) apimodels.InvoiceAvailableActions {
	return apimodels.InvoiceAvailableActions{
		Advance:            mapInvoiceAvailableActionDetailsToAPI(actions.Advance),
		Approve:            mapInvoiceAvailableActionDetailsToAPI(actions.Approve),
		Delete:             mapInvoiceAvailableActionDetailsToAPI(actions.Delete),
		Retry:              mapInvoiceAvailableActionDetailsToAPI(actions.Retry),
		Void:               mapInvoiceAvailableActionDetailsToAPI(actions.Void),
		SnapshotQuantities: mapInvoiceAvailableActionDetailsToAPI(actions.SnapshotQuantities),
		Invoice:            lo.If(actions.Invoice != nil, &apimodels.InvoiceAvailableActionInvoiceDetails{}).Else(nil),
	}
}

func mapInvoiceAvailableActionDetailsToAPI(actions *billing.StandardInvoiceAvailableActionDetails) *apimodels.InvoiceAvailableActionDetails {
	if actions == nil {
		return nil
	}

	return &apimodels.InvoiceAvailableActionDetails{
		ResultingState: string(actions.ResultingState),
	}
}
