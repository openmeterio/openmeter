package billingservice

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

var _ billing.InvoiceLineService = (*Service)(nil)

func (s *Service) CreateInvoiceLines(ctx context.Context, input billing.CreateInvoiceLinesInput) (*billing.CreateInvoiceLinesResponse, error) {
	if err := input.Validate(); err != nil {
		return nil, billingentity.ValidationError{
			Err: err,
		}
	}

	return TransactingRepoForGatheringInvoiceManipulation(
		ctx,
		s.adapter,
		customerentity.CustomerID{
			Namespace: input.Namespace,
			ID:        input.CustomerID,
		},
		func(ctx context.Context, txAdapter billing.Adapter) (*billing.CreateInvoiceLinesResponse, error) {
			// let's resolve the customer's settings
			customerProfile, err := s.GetProfileWithCustomerOverride(ctx, billing.GetProfileWithCustomerOverrideInput{
				Namespace:  input.Namespace,
				CustomerID: input.CustomerID,
			})
			if err != nil {
				return nil, fmt.Errorf("fetching customer profile: %w", err)
			}

			// TODO: we should optimize this as this does O(n) queries for invoices per line
			for i, line := range input.Lines {
				updatedLine, err := s.upsertLineInvoice(ctx, txAdapter, line, input, customerProfile)
				if err != nil {
					return nil, fmt.Errorf("upserting line[%d]: %w", i, err)
				}

				input.Lines[i] = updatedLine
			}

			// Create the invoice Lines
			lines, err := txAdapter.CreateInvoiceLines(ctx, billing.CreateInvoiceLinesAdapterInput{
				Namespace: input.Namespace,
				Lines:     input.Lines,
			})
			if err != nil {
				return nil, fmt.Errorf("creating invoice Line: %w", err)
			}

			return lines, nil
		})
}

func (s *Service) upsertLineInvoice(ctx context.Context, txAdapter billing.Adapter, line billingentity.Line, input billing.CreateInvoiceLinesInput, customerProfile *billingentity.ProfileWithCustomerDetails) (billingentity.Line, error) {
	// Let's set the default values for the line item
	line.Status = billingentity.InvoiceLineStatusValid

	if line.InvoiceID != "" {
		// We would want to attach the line to an existing invoice
		invoice, err := txAdapter.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
			Invoice: billingentity.InvoiceID{
				ID:        line.InvoiceID,
				Namespace: input.Namespace,
			},
		})
		if err != nil {
			return line, billingentity.ValidationError{
				Err: fmt.Errorf("fetching invoice [%s]: %w", line.InvoiceID, err),
			}
		}

		if !invoice.StatusDetails.Immutable {
			return line, billingentity.ValidationError{
				Err: fmt.Errorf("invoice [%s] is not mutable", line.InvoiceID),
			}
		}

		if invoice.Currency != line.Currency {
			return line, billingentity.ValidationError{
				Err: fmt.Errorf("currency mismatch: invoice [%s] currency is %s, but line currency is %s", line.InvoiceID, invoice.Currency, line.Currency),
			}
		}

		line.InvoiceID = invoice.ID
		return line, nil
	}

	// We would want to stage a pending invoice Line
	pendingInvoiceList, err := txAdapter.ListInvoices(ctx, billing.ListInvoicesInput{
		Page: pagination.Page{
			PageNumber: 1,
			PageSize:   10,
		},
		Customers:        []string{input.CustomerID},
		Namespace:        input.Namespace,
		ExtendedStatuses: []billingentity.InvoiceStatus{billingentity.InvoiceStatusGathering},
		Currencies:       []currencyx.Code{line.Currency},
		OrderBy:          api.BillingInvoiceOrderByCreatedAt,
		Order:            sortx.OrderAsc,
	})
	if err != nil {
		return line, fmt.Errorf("fetching gathering invoices: %w", err)
	}

	if len(pendingInvoiceList.Items) == 0 {
		// Create a new invoice
		invoice, err := txAdapter.CreateInvoice(ctx, billing.CreateInvoiceAdapterInput{
			Namespace: input.Namespace,
			Customer:  customerProfile.Customer,
			Profile:   customerProfile.Profile,
			Currency:  line.Currency,
			Status:    billingentity.InvoiceStatusGathering,
			Type:      billingentity.InvoiceTypeStandard,
		})
		if err != nil {
			return line, fmt.Errorf("creating invoice: %w", err)
		}

		line.InvoiceID = invoice.ID
	} else {
		// Attach to the first pending invoice
		line.InvoiceID = pendingInvoiceList.Items[0].ID

		if len(pendingInvoiceList.Items) > 1 {
			// Note: Given that we are not using serializable transactions (which is fine), we might
			// have multiple gathering invoices for the same customer.
			// This is a rare case, but we should log it at least, later we can implement a call that
			// merges these invoices (it's fine to just move the Lines to the first invoice)
			s.logger.Warn("more than one pending invoice found", "customer", input.CustomerID, "namespace", input.Namespace)
		}
	}

	return line, nil
}

func (s *Service) associateLinesToInvoice(ctx context.Context, txAdapter billing.Adapter, invoice billingentity.Invoice, lines []billingentity.Line) error {
	for _, line := range lines {
		if line.InvoiceID == invoice.ID {
			return billingentity.ValidationError{
				Err: fmt.Errorf("line[%s]: line already associated with invoice[%s]", line.ID, invoice.ID),
			}
		}
	}

	// Associate the lines to the invoice
	err := txAdapter.AssociateLinesToInvoice(ctx, billing.AssociateLinesToInvoiceAdapterInput{
		Invoice: billingentity.InvoiceID{
			ID:        invoice.ID,
			Namespace: invoice.Namespace,
		},

		LineIDs: lo.Map(lines, func(l billingentity.Line, _ int) string {
			return l.ID
		}),
	})
	if err != nil {
		return err
	}

	// TODO[later]: Here we need to recalculate any line specific fields for both invoices
	return nil
}
