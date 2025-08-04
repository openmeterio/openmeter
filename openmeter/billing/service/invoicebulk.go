package billingservice

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *Service) BulkUpdateInvoices(ctx context.Context, input billing.BulkUpdateInvoicesInput) (billing.BulkUpdateInvoicesResult, error) {
	if err := input.Validate(); err != nil {
		return billing.BulkUpdateInvoicesResult{}, billing.ValidationError{
			Err: err,
		}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billing.BulkUpdateInvoicesResult, error) {
		invoiceToCustomerID, err := s.adapter.GetInvoiceOwnership(ctx, billing.GetInvoiceOwnershipAdapterInput{
			InvoiceIDs: input.Invoices,
		})
		if err != nil {
			return billing.BulkUpdateInvoicesResult{}, fmt.Errorf("getting invoice ownership: %w", err)
		}

		invoiceIDsByCustomerID := map[customer.CustomerID][]billing.InvoiceID{}
		for invoiceID, customerID := range invoiceToCustomerID {
			invoiceIDsByCustomerID[customerID] = append(invoiceIDsByCustomerID[customerID], invoiceID)
		}

		for customerID, invoiceIDs := range invoiceIDsByCustomerID {
			xxx, err := transcationForInvoiceManipulation(ctx, s, customerID, func(ctx context.Context) (billing.BulkUpdateInvoicesResult, error) {
				invoices := make([]*billing.Invoice, 0, len(invoiceIDs))
				for _, invoiceID := range invoiceIDs {
					invoice, err := s.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
						Invoice: invoiceID,
						Expand: billing.InvoiceExpandAll.
							SetDeletedLines(input.IncludeDeletedLines),
					})
					if err != nil {
						return billing.BulkUpdateInvoicesResult{}, fmt.Errorf("getting invoice[%s]: %w", invoiceID.ID, err)
					}

					invoices = append(invoices, &invoice)
				}
			})
			if err != nil {
				return billing.BulkUpdateInvoicesResult{}, fmt.Errorf("updating invoices: %w", err)
			}
		}
	})
}
