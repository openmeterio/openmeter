package billingservice

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/qmuntal/stateless"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	lineservice "github.com/openmeterio/openmeter/openmeter/billing/service/lineservice"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

var _ billing.InvoiceService = (*Service)(nil)

func (s *Service) ListInvoices(ctx context.Context, input billing.ListInvoicesInput) (billing.ListInvoicesResponse, error) {
	invoices, err := s.adapter.ListInvoices(ctx, input)
	if err != nil {
		return billing.ListInvoicesResponse{}, err
	}

	for i := range invoices.Items {
		invoices.Items[i], err = s.resolveWorkflowApps(ctx, invoices.Items[i])
		if err != nil {
			return billing.ListInvoicesResponse{}, fmt.Errorf("error adding fields to invoice [%s]: %w", invoices.Items[i].ID, err)
		}

		invoices.Items[i], err = s.resolveStatusDetails(ctx, invoices.Items[i])
		if err != nil {
			return billing.ListInvoicesResponse{}, fmt.Errorf("error resolving status details for invoice [%s]: %w", invoices.Items[i].ID, err)
		}
	}

	return invoices, nil
}

func (s *Service) resolveWorkflowApps(ctx context.Context, invoice billingentity.Invoice) (billingentity.Invoice, error) {
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

	return invoice, nil
}

