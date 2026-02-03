package httpdriver

import (
	"fmt"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	"github.com/samber/lo"
)

func MapGatheringInvoiceToAPI(invoice billing.GatheringInvoice) (api.GatheringInvoice, error) {
	var apps *api.BillingProfileAppsOrReference
	var err error

	// Sort the lines to make the response more consistent (internally we don't care about the order)
	invoice.SortLines()

	out := api.Invoice{
		Id: invoice.ID,

		CreatedAt:    invoice.CreatedAt,
		UpdatedAt:    invoice.UpdatedAt,
		DeletedAt:    invoice.DeletedAt,
		CollectionAt: lo.ToPtr(invoice.NextCollectionAt),
		Period:       mapServicePeriodToAPI(invoice.ServicePeriod),

		Currency: string(invoice.Currency),
		Customer: mapInvoiceCustomerToAPI(invoice.Customer),

		Number:      invoice.Number,
		Description: invoice.Description,
		Metadata:    convert.MapToPointer(invoice.Metadata),

		Status: api.InvoiceStatus(invoice.Status.ShortStatus()),
		// TODO: implement checking if the gathering invoice is invoicable now
		StatusDetails: api.InvoiceStatusDetails{
			Failed:         invoice.StatusDetails.Failed,
			Immutable:      invoice.StatusDetails.Immutable,
			ExtendedStatus: string(invoice.Status),

			AvailableActions: mapInvoiceAvailableActionsToAPI(invoice.StatusDetails.AvailableActions),
		},
		Supplier: api.BillingParty{},
		Totals:   api.InvoiceTotals{},
		Type:     api.InvoiceType(billing.StandardInvoiceStatusCategoryGathering),
	}

	workflowConfig, err := mapWorkflowConfigSettingsToAPI(invoice.Workflow.Config)
	if err != nil {
		return api.Invoice{}, fmt.Errorf("failed to map workflow config to API: %w", err)
	}

	out.Workflow = api.InvoiceWorkflowSettings{
		Apps:                   apps,
		SourceBillingProfileId: invoice.Workflow.SourceBillingProfileID,
		Workflow:               workflowConfig,
	}

	outLines, err := slicesx.MapWithErr(invoice.Lines.OrEmpty(), func(line *billing.StandardLine) (api.InvoiceLine, error) {
		mappedLine, err := mapInvoiceLineToAPI(line)
		if err != nil {
			return api.InvoiceLine{}, fmt.Errorf("failed to map billing line[%s] to API: %w", line.ID, err)
		}

		return mappedLine, nil
	})
	if err != nil {
		return api.Invoice{}, err
	}

	if len(outLines) > 0 {
		out.Lines = &outLines
	}

	return out, nil
}

func mapServicePeriodToAPI(p timeutil.ClosedPeriod) *api.Period {
	if lo.IsEmpty(p) {
		return nil
	}

	return &api.Period{
		From: p.From,
		To:   p.To,
	}
}
