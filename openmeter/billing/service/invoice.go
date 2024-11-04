package billingservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/qmuntal/stateless"
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

		for i := range invoices.Items {
			invoices.Items[i], err = s.addInvoiceFields(ctx, invoices.Items[i])
			if err != nil {
				return billing.ListInvoicesResponse{}, fmt.Errorf("error adding fields to invoice [%s]: %w", invoices.Items[i].ID, err)
			}
		}

		return invoices, nil
	})
}

func (s *Service) addInvoiceFields(ctx context.Context, invoice billingentity.Invoice) (billingentity.Invoice, error) {
	if invoice.ExpandedFields.WorkflowApps {
		resolvedApps, err := s.resolveApps(ctx, invoice.Namespace, invoice.Workflow.AppReferences)
		if err != nil {
			return invoice, fmt.Errorf("error resolving apps for invoice [%s]: %w", invoice.ID, err)
		}

		invoice.Workflow.Apps = &billingentity.ProfileApps{
			Tax:       resolvedApps.Tax.App,
			Invoicing: resolvedApps.Invoicing.App,
			Payment:   resolvedApps.Payment.App,
		}
	}

	// let's resolve the statatus details
	_, err := WithInvoiceStateMachine(ctx, invoice, s.invoiceCalculator, func(ctx context.Context, sm *InvoiceStateMachine) error {
		sd, err := sm.StatusDetails(ctx)
		if err != nil {
			return fmt.Errorf("error resolving status details: %w", err)
		}

		invoice.StatusDetails = sd
		return nil
	})
	if err != nil {
		return invoice, fmt.Errorf("error resolving status details for invoice [%s]: %w", invoice.ID, err)
	}

	return invoice, nil
}

func (s *Service) GetInvoiceByID(ctx context.Context, input billing.GetInvoiceByIdInput) (billingentity.Invoice, error) {
	return entutils.TransactingRepo(ctx, s.adapter, func(ctx context.Context, txAdapter billing.Adapter) (billingentity.Invoice, error) {
		invoice, err := txAdapter.GetInvoiceById(ctx, input)
		if err != nil {
			return billingentity.Invoice{}, err
		}

		invoice, err = s.addInvoiceFields(ctx, invoice)
		if err != nil {
			return billingentity.Invoice{}, fmt.Errorf("error adding fields to invoice [%s]: %w", invoice.ID, err)
		}
		return invoice, nil
	})
}

func (s *Service) CreateInvoice(ctx context.Context, input billing.CreateInvoiceInput) ([]billingentity.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, billingentity.ValidationError{
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

			if len(sourceInvoiceIDs) == 0 {
				return nil, billingentity.ValidationError{
					Err: fmt.Errorf("no source lines found"),
				}
			}

			// let's lock the source gathering invoices, so that no other invoice operation can interfere
			err = txAdapter.LockInvoicesForUpdate(ctx, billing.LockInvoicesForUpdateInput{
				Namespace:  input.Customer.Namespace,
				InvoiceIDs: sourceInvoiceIDs,
			})
			if err != nil {
				return nil, fmt.Errorf("locking gathering invoices: %w", err)
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
					Status:   billingentity.InvoiceStatusDraftCreated,

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
				invoice, err := s.withLockedInvoiceStateMachine(ctx, txAdapter, invoiceID, func(ctx context.Context, sm *InvoiceStateMachine) error {
					err := sm.AdvanceUntilStateStable(ctx)
					if err != nil {
						return fmt.Errorf("activating invoice: %w", err)
					}

					err = txAdapter.UpdateInvoice(ctx, sm.Invoice)
					if err != nil {
						return fmt.Errorf("updating invoice: %w", err)
					}

					return nil
				})
				if err != nil {
					return nil, fmt.Errorf("advancing invoice: %w", err)
				}

				out = append(out, invoice)
			}
			return out, nil
		})
}