func (s *Service) resolveStatusDetails(ctx context.Context, invoice billingentity.Invoice) (billingentity.Invoice, error) {
	// let's resolve the statatus details
	_, err := s.WithInvoiceStateMachine(ctx, invoice, func(ctx context.Context, sm *InvoiceStateMachine) error {
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
	invoice, err := s.adapter.GetInvoiceById(ctx, input)
	if err != nil {
		return billingentity.Invoice{}, err
	}

	invoice, err = s.resolveWorkflowApps(ctx, invoice)
	if err != nil {
		return billingentity.Invoice{}, fmt.Errorf("error adding fields to invoice [%s]: %w", invoice.ID, err)
	}

	invoice, err = s.resolveStatusDetails(ctx, invoice)
	if err != nil {
		return billingentity.Invoice{}, fmt.Errorf("error resolving status details for invoice [%s]: %w", invoice.ID, err)
	}

	return invoice, nil
}

func (s *Service) CreateInvoice(ctx context.Context, input billing.CreateInvoiceInput) ([]billingentity.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, billingentity.ValidationError{
			Err: err,
		}
	}

	return TranscationForGatheringInvoiceManipulation(
		ctx,
		s,
		input.Customer,
		func(ctx context.Context) ([]billingentity.Invoice, error) {
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
			inScopeLines, err := s.gatherInscopeLines(ctx, input, asof)
			if err != nil {
				return nil, err
			}

			sourceInvoiceIDs := lo.Uniq(lo.Map(inScopeLines, func(l lineservice.LineWithBillablePeriod, _ int) string {
				return l.InvoiceID()
			}))

			if len(sourceInvoiceIDs) == 0 {
				return nil, billingentity.ValidationError{
					Err: fmt.Errorf("no source lines found"),
				}
			}

			// let's lock the source gathering invoices, so that no other invoice operation can interfere
			err = s.adapter.LockInvoicesForUpdate(ctx, billing.LockInvoicesForUpdateInput{
				Namespace:  input.Customer.Namespace,
				InvoiceIDs: sourceInvoiceIDs,
			})
			if err != nil {
				return nil, fmt.Errorf("locking gathering invoices: %w", err)
			}

			linesByCurrency := lo.GroupBy(inScopeLines, func(line lineservice.LineWithBillablePeriod) currencyx.Code {
				return line.Currency()
			})

			createdInvoices := make([]billingentity.InvoiceID, 0, len(linesByCurrency))

			for currency, lines := range linesByCurrency {
				// let's create the invoice
				invoice, err := s.adapter.CreateInvoice(ctx, billing.CreateInvoiceAdapterInput{
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
				invoice, err = s.associateLinesToInvoice(ctx, invoice, lines)
				if err != nil {
					return nil, fmt.Errorf("associating lines to invoice: %w", err)
				}

				// TODO[later]: we are saving here, and the state machine advancement will do a load/save later
				// this is something we could optimize in the future by adding the option to withLockedInvoiceStateMachine
				// to either pass in the invoice or the ID
				_, err = s.adapter.UpdateInvoice(ctx, invoice)
				if err != nil {
					return nil, fmt.Errorf("updating invoice: %w", err)
				}
			}

			// Let's check if we need to remove any empty gathering invoices (e.g. if they don't have any line items)
			// This typically should happen when a subscription has ended.

			invoiceLineCounts, err := s.adapter.AssociatedLineCounts(ctx, billing.AssociatedLineCountsAdapterInput{
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
				err = s.adapter.DeleteInvoices(ctx, billing.DeleteInvoicesAdapterInput{
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
				invoice, err := s.withLockedInvoiceStateMachine(ctx, invoiceID, func(ctx context.Context, sm *InvoiceStateMachine) error {
					validationIssues, err := billingentity.ToValidationIssues(
						sm.AdvanceUntilStateStable(ctx),
					)
					if err != nil {
						return fmt.Errorf("activating invoice: %w", err)
					}

					sm.Invoice.ValidationIssues = validationIssues

					sm.Invoice, err = s.adapter.UpdateInvoice(ctx, sm.Invoice)
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

func (s *Service) gatherInscopeLines(ctx context.Context, input billing.CreateInvoiceInput, asOf time.Time) ([]lineservice.LineWithBillablePeriod, error) {
	if input.IncludePendingLines.IsPresent() {
		lineIDs := input.IncludePendingLines.OrEmpty()

		if len(lineIDs) == 0 {
			return nil, billingentity.ValidationError{
				Err: billingentity.ErrInvoiceEmpty,
			}
		}

		inScopeLines, err := s.adapter.ListInvoiceLines(ctx,
			billing.ListInvoiceLinesAdapterInput{
				Namespace:  input.Customer.Namespace,
				CustomerID: input.Customer.ID,

				LineIDs: input.IncludePendingLines.OrEmpty(),
			})
		if err != nil {
			return nil, fmt.Errorf("resolving in scope lines: %w", err)
		}

		lines, err := s.lineService.FromEntities(inScopeLines)
		if err != nil {
			return nil, fmt.Errorf("creating line services: %w", err)
		}

		// output validation
		resolvedLines, err := lines.ResolveBillablePeriod(ctx, asOf)
		if err != nil {
			return nil, err
		}

		// all lines must be found
		if len(resolvedLines) != len(lineIDs) {
			includedLines := lo.Map(resolvedLines, func(l lineservice.LineWithBillablePeriod, _ int) string {
				return l.ID()
			})

			missingIDs := lo.Without(lineIDs, includedLines...)

			return nil, billingentity.NotFoundError{
				ID:     strings.Join(missingIDs, ","),
				Entity: billingentity.EntityInvoiceLine,
				Err:    billingentity.ErrInvoiceLinesNotBillable,
			}
		}

		return resolvedLines, nil
	}

	lines, err := s.adapter.ListInvoiceLines(ctx,
		billing.ListInvoiceLinesAdapterInput{
			Namespace:  input.Customer.Namespace,
			CustomerID: input.Customer.ID,

			InvoiceStatuses: []billingentity.InvoiceStatus{
				billingentity.InvoiceStatusGathering,
			},
			Statuses: []billingentity.InvoiceLineStatus{
				billingentity.InvoiceLineStatusValid,
			},
		})
	if err != nil {
		return nil, err
	}

	lineSrvs, err := s.lineService.FromEntities(lines)
	if err != nil {
		return nil, err
	}

	return lineSrvs.ResolveBillablePeriod(ctx, asOf)
}

func (s *Service) withLockedInvoiceStateMachine(
	ctx context.Context,
	invoiceID billingentity.InvoiceID,
	cb InvoiceStateMachineCallback,
) (billingentity.Invoice, error) {
	// let's lock the invoice for update, we are using the dedicated call, so that
	// edges won't end up having SELECT FOR UPDATE locks
	if err := s.adapter.LockInvoicesForUpdate(ctx, billing.LockInvoicesForUpdateInput{
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

	return s.WithInvoiceStateMachine(ctx, invoice, cb)
}

func (s *Service) AdvanceInvoice(ctx context.Context, input billing.AdvanceInvoiceInput) (billingentity.Invoice, error) {
	if err := input.Validate(); err != nil {
		return billingentity.Invoice{}, billingentity.ValidationError{
			Err: err,
		}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billingentity.Invoice, error) {
		invoice, err := s.withLockedInvoiceStateMachine(ctx, input, func(ctx context.Context, sm *InvoiceStateMachine) error {
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
			s.logger.InfoContext(ctx, "invoice advanced", "invoice", input.ID, "from", preActivationStatus, "to", sm.Invoice.Status)

			return nil
		})
		if err != nil {
			return billingentity.Invoice{}, err
		}

		// Given the amount of state transitions, we are only saving the invoice after the whole chain
		// this means that some of the intermittent states will not be persisted in the DB.
		return s.adapter.UpdateInvoice(ctx, invoice)
	})
}

func (s *Service) ApproveInvoice(ctx context.Context, input billing.ApproveInvoiceInput) (billingentity.Invoice, error) {
	return s.executeTriggerOnInvoice(ctx, input, triggerApprove)
}

func (s *Service) RetryInvoice(ctx context.Context, input billing.RetryInvoiceInput) (billingentity.Invoice, error) {
	return s.executeTriggerOnInvoice(ctx, input, triggerRetry)
}

type (
	editCallbackFunc              func(sm *InvoiceStateMachine) error
	executeTriggerApplyOptionFunc func(opts *executeTriggerOnInvoiceOptions)
)

type executeTriggerOnInvoiceOptions struct {
	editCallback  func(sm *InvoiceStateMachine) error
	allowInStates []billingentity.InvoiceStatus
}

func ExecuteTriggerWithEditCallback(cb editCallbackFunc) executeTriggerApplyOptionFunc {
	return func(opts *executeTriggerOnInvoiceOptions) {
		opts.editCallback = cb
	}
}

func ExecuteTriggerWithAllowInStates(states ...billingentity.InvoiceStatus) executeTriggerApplyOptionFunc {
	return func(opts *executeTriggerOnInvoiceOptions) {
		opts.allowInStates = states
	}
}

func (s *Service) executeTriggerOnInvoice(ctx context.Context, invoiceID billingentity.InvoiceID, trigger stateless.Trigger, opts ...executeTriggerApplyOptionFunc) (billingentity.Invoice, error) {
	if err := invoiceID.Validate(); err != nil {
		return billingentity.Invoice{}, billingentity.ValidationError{
			Err: err,
		}
	}

	options := executeTriggerOnInvoiceOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billingentity.Invoice, error) {
		invoice, err := s.withLockedInvoiceStateMachine(ctx, invoiceID, func(ctx context.Context, sm *InvoiceStateMachine) error {
			canFire, err := sm.CanFire(ctx, trigger)
			if err != nil {
				return fmt.Errorf("checking if can fire: %w", err)
			}

			if !canFire && !slices.Contains(options.allowInStates, sm.Invoice.Status) {
				return billingentity.ValidationError{
					Err: fmt.Errorf("cannot %s invoice in status [%s]: %w", trigger, sm.Invoice.Status, billingentity.ErrInvoiceActionNotAvailable),
				}
			}

			if options.editCallback != nil {
				if err := options.editCallback(sm); err != nil {
					return err
				}
			}

			if err := sm.FireAndActivate(ctx, trigger); err != nil {
				validationIssues, err := billingentity.ToValidationIssues(err)
				sm.Invoice.ValidationIssues = validationIssues

				if err != nil {
					return fmt.Errorf("firing %s: %w", trigger, err)
				}

				sm.Invoice, err = s.adapter.UpdateInvoice(ctx, sm.Invoice)
				if err != nil {
					return fmt.Errorf("updating invoice: %w", err)
				}

				return nil
			}

			validationIssues, err := billingentity.ToValidationIssues(
				sm.AdvanceUntilStateStable(ctx),
			)
			if err != nil {
				return fmt.Errorf("advancing invoice: %w", err)
			}

			sm.Invoice.ValidationIssues = validationIssues

			sm.Invoice, err = s.adapter.UpdateInvoice(ctx, sm.Invoice)
			if err != nil {
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

func (s *Service) ValidateInvoiceOwnership(ctx context.Context, input billing.ValidateInvoiceOwnershipInput) error {
	if err := input.Validate(); err != nil {
		return billingentity.ValidationError{
			Err: err,
		}
	}

	ownership, err := s.adapter.GetInvoiceOwnership(ctx, billing.GetInvoiceOwnershipAdapterInput{
		Namespace: input.Namespace,
		ID:        input.InvoiceID,
	})
	if err != nil {
		return err
	}

	if ownership.CustomerID != input.CustomerID {
		return billingentity.NotFoundError{
			Err: fmt.Errorf("customer [%s] does not own invoice [%s]", input.CustomerID, input.InvoiceID),
		}
	}
	return nil
}

func (s *Service) DeleteInvoice(ctx context.Context, input billing.DeleteInvoiceInput) error {
	if err := input.Validate(); err != nil {
		return billingentity.ValidationError{
			Err: err,
		}
	}

	invoice, err := s.executeTriggerOnInvoice(ctx, input, triggerDelete)
	if err != nil {
		return err
	}

	if invoice.Status != billingentity.InvoiceStatusDeleted {
		return billingentity.ValidationError{
			Err: fmt.Errorf("%w [status=%s]", billingentity.ErrInvoiceDeleteFailed, invoice.Status),
		}
	}

	return err
}
