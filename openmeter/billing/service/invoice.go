package billingservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
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

func (s *Service) GetInvoiceByID(ctx context.Context, input billing.GetInvoiceByIdInput) (billingentity.Invoice, error) {
	return entutils.TransactingRepo(ctx, s.adapter, func(ctx context.Context, txAdapter billing.Adapter) (billingentity.Invoice, error) {
		invoice, err := txAdapter.GetInvoiceById(ctx, input)
		if err != nil {
			return billingentity.Invoice{}, err
		}

		if input.Expand.WorkflowApps {
			resolvedApps, err := s.resolveApps(ctx, input.Invoice.Namespace, invoice.Workflow.AppReferences)
			if err != nil {
				return billingentity.Invoice{}, fmt.Errorf("error resolving apps for invoice [%s]: %w", invoice.ID, err)
			}

			invoice.Workflow.Apps = &billingentity.ProfileApps{
				Tax:       resolvedApps.Tax.App,
				Invoicing: resolvedApps.Invoicing.App,
				Payment:   resolvedApps.Payment.App,
			}
		}

		return invoice, nil
	})
}

func (s *Service) CreateInvoice(ctx context.Context, input billing.CreateInvoiceInput) ([]billingentity.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return TransactingRepoForGatheringInvoiceManipulation(
		ctx,
		s.adapter,
		input.Customer,
		func(ctx context.Context, txAdapter billing.Adapter) ([]billingentity.Invoice, error) {
			// let's resolve the customer's settings
			customerProfile, err := s.GetProfileWithCustomerOverride(ctx, billing.GetProfileWithCustomerOverrideInput{
				Namespace:  input.Customer.Namespace,
				CustomerID: input.Customer.ID,
			})
			if err != nil {
				return nil, fmt.Errorf("fetching customer profile: %w", err)
			}

			asof := lo.FromPtrOr(input.AsOf, clock.Now())

			// let's gather the in-scope lines and validate it
			inScopeLines, err := s.gatherInscopeLines(ctx, input, txAdapter, asof)
			if err != nil {
				return nil, err
			}

			sourceInvoiceIDs := lo.Uniq(lo.Map(inScopeLines, func(l billingentity.Line, _ int) string {
				return l.InvoiceID
			}))

			if len(sourceInvoiceIDs) > 0 {
				// let's lock the source gathering invoices, so that no other staging call can add items to them in
				// case we would need to delete them
				err = txAdapter.LockInvoicesForUpdate(ctx, billing.LockInvoicesForUpdateInput{
					Namespace:  input.Customer.Namespace,
					InvoiceIDs: sourceInvoiceIDs,
				})
				if err != nil {
					return nil, fmt.Errorf("locking gathering invoices: %w", err)
				}
			}

			linesByCurrency := lo.GroupBy(inScopeLines, func(line billingentity.Line) currencyx.Code {
				return line.Currency
			})

			createdInvoices := make([]billingentity.InvoiceID, 0, len(linesByCurrency))

			for currency, lines := range linesByCurrency {
				// let's create the invoice
				invoice, err := txAdapter.CreateInvoice(ctx, billing.CreateInvoiceAdapterInput{
					Namespace: input.Customer.Namespace,
					Customer:  customerProfile.Customer,
					Profile:   customerProfile.Profile,

					Currency: currency,
					Status:   billingentity.InvoiceStatusDraft,

					Type: billingentity.InvoiceTypeStandard,
				})
				if err != nil {
					return nil, fmt.Errorf("creating invoice: %w", err)
				}

				createdInvoices = append(createdInvoices, billingentity.InvoiceID{
					Namespace: invoice.Namespace,
					ID:        invoice.ID,
				})

				// let's associate the invoice lines to the invoice
				err = s.associateLinesToInvoice(ctx, txAdapter, invoice, lines)
				if err != nil {
					return nil, fmt.Errorf("associating lines to invoice: %w", err)
				}
			}

			// Let's check if we need to remove any empty gathering invoices (e.g. if they don't have any line items)
			// This typically should happen when a subscription has ended.

			invoiceLineCounts, err := txAdapter.AssociatedLineCounts(ctx, billing.AssociatedLineCountsAdapterInput{
				Namespace:  input.Customer.Namespace,
				InvoiceIDs: sourceInvoiceIDs,
			})
			if err != nil {
				return nil, fmt.Errorf("cleanup: line counts check: %w", err)
			}

			invoicesWithoutLines := lo.Filter(sourceInvoiceIDs, func(id string, _ int) bool {
				return invoiceLineCounts.Counts[billingentity.InvoiceID{
					Namespace: input.Customer.Namespace,
					ID:        id,
				}] == 0
			})

			if len(invoicesWithoutLines) > 0 {
				err = txAdapter.DeleteInvoices(ctx, billing.DeleteInvoicesAdapterInput{
					Namespace:  input.Customer.Namespace,
					InvoiceIDs: invoicesWithoutLines,
				})
				if err != nil {
					return nil, fmt.Errorf("cleanup invoices: %w", err)
				}
			}

			// Assemble output: we need to refetch as the association call will have side-effects of updating
			// invoice objects (e.g. totals, period, etc.)
			out := make([]billingentity.Invoice, 0, len(createdInvoices))
			for _, invoiceID := range createdInvoices {
				invoiceWithLines, err := s.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
					Invoice: invoiceID,
					Expand:  billing.InvoiceExpandAll,
				})
				if err != nil {
					return nil, fmt.Errorf("cannot get invoice[%s]: %w", invoiceWithLines.ID, err)
				}

				out = append(out, invoiceWithLines)
			}
			return out, nil
		})
}