func (s *Service) gatherInscopeLines(ctx context.Context, input billing.CreateInvoiceInput, txAdapter billing.Adapter, asOf time.Time) ([]billingentity.Line, error) {
	if input.IncludePendingLines != nil {
		inScopeLines, err := txAdapter.ListInvoiceLines(ctx,
			billing.ListInvoiceLinesAdapterInput{
				Namespace:  input.Customer.Namespace,
				CustomerID: input.Customer.ID,

				LineIDs: input.IncludePendingLines,
			})
		if err != nil {
			return nil, fmt.Errorf("resolving in scope lines: %w", err)
		}

		// output validation

		// asOf validity
		for _, line := range inScopeLines {
			if line.InvoiceAt.After(asOf) {
				return nil, billingentity.ValidationError{
					Err: fmt.Errorf("line [%s] has invoiceAt [%s] after asOf [%s]", line.ID, line.InvoiceAt, asOf),
				}
			}
		}

		// all lines must be found
		if len(inScopeLines) != len(input.IncludePendingLines) {
			includedLines := lo.Map(inScopeLines, func(l billingentity.Line, _ int) string {
				return l.ID
			})

			missingIDs := lo.Without(input.IncludePendingLines, includedLines...)

			return nil, billingentity.NotFoundError{
				ID:     strings.Join(missingIDs, ","),
				Entity: billingentity.EntityInvoiceLine,
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
		return nil, billingentity.ValidationError{
			Err: fmt.Errorf("no pending lines found"),
		}
	}

	return lines, nil
}

func (s *Service) withLockedInvoiceStateMachine(
	ctx context.Context,
	txAdapter billing.Adapter,
	invoiceID billingentity.InvoiceID,
	cb InvoiceStateMachineCallback,
) (billingentity.Invoice, error) {
	// let's lock the invoice for update, we are using the dedicated call, so that
	// edges won't end up having SELECT FOR UPDATE locks
	if err := txAdapter.LockInvoicesForUpdate(ctx, billing.LockInvoicesForUpdateInput{
		Namespace:  invoiceID.Namespace,
		InvoiceIDs: []string{invoiceID.ID},
	}); err != nil {
		return billingentity.Invoice{}, fmt.Errorf("locking invoice: %w", err)
	}

	invoice, err := s.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: invoiceID,
		Expand:  billingentity.InvoiceExpandAll,
	})
	if err != nil {
		return billingentity.Invoice{}, fmt.Errorf("fetching invoice: %w", err)
	}

	return WithInvoiceStateMachine(ctx, invoice, s.invoiceCalculator, cb)
}

func (s *Service) AdvanceInvoice(ctx context.Context, input billing.AdvanceInvoiceInput) (billingentity.Invoice, error) {
	if err := input.Validate(); err != nil {
		return billingentity.Invoice{}, billingentity.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, s.adapter, func(ctx context.Context, txAdapter billing.Adapter) (billingentity.Invoice, error) {
		invoice, err := s.withLockedInvoiceStateMachine(ctx, txAdapter, input, func(ctx context.Context, sm *InvoiceStateMachine) error {
			preActivationStatus := sm.Invoice.Status

			canAdvance, err := sm.CanFire(ctx, triggerNext)
			if err != nil {
				return fmt.Errorf("checking if can advance: %w", err)
			}

			if !canAdvance {
				return billingentity.ValidationError{
					Err: fmt.Errorf("cannot advance invoice in status [%s]: %w", sm.Invoice.Status, billingentity.ErrInvoiceCannotAdvance),
				}
			}

			validationIssues, err := billingentity.ToValidationIssues(
				sm.AdvanceUntilStateStable(ctx),
			)
			if err != nil {
				return fmt.Errorf("advancing invoice: %w", err)
			}

			sm.Invoice.ValidationIssues = validationIssues
			s.logger.Info("invoice advanced", "invoice", input.ID, "from", preActivationStatus, "to", sm.Invoice.Status)

			return nil
		})
		if err != nil {
			return billingentity.Invoice{}, err
		}

		// Given the amount of state transitions, we are only saving the invoice after the whole chain
		// this means that some of the intermittent states will not be persisted in the DB.
		if err := txAdapter.UpdateInvoice(ctx, invoice); err != nil {
			return billingentity.Invoice{}, fmt.Errorf("updating invoice: %w", err)
		}

		return invoice, nil
	})
}

func (s *Service) ApproveInvoice(ctx context.Context, input billing.ApproveInvoiceInput) (billingentity.Invoice, error) {
	return s.executeTriggerOnInvoice(ctx, input, triggerApprove)
}

func (s *Service) RetryInvoice(ctx context.Context, input billing.RetryInvoiceInput) (billingentity.Invoice, error) {
	return s.executeTriggerOnInvoice(ctx, input, triggerRetry)
}

func (s *Service) executeTriggerOnInvoice(ctx context.Context, invoiceID billingentity.InvoiceID, trigger stateless.Trigger) (billingentity.Invoice, error) {
	if err := invoiceID.Validate(); err != nil {
		return billingentity.Invoice{}, billingentity.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, s.adapter, func(ctx context.Context, txAdapter billing.Adapter) (billingentity.Invoice, error) {
		invoice, err := s.withLockedInvoiceStateMachine(ctx, txAdapter, invoiceID, func(ctx context.Context, sm *InvoiceStateMachine) error {
			canFire, err := sm.CanFire(ctx, trigger)
			if err != nil {
				return fmt.Errorf("checking if can fire: %w", err)
			}

			if !canFire {
				return billingentity.ValidationError{
					Err: fmt.Errorf("cannot %s invoice in status [%s]: %w", trigger, sm.Invoice.Status, billingentity.ErrInvoiceActionNotAvailable),
				}
			}

			if err := sm.FireAndActivate(ctx, trigger); err != nil {
				return fmt.Errorf("firing %s: %w", trigger, err)
			}

			if err := txAdapter.UpdateInvoice(ctx, sm.Invoice); err != nil {
				return fmt.Errorf("updating invoice: %w", err)
			}

			return nil
		})
		if err != nil {
			return invoice, err
		}

		return invoice, nil
	})
}
