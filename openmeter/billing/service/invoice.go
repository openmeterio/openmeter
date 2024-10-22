package billingservice

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ billing.InvoiceService = (*Service)(nil)

func (s *Service) ListInvoices(ctx context.Context, input billing.ListInvoicesInput) (billing.ListInvoicesResponse, error) {
	return entutils.TransactingRepo(ctx, s.adapter, func(ctx context.Context, txAdapter billing.Adapter) (billing.ListInvoicesResponse, error) {
		invoices, err := s.adapter.ListInvoices(ctx, input)
		if err != nil {
			return billing.ListInvoicesResponse{}, err
		}

		if input.Expand.WorkflowApps {
			for i := range invoices.Items {
				invoice := &invoices.Items[i]
				resolvedApps, err := s.resolveApps(ctx, input.Namespace, invoice.Workflow.AppReferences)
				if err != nil {
					return billing.ListInvoicesResponse{}, fmt.Errorf("error resolving apps for invoice [%s]: %w", invoice.ID, err)
				}

				invoice.Workflow.Apps = &billingentity.ProfileApps{
					Tax:       resolvedApps.Tax.App,
					Invoicing: resolvedApps.Invoicing.App,
					Payment:   resolvedApps.Payment.App,
				}
			}
		}

		return invoices, nil
	})
}
