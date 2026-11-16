package httpdriver

import (
	"fmt"

	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingconvert "github.com/openmeterio/openmeter/openmeter/billing/apiconvert"
	"github.com/openmeterio/openmeter/openmeter/customer"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func MapGatheringInvoiceToAPI(invoice billing.GatheringInvoice, customer *customer.Customer, profile billing.Profile) (apiv1models.Invoice, error) {
	var err error

	invoiceCustomer := billing.InvoiceCustomer{
		Key:  customer.Key,
		Name: customer.Name,
	}

	statusDetails := apiv1models.InvoiceStatusDetails{
		Failed:         false,
		Immutable:      false,
		ExtendedStatus: string(billing.StandardInvoiceStatusGathering),
	}

	if invoice.AvailableActions != nil && invoice.AvailableActions.CanBeInvoiced {
		statusDetails.AvailableActions = apiv1models.InvoiceAvailableActions{
			Invoice: &apiv1models.InvoiceAvailableActionInvoiceDetails{},
		}
	}

	// Sort the lines to make the response more consistent (internally we don't care about the order)
	invoice.SortLines()

	out := apiv1models.Invoice{
		Id: invoice.ID,

		CreatedAt:    invoice.CreatedAt,
		UpdatedAt:    invoice.UpdatedAt,
		DeletedAt:    invoice.DeletedAt,
		CollectionAt: invoice.NextCollectionAt,
		Period:       mapServicePeriodToAPI(invoice.ServicePeriod),

		Currency: string(invoice.Currency),
		Customer: billingconvert.ToAPIBillingInvoiceCustomerExtendedDetails(invoiceCustomer),

		Number:      invoice.Number,
		Description: invoice.Description,
		Metadata:    convert.MapToPointer(invoice.Metadata),

		Status:        apiv1models.InvoiceStatus(billing.StandardInvoiceStatusGathering),
		StatusDetails: statusDetails,
		Supplier:      apiv1models.BillingParty{},
		Totals:        apiv1models.InvoiceTotals{},
		Type:          apiv1models.InvoiceType(billing.StandardInvoiceStatusCategoryGathering),
	}

	workflowConfig, err := billingconvert.ToAPIBillingWorkflow(profile.WorkflowConfig)
	if err != nil {
		return apiv1models.Invoice{}, fmt.Errorf("failed to map workflow config to API: %w", err)
	}

	out.Workflow = apiv1models.InvoiceWorkflowSettings{
		SourceBillingProfileId: profile.ID,
		Workflow:               workflowConfig,
	}

	outLines, err := slicesx.MapWithErr(invoice.Lines.OrEmpty(), func(line billing.GatheringLine) (apiv1models.InvoiceLine, error) {
		mappedLine, err := mapGatheringInvoiceLineToAPI(line)
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

func mapServicePeriodToAPI(p timeutil.ClosedPeriod) *apiv1models.Period {
	if lo.IsEmpty(p) {
		return nil
	}

	return &apiv1models.Period{
		From: p.From,
		To:   p.To,
	}
}

func mapGatheringInvoiceLineToAPI(line billing.GatheringLine) (apiv1models.InvoiceLine, error) {
	price, err := productcataloghttp.FromRateCardUsageBasedPrice(line.Price)
	if err != nil {
		return apiv1models.InvoiceLine{}, fmt.Errorf("failed to map price: %w", err)
	}

	invoiceLine := apiv1models.InvoiceLine{
		Type: apiv1models.InvoiceLineTypeUsageBased,
		Id:   line.ID,

		CreatedAt: line.CreatedAt,
		DeletedAt: line.DeletedAt,
		UpdatedAt: line.UpdatedAt,
		InvoiceAt: line.InvoiceAt,

		Currency: string(line.Currency),
		Status:   apiv1models.InvoiceLineStatusValid,

		Description: line.Description,
		Name:        line.Name,
		ManagedBy:   apiv1models.InvoiceLineManagedBy(line.ManagedBy),

		Invoice: &apiv1models.InvoiceReference{
			Id: line.InvoiceID,
		},

		Metadata: convert.MapToPointer(line.Metadata),
		Period: apiv1models.Period{
			From: line.ServicePeriod.From,
			To:   line.ServicePeriod.To,
		},

		TaxConfig: billingconvert.ToAPITaxConfig(line.TaxConfig),

		FeatureKey: lo.EmptyableToPtr(line.FeatureKey),

		Price: lo.ToPtr(price),

		RateCard: &apiv1models.InvoiceUsageBasedRateCard{
			TaxConfig:  billingconvert.ToAPITaxConfig(line.TaxConfig),
			Price:      lo.ToPtr(price),
			FeatureKey: lo.EmptyableToPtr(line.FeatureKey),
		},

		Subscription: billingconvert.ToAPIInvoiceLineSubscriptionReference(line.Subscription),
	}

	return invoiceLine, nil
}
