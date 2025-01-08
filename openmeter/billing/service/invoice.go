package billingservice

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/qmuntal/stateless"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	lineservice "github.com/openmeterio/openmeter/openmeter/billing/service/lineservice"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
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

		if input.Expand.GatheringTotals {
			invoices.Items[i], err = s.recalculateGatheringInvoice(ctx, invoices.Items[i])
			if err != nil {
				return billing.ListInvoicesResponse{}, fmt.Errorf("error recalculating gathering invoice [%s]: %w", invoices.Items[i].ID, err)
			}
		}
	}

	return invoices, nil
}

func (s *Service) resolveWorkflowApps(ctx context.Context, invoice billing.Invoice) (billing.Invoice, error) {
	if invoice.ExpandedFields.WorkflowApps {
		resolvedApps, err := s.resolveApps(ctx, invoice.Namespace, invoice.Workflow.AppReferences)
		if err != nil {
			return invoice, fmt.Errorf("error resolving apps for invoice [%s]: %w", invoice.ID, err)
		}

		invoice.Workflow.Apps = &billing.ProfileApps{
			Tax:       resolvedApps.Tax.App,
			Invoicing: resolvedApps.Invoicing.App,
			Payment:   resolvedApps.Payment.App,
		}
	}

	return invoice, nil
}