func (s *Service) gatherInscopeLines(ctx context.Context, input billing.CreateInvoiceInput, txAdapter billing.Adapter, asOf time.Time) ([]billingentity.Line, error) {
	if input.IncludePendingLines != nil {
		if len(*input.IncludePendingLines) == 0 {
			// We would like to create an empty invoice
			return []billingentity.Line{}, nil
		}

		inScopeLines, err := txAdapter.ListInvoiceLines(ctx,
			billing.ListInvoiceLinesAdapterInput{
				Namespace:  input.Customer.Namespace,
				CustomerID: input.Customer.ID,

				LineIDs: *input.IncludePendingLines,
			})
		if err != nil {
			return nil, fmt.Errorf("resolving in scope lines: %w", err)
		}

		// output validation

		// asOf validity
		for _, line := range inScopeLines {
			if line.InvoiceAt.After(asOf) {
				return nil, billing.ValidationError{
					Err: fmt.Errorf("line [%s] has invoiceAt [%s] after asOf [%s]", line.ID, line.InvoiceAt, asOf),
				}
			}
		}

		// all lines must be found
		if len(inScopeLines) != len(*input.IncludePendingLines) {
			includedLines := lo.Map(inScopeLines, func(l billingentity.Line, _ int) string {
				return l.ID
			})

			missingIDs := lo.Without(*input.IncludePendingLines, includedLines...)

			return nil, billing.NotFoundError{
				ID:     strings.Join(missingIDs, ","),
				Entity: billing.EntityInvoiceLine,
				Err:    fmt.Errorf("some invoice lines are not found"),
			}
		}

		return inScopeLines, nil
	}

	lines, err := txAdapter.ListInvoiceLines(ctx,
		billing.ListInvoiceLinesAdapterInput{
			Namespace:  input.Customer.Namespace,
			CustomerID: input.Customer.ID,

			InvoiceStatuses: []billingentity.InvoiceStatus{
				billingentity.InvoiceStatusGathering,
			},

			InvoiceAtBefore: lo.ToPtr(asOf),
		})
	if err != nil {
		return nil, err
	}

	if len(lines) == 0 {
		// We haven't requested explicit empty invoice, so we should have some pending lines
		return nil, billing.ValidationError{
			Err: fmt.Errorf("no pending lines found"),
		}
	}

	return lines, nil
}
