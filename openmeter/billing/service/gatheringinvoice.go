package billingservice

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ billing.GatheringInvoiceService = (*Service)(nil)

func (s *Service) ListGatheringInvoices(ctx context.Context, input billing.ListGatheringInvoicesInput) (pagination.Result[billing.GatheringInvoice], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[billing.GatheringInvoice]{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (pagination.Result[billing.GatheringInvoice], error) {
		return s.adapter.ListGatheringInvoices(ctx, input)
	})
}

func (s *Service) emulateStandardInvoicesGatheringInvoiceFields(ctx context.Context, invoices []billing.StandardInvoice) ([]billing.StandardInvoice, error) {
	mergedProfiles := make(map[customer.CustomerID]billing.CustomerOverrideWithDetails)

	for idx := range invoices {
		invoice := &invoices[idx]

		if invoice.Status != billing.StandardInvoiceStatusGathering {
			continue
		}

		if _, ok := mergedProfiles[invoice.CustomerID()]; !ok {
			expand := billing.CustomerOverrideExpand{
				Customer: true,
				Apps:     true,
			}

			mergedProfile, err := s.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
				Customer: invoice.CustomerID(),
				Expand:   expand,
			})
			if err != nil {
				return nil, err
			}

			mergedProfiles[invoice.CustomerID()] = mergedProfile
		}

		mergedProfile := mergedProfiles[invoice.CustomerID()]

		invoice.Customer = billing.InvoiceCustomer{
			CustomerID:       invoice.CustomerID().ID,
			Name:             mergedProfile.Customer.Name,
			Key:              mergedProfile.Customer.Key,
			UsageAttribution: lo.ToPtr(mergedProfile.Customer.GetUsageAttribution()),
		}

		invoice.Supplier = mergedProfile.MergedProfile.Supplier

		invoice.Workflow = billing.InvoiceWorkflow{
			AppReferences:          lo.FromPtr(mergedProfile.MergedProfile.AppReferences),
			Apps:                   mergedProfile.MergedProfile.Apps,
			SourceBillingProfileID: mergedProfile.MergedProfile.ID,
			Config:                 mergedProfile.MergedProfile.WorkflowConfig,
		}
	}

	return invoices, nil
}

func (s *Service) emulateStandardInvoiceGatheringInvoiceFields(ctx context.Context, invoice billing.StandardInvoice) (billing.StandardInvoice, error) {
	invoices, err := s.emulateStandardInvoicesGatheringInvoiceFields(ctx, []billing.StandardInvoice{invoice})
	if err != nil {
		return billing.StandardInvoice{}, err
	}

	if len(invoices) != 1 {
		return billing.StandardInvoice{}, fmt.Errorf("expected 1 invoice, got %d", len(invoices))
	}

	return invoices[0], nil
}