func (s *Service) resolveStatusDetails(ctx context.Context, invoice billing.Invoice) (billing.Invoice, error) {
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

func (s *Service) recalculateGatheringInvoice(ctx context.Context, invoice billing.Invoice) (billing.Invoice, error) {
	if invoice.Status != billing.InvoiceStatusGathering {
		return invoice, nil
	}

	if invoice.Lines.IsAbsent() {
		// Let's load the lines, if not expanded. This can happen when we are responding to a list request, however
		// this at least allows us to not to expand all the invoices.
		lines, err := s.adapter.ListInvoiceLines(ctx, billing.ListInvoiceLinesAdapterInput{
			Namespace:  invoice.Namespace,
			InvoiceIDs: []string{invoice.ID},
		})
		if err != nil {
			return invoice, fmt.Errorf("loading lines: %w", err)
		}

		invoice.Lines = billing.NewLineChildren(lines)
	}

	for _, line := range invoice.Lines.OrEmpty() {
		if line.Status != billing.InvoiceLineStatusValid || line.DeletedAt != nil {
			continue
		}

		srv, err := s.lineService.FromEntity(line)
		if err != nil {
			return invoice, fmt.Errorf("creating line service: %w", err)
		}

		if err := srv.SnapshotQuantity(ctx, &invoice); err != nil {
			return invoice, fmt.Errorf("snapshotting quantity: %w", err)
		}
	}

	if err := s.invoiceCalculator.Calculate(&invoice); err != nil {
		return invoice, fmt.Errorf("calculating invoice: %w", err)
	}

	return invoice, nil
}

func (s *Service) GetInvoiceByID(ctx context.Context, input billing.GetInvoiceByIdInput) (billing.Invoice, error) {
	invoice, err := s.adapter.GetInvoiceById(ctx, input)
	if err != nil {
		return billing.Invoice{}, err
	}

	invoice, err = s.resolveWorkflowApps(ctx, invoice)
	if err != nil {
		return billing.Invoice{}, fmt.Errorf("error adding fields to invoice [%s]: %w", invoice.ID, err)
	}

	invoice, err = s.resolveStatusDetails(ctx, invoice)
	if err != nil {
		return billing.Invoice{}, fmt.Errorf("error resolving status details for invoice [%s]: %w", invoice.ID, err)
	}

	if input.Expand.GatheringTotals {
		invoice, err = s.recalculateGatheringInvoice(ctx, invoice)
		if err != nil {
			return billing.Invoice{}, fmt.Errorf("error recalculating gathering invoice [%s]: %w", invoice.ID, err)
		}
	}

	return invoice, nil
}

func (s *Service) InvoicePendingLines(ctx context.Context, input billing.InvoicePendingLinesInput) ([]billing.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return TranscationForGatheringInvoiceManipulation(
		ctx,
		s,
		input.Customer,
		func(ctx context.Context) ([]billing.Invoice, error) {
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
			inScopeLines, err := s.gatherInscopeLines(ctx, gatherInScopeLineInput{
				Customer:           input.Customer,
				LinesToInclude:     input.IncludePendingLines,
				AsOf:               asof,
				ProgressiveBilling: customerProfile.Profile.WorkflowConfig.Invoicing.ProgressiveBilling,
			})
			if err != nil {
				return nil, err
			}

			sourceInvoiceIDs := lo.Uniq(lo.Map(inScopeLines, func(l lineservice.LineWithBillablePeriod, _ int) string {
				return l.InvoiceID()
			}))

			if len(sourceInvoiceIDs) == 0 {
				return nil, billing.ValidationError{
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

			createdInvoices := make([]billing.InvoiceID, 0, len(linesByCurrency))

			for currency, lines := range linesByCurrency {
				// let's create the invoice
				invoice, err := s.adapter.CreateInvoice(ctx, billing.CreateInvoiceAdapterInput{
					Namespace: input.Customer.Namespace,
					Customer:  customerProfile.Customer,
					Profile:   customerProfile.Profile,

					Currency: currency,
					Status:   billing.InvoiceStatusDraftCreated,

					Type: billing.InvoiceTypeStandard,
				})
				if err != nil {
					return nil, fmt.Errorf("creating invoice: %w", err)
				}

				createdInvoices = append(createdInvoices, billing.InvoiceID{
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
				return invoiceLineCounts.Counts[billing.InvoiceID{
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
			out := make([]billing.Invoice, 0, len(createdInvoices))
			for _, invoiceID := range createdInvoices {
				invoice, err := s.withLockedInvoiceStateMachine(ctx, invoiceID, func(ctx context.Context, sm *InvoiceStateMachine) error {
					validationIssues, err := billing.ToValidationIssues(
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

type gatherInScopeLineInput struct {
	Customer customerentity.CustomerID
	// If set restricts the lines to be included to these IDs, otherwise the AsOf is used
	// to determine the lines to be included.
	LinesToInclude     mo.Option[[]string]
	AsOf               time.Time
	ProgressiveBilling bool
}

func (s *Service) gatherInscopeLines(ctx context.Context, in gatherInScopeLineInput) ([]lineservice.LineWithBillablePeriod, error) {
	if in.LinesToInclude.IsPresent() {
		lineIDs := in.LinesToInclude.OrEmpty()

		if len(lineIDs) == 0 {
			return nil, billing.ValidationError{
				Err: billing.ErrInvoiceEmpty,
			}
		}

		inScopeLines, err := s.adapter.ListInvoiceLines(ctx,
			billing.ListInvoiceLinesAdapterInput{
				Namespace:  in.Customer.Namespace,
				CustomerID: in.Customer.ID,

				LineIDs: lineIDs,
			})
		if err != nil {
			return nil, fmt.Errorf("resolving in scope lines: %w", err)
		}

		lines, err := s.lineService.FromEntities(inScopeLines)
		if err != nil {
			return nil, fmt.Errorf("creating line services: %w", err)
		}

		// output validation
		resolvedLines, err := lines.ResolveBillablePeriod(ctx, lineservice.ResolveBillablePeriodInput{
			AsOf:               in.AsOf,
			ProgressiveBilling: in.ProgressiveBilling,
		})
		if err != nil {
			return nil, err
		}

		// all lines must be found
		if len(resolvedLines) != len(lineIDs) {
			includedLines := lo.Map(resolvedLines, func(l lineservice.LineWithBillablePeriod, _ int) string {
				return l.ID()
			})

			missingIDs := lo.Without(lineIDs, includedLines...)

			return nil, billing.NotFoundError{
				ID:     strings.Join(missingIDs, ","),
				Entity: billing.EntityInvoiceLine,
				Err:    billing.ErrInvoiceLinesNotBillable,
			}
		}

		return resolvedLines, nil
	}

	lines, err := s.adapter.ListInvoiceLines(ctx,
		billing.ListInvoiceLinesAdapterInput{
			Namespace:  in.Customer.Namespace,
			CustomerID: in.Customer.ID,

			InvoiceStatuses: []billing.InvoiceStatus{
				billing.InvoiceStatusGathering,
			},
			Statuses: []billing.InvoiceLineStatus{
				billing.InvoiceLineStatusValid,
			},
		})
	if err != nil {
		return nil, err
	}

	lineSrvs, err := s.lineService.FromEntities(lines)
	if err != nil {
		return nil, err
	}

	return lineSrvs.ResolveBillablePeriod(ctx, lineservice.ResolveBillablePeriodInput{
		AsOf:               in.AsOf,
		ProgressiveBilling: in.ProgressiveBilling,
	})
}

func (s *Service) withLockedInvoiceStateMachine(
	ctx context.Context,
	invoiceID billing.InvoiceID,
	cb InvoiceStateMachineCallback,
) (billing.Invoice, error) {
	// let's lock the invoice for update, we are using the dedicated call, so that
	// edges won't end up having SELECT FOR UPDATE locks
	if err := s.adapter.LockInvoicesForUpdate(ctx, billing.LockInvoicesForUpdateInput{
		Namespace:  invoiceID.Namespace,
		InvoiceIDs: []string{invoiceID.ID},
	}); err != nil {
		return billing.Invoice{}, fmt.Errorf("locking invoice: %w", err)
	}

	invoice, err := s.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: invoiceID,
		Expand:  billing.InvoiceExpandAll,
	})
	if err != nil {
		return billing.Invoice{}, fmt.Errorf("fetching invoice: %w", err)
	}

	return s.WithInvoiceStateMachine(ctx, invoice, cb)
}

func (s *Service) AdvanceInvoice(ctx context.Context, input billing.AdvanceInvoiceInput) (billing.Invoice, error) {
	if err := input.Validate(); err != nil {
		return billing.Invoice{}, billing.ValidationError{
			Err: err,
		}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billing.Invoice, error) {
		invoice, err := s.withLockedInvoiceStateMachine(ctx, input, func(ctx context.Context, sm *InvoiceStateMachine) error {
			preActivationStatus := sm.Invoice.Status

			canAdvance, err := sm.CanFire(ctx, triggerNext)
			if err != nil {
				return fmt.Errorf("checking if can advance: %w", err)
			}

			if !canAdvance {
				return billing.ValidationError{
					Err: fmt.Errorf("cannot advance invoice in status [%s]: %w", sm.Invoice.Status, billing.ErrInvoiceCannotAdvance),
				}
			}

			validationIssues, err := billing.ToValidationIssues(
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
			return billing.Invoice{}, err
		}

		// Given the amount of state transitions, we are only saving the invoice after the whole chain
		// this means that some of the intermittent states will not be persisted in the DB.
		return s.adapter.UpdateInvoice(ctx, invoice)
	})
}

func (s *Service) ApproveInvoice(ctx context.Context, input billing.ApproveInvoiceInput) (billing.Invoice, error) {
	return s.executeTriggerOnInvoice(ctx, input, triggerApprove)
}

func (s *Service) RetryInvoice(ctx context.Context, input billing.RetryInvoiceInput) (billing.Invoice, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billing.Invoice, error) {
		invoice, err := s.adapter.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
			Invoice: input,
			Expand:  billing.InvoiceExpandAll,
		})
		if err != nil {
			return billing.Invoice{}, err
		}

		// let's clean up all critical validation issues first (as it would prevent advancing the invoice further)
		if len(invoice.ValidationIssues) > 0 {
			invoice.ValidationIssues = invoice.ValidationIssues.Map(func(issue billing.ValidationIssue, _ int) billing.ValidationIssue {
				if issue.Severity == billing.ValidationIssueSeverityCritical {
					issue.Severity = billing.ValidationIssueSeverityWarning
				}

				return issue
			})
		}

		if _, err := s.adapter.UpdateInvoice(ctx, invoice); err != nil {
			return billing.Invoice{}, fmt.Errorf("updating invoice: %w", err)
		}

		return s.executeTriggerOnInvoice(ctx, input, triggerRetry)
	})
}

type (
	editCallbackFunc              func(sm *InvoiceStateMachine) error
	executeTriggerApplyOptionFunc func(opts *executeTriggerOnInvoiceOptions)
)

type executeTriggerOnInvoiceOptions struct {
	editCallback  func(sm *InvoiceStateMachine) error
	allowInStates []billing.InvoiceStatus
}

func ExecuteTriggerWithEditCallback(cb editCallbackFunc) executeTriggerApplyOptionFunc {
	return func(opts *executeTriggerOnInvoiceOptions) {
		opts.editCallback = cb
	}
}

func ExecuteTriggerWithAllowInStates(states ...billing.InvoiceStatus) executeTriggerApplyOptionFunc {
	return func(opts *executeTriggerOnInvoiceOptions) {
		opts.allowInStates = states
	}
}

func (s *Service) executeTriggerOnInvoice(ctx context.Context, invoiceID billing.InvoiceID, trigger stateless.Trigger, opts ...executeTriggerApplyOptionFunc) (billing.Invoice, error) {
	if err := invoiceID.Validate(); err != nil {
		return billing.Invoice{}, billing.ValidationError{
			Err: err,
		}
	}

	options := executeTriggerOnInvoiceOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billing.Invoice, error) {
		invoice, err := s.withLockedInvoiceStateMachine(ctx, invoiceID, func(ctx context.Context, sm *InvoiceStateMachine) error {
			canFire, err := sm.CanFire(ctx, trigger)
			if err != nil {
				return fmt.Errorf("checking if can fire: %w", err)
			}

			if !canFire && !slices.Contains(options.allowInStates, sm.Invoice.Status) {
				return billing.ValidationError{
					Err: fmt.Errorf("cannot %s invoice in status [%s]: %w", trigger, sm.Invoice.Status, billing.ErrInvoiceActionNotAvailable),
				}
			}

			if options.editCallback != nil {
				if err := options.editCallback(sm); err != nil {
					return err
				}
			}

			if err := sm.FireAndActivate(ctx, trigger); err != nil {
				validationIssues, err := billing.ToValidationIssues(err)
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

			validationIssues, err := billing.ToValidationIssues(
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

func (s *Service) DeleteInvoice(ctx context.Context, input billing.DeleteInvoiceInput) error {
	if err := input.Validate(); err != nil {
		return billing.ValidationError{
			Err: err,
		}
	}

	invoice, err := s.executeTriggerOnInvoice(ctx, input, triggerDelete)
	if err != nil {
		return err
	}

	if invoice.Status != billing.InvoiceStatusDeleted {
		// If we have validation issues we return them as the deletion sync handler
		// yields validation errors
		if len(invoice.ValidationIssues) > 0 {
			return billing.ValidationError{
				Err: invoice.ValidationIssues.AsError(),
			}
		}

		return billing.ValidationError{
			Err: fmt.Errorf("%w [status=%s]", billing.ErrInvoiceDeleteFailed, invoice.Status),
		}
	}

	return err
}

func (s *Service) UpdateInvoiceLinesInternal(ctx context.Context, input billing.UpdateInvoiceLinesInternalInput) error {
	if err := input.Validate(); err != nil {
		return billing.ValidationError{
			Err: err,
		}
	}

	if len(input.Lines) == 0 {
		return nil
	}

	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		// Split line child updates should be done invoice by invoice, so let's flatten split lines
		lines := s.flattenSplitLines(input.Lines)

		linesByInvoice := lo.GroupBy(lines, func(line *billing.Line) string {
			return line.InvoiceID
		})

		// We want to upsert the new lines at the end, so that any updates on the gathering invoice will not interfere
		newPendingLines := linesByInvoice[""]
		delete(linesByInvoice, "")

		for invoiceID, lines := range linesByInvoice {
			invoice, err := s.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
				Invoice: billing.InvoiceID{
					ID:        invoiceID,
					Namespace: input.Namespace,
				},
				Expand: billing.InvoiceExpand{
					Lines:        true,
					DeletedLines: true,
					SplitLines:   true,
				},
			})
			if err != nil {
				return fmt.Errorf("fetching invoice: %w", err)
			}

			if input.CustomerID != invoice.Customer.CustomerID {
				return billing.ValidationError{
					Err: fmt.Errorf("customer mismatch: [input.customer=%s] vs [invoice.customer=%s]", input.CustomerID, invoice.Customer.CustomerID),
				}
			}

			if invoice.Status == billing.InvoiceStatusGathering {
				for _, line := range lines {
					if !invoice.Lines.ReplaceByID(line.ID, line) {
						return fmt.Errorf("line[%s] not found in invoice[%s]", line.ID, invoice.ID)
					}
				}

				if _, err := s.adapter.UpdateInvoice(ctx, invoice); err != nil {
					return fmt.Errorf("updating gathering invoice: %w", err)
				}
			} else {
				if err := s.handleNonGatheringInvoiceLineUpdate(ctx, invoice, lines); err != nil {
					return fmt.Errorf("handling invoice line update: %w", err)
				}
			}
		}

		_, err := s.CreatePendingInvoiceLines(ctx, billing.CreateInvoiceLinesInput{
			Namespace: input.Namespace,
			Lines: lo.Map(newPendingLines, func(line *billing.Line, _ int) billing.LineWithCustomer {
				return billing.LineWithCustomer{
					Line:       *line,
					CustomerID: input.CustomerID,
				}
			}),
		})
		if err != nil {
			return fmt.Errorf("creating new pending lines: %w", err)
		}

		// Note: The gathering invoice will be maintained by the CreatePendingInvoiceLines call, so we don't need to care for any empty gathering
		// invoices here.
		return nil
	})
}

func (s *Service) handleNonGatheringInvoiceLineUpdate(ctx context.Context, invoice billing.Invoice, lines []*billing.Line) error {
	if invoice.Lines.IsAbsent() {
		return errors.New("cannot update invoice without expanded lines")
	}

	existingInvoiceLinesByID := lo.GroupBy(invoice.Lines.OrEmpty(), func(line *billing.Line) string {
		return line.ID
	})

	// Let's look for the lines that have been updated
	changedLines := make([]*billing.Line, 0, len(lines))

	for _, line := range lines {
		existingLines, existingLineFound := existingInvoiceLinesByID[line.ID]

		if existingLineFound {
			if len(existingLines) != 1 {
				return fmt.Errorf("line[%s] has more than one entry in the invoice", line.ID)
			}

			existingLine := existingLines[0]
			if !existingLine.LineBase.Equal(line.LineBase) {
				changedLines = append(changedLines, line)
			}
		} else {
			changedLines = append(changedLines, line)
		}
	}

	if len(changedLines) == 0 {
		return nil
	}

	// Let's try to avoid touching an immutable invoice
	if invoice.StatusDetails.Immutable {
		// We only care about lines that are affecting the balance at this stage, as
		// there's a chance that an invoice being created and a subscription update is
		// happening in the same time.

		return fmt.Errorf("invoice is immutable, but voiding is not implemented yet: invoice[%s] lineIDs:[%s]",
			invoice.ID,
			strings.Join(lo.Map(changedLines, func(line *billing.Line, _ int) string {
				return line.ID
			}), ","),
		)
	}

	// Note: in the current setup this could only happen if there's a parallel progressive invoice creation and
	// subscription edit.
	for _, line := range changedLines {
		// Should not happen as split lines can only live on gathering invoices
		if line.Status == billing.InvoiceLineStatusSplit {
			return fmt.Errorf("split line[%s] cannot be updated", line.ID)
		}

		// Let's update the snapshot of the line, as we might have changed the period
		srv, err := s.lineService.FromEntity(line)
		if err != nil {
			return fmt.Errorf("creating line service: %w", err)
		}

		if err := srv.Validate(ctx, &invoice); err != nil {
			return fmt.Errorf("validating line: %w", err)
		}

		if err := srv.SnapshotQuantity(ctx, &invoice); err != nil {
			return fmt.Errorf("snapshotting quantity: %w", err)
		}

		if err := srv.CalculateDetailedLines(); err != nil {
			return fmt.Errorf("calculating detailed lines: %w", err)
		}

		if err := srv.UpdateTotals(); err != nil {
			return fmt.Errorf("updating totals: %w", err)
		}
	}

	invoice, err := s.executeTriggerOnInvoice(
		ctx,
		invoice.InvoiceID(),
		triggerUpdated,
		ExecuteTriggerWithAllowInStates(billing.InvoiceStatusDraftUpdating),
		ExecuteTriggerWithEditCallback(func(sm *InvoiceStateMachine) error {
			for _, line := range changedLines {
				if !invoice.Lines.ReplaceByID(line.ID, line) {
					return fmt.Errorf("line[%s] not found in invoice[%s]", line.ID, invoice.ID)
				}
			}
			return nil
		}),
	)

	return err
}

func (s *Service) flattenSplitLines(lines []*billing.Line) []*billing.Line {
	out := make([]*billing.Line, 0, len(lines))
	for _, line := range lines {
		out = append(out, line)

		if line.Status == billing.InvoiceLineStatusSplit {
			out = append(out, line.Children.OrEmpty()...)
			line.DisassociateChildren()
		}
	}

	return out
}
