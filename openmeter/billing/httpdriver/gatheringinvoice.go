package httpdriver

import (
	"fmt"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	"github.com/samber/lo"
)

func MapGatheringInvoiceToAPI(invoice billing.GatheringInvoice, customer *customer.Customer, profile billing.Profile) (api.Invoice, error) {
	var err error

	invoiceCustomer := billing.InvoiceCustomer{
		Key:  customer.Key,
		Name: customer.Name,
	}

	statusDetails := api.InvoiceStatusDetails{
		Failed:         false,
		Immutable:      false,
		ExtendedStatus: string(billing.StandardInvoiceStatusGathering),
	}

	if invoice.AvailableActions != nil && invoice.AvailableActions.CanBeInvoiced {
		statusDetails.AvailableActions = api.InvoiceAvailableActions{
			Invoice: &api.InvoiceAvailableActionInvoiceDetails{},
		}
	}

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
		Customer: mapInvoiceCustomerToAPI(invoiceCustomer),

		Number:      invoice.Number,
		Description: invoice.Description,
		Metadata:    convert.MapToPointer(invoice.Metadata),

		Status:        api.InvoiceStatus(billing.StandardInvoiceStatusGathering),
		StatusDetails: statusDetails,
		Supplier:      api.BillingParty{},
		Totals:        api.InvoiceTotals{},
		Type:          api.InvoiceType(billing.StandardInvoiceStatusCategoryGathering),
	}

	workflowConfig, err := mapWorkflowConfigSettingsToAPI(profile.WorkflowConfig)
	if err != nil {
		return api.Invoice{}, fmt.Errorf("failed to map workflow config to API: %w", err)
	}

	out.Workflow = api.InvoiceWorkflowSettings{
		SourceBillingProfileId: profile.ID,
		Workflow:               workflowConfig,
	}

	outLines, err := slicesx.MapWithErr(invoice.Lines.OrEmpty(), func(line billing.GatheringLine) (api.InvoiceLine, error) {
		mappedLine, err := mapGatheringInvoiceLineToAPI(line)
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

func mapGatheringInvoiceLineToAPI(line billing.GatheringLine) (api.InvoiceLine, error) {
	price, err := productcataloghttp.FromRateCardUsageBasedPrice(line.Price)
	if err != nil {
		return api.InvoiceLine{}, fmt.Errorf("failed to map price: %w", err)
	}

	invoiceLine := api.InvoiceLine{
		Type: api.InvoiceLineTypeUsageBased,
		Id:   line.ID,

		CreatedAt: line.CreatedAt,
		DeletedAt: line.DeletedAt,
		UpdatedAt: line.UpdatedAt,
		InvoiceAt: line.InvoiceAt,

		// TODO: deprecation
		Currency: string(line.Currency),
		Status:   api.InvoiceLineStatusValid,

		Description: line.Description,
		Name:        line.Name,
		ManagedBy:   api.InvoiceLineManagedBy(line.ManagedBy),

		// TODO: deprecation
		Invoice: &api.InvoiceReference{
			Id: line.InvoiceID,
		},

		Metadata: convert.MapToPointer(line.Metadata),
		Period: api.Period{
			From: line.ServicePeriod.From,
			To:   line.ServicePeriod.To,
		},

		TaxConfig: mapTaxConfigToAPI(line.TaxConfig),

		FeatureKey: lo.EmptyableToPtr(line.FeatureKey),

		Price: lo.ToPtr(price),

		RateCard: &api.InvoiceUsageBasedRateCard{
			TaxConfig:  mapTaxConfigToAPI(line.TaxConfig),
			Price:      lo.ToPtr(price),
			FeatureKey: lo.EmptyableToPtr(line.FeatureKey),
		},

		Subscription: mapSubscriptionReferencesToAPI(line.Subscription),
	}

	return invoiceLine, nil
}
