package billingservice

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
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

func (s *Service) UpdateInvoice(ctx context.Context, input billing.UpdateInvoiceInput) (billing.Invoice, error) {
	if err := input.Validate(); err != nil {
		return billing.Invoice{}, billing.ValidationError{
			Err: err,
		}
	}

	invoice, err := s.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: input.Invoice,
		Expand:  billing.InvoiceExpand{}, // We don't want to expand anything as we will have to refetch the invoice anyway
	})
	if err != nil {
		return billing.Invoice{}, fmt.Errorf("fetching invoice: %w", err)
	}

	if invoice.Status == billing.InvoiceStatusGathering {
		return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billing.Invoice, error) {
			// let's lock the invoice for update, we are using the dedicated call, so that
			// edges won't end up having SELECT FOR UPDATE locks
			if err := s.adapter.LockInvoicesForUpdate(ctx, billing.LockInvoicesForUpdateInput{
				Namespace:  input.Invoice.Namespace,
				InvoiceIDs: []string{input.Invoice.ID},
			}); err != nil {
				return billing.Invoice{}, fmt.Errorf("locking invoice: %w", err)
			}

			invoice, err := s.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
				Invoice: input.Invoice,
				// We need split lines too, as for gathering invoices we need to edit those too
				Expand: billing.InvoiceExpandAll.SetSplitLines(true),
			})
			if err != nil {
				return billing.Invoice{}, fmt.Errorf("fetching invoice: %w", err)
			}

			if err := input.EditFn(&invoice); err != nil {
				return billing.Invoice{}, fmt.Errorf("editing invoice: %w", err)
			}

			if err := invoice.Validate(); err != nil {
				return billing.Invoice{}, billing.ValidationError{
					Err: err,
				}
			}

			return s.adapter.UpdateInvoice(ctx, invoice)
		})
	}

	return s.executeTriggerOnInvoice(
		ctx,
		input.Invoice,
		triggerUpdated,
		ExecuteTriggerWithEditCallback(func(sm *InvoiceStateMachine) error {
			if err := input.EditFn(&sm.Invoice); err != nil {
				return fmt.Errorf("editing invoice: %w", err)
			}

			if err := sm.Invoice.Validate(); err != nil {
				return billing.ValidationError{
					Err: err,
				}
			}

			return nil
		}),
	)
}

func (s Service) SimulateInvoice(ctx context.Context, input billing.SimulateInvoiceInput) (billing.Invoice, error) {
	if err := input.Validate(); err != nil {
		return billing.Invoice{}, fmt.Errorf("validating input: %w", err)
	}

	billingProfile, err := s.getProfileWithCustomerOverride(ctx, s.adapter, billing.GetProfileWithCustomerOverrideInput{
		Namespace:  input.CustomerID.Namespace,
		CustomerID: input.CustomerID.ID,
	})
	if err != nil {
		return billing.Invoice{}, fmt.Errorf("getting profile with customer override: %w", err)
	}

	now := time.Now()

	invoice := billing.Invoice{
		InvoiceBase: billing.InvoiceBase{
			Namespace: input.CustomerID.Namespace,
			ID:        ulid.Make().String(),

			Number: input.Number,

			Type: billing.InvoiceTypeStandard,

			Currency:      input.Currency,
			Status:        billing.InvoiceStatusDraftCreated,
			StatusDetails: billing.InvoiceStatusDetails{},
			CreatedAt:     now,
			UpdatedAt:     now,

			Customer: billing.InvoiceCustomer{
				CustomerID: input.CustomerID.ID,

				Name:             billingProfile.Customer.Name,
				BillingAddress:   billingProfile.Customer.BillingAddress,
				UsageAttribution: billingProfile.Customer.UsageAttribution,
			},

			Supplier: billing.SupplierContact{
				ID:      billingProfile.Profile.Supplier.ID,
				Name:    billingProfile.Profile.Supplier.Name,
				Address: billingProfile.Profile.Supplier.Address,
				TaxCode: billingProfile.Profile.Supplier.TaxCode,
			},

			Workflow: billing.InvoiceWorkflow{
				AppReferences:          lo.FromPtrOr(billingProfile.Profile.AppReferences, billing.ProfileAppReferences{}),
				Apps:                   billingProfile.Profile.Apps,
				SourceBillingProfileID: billingProfile.Profile.ID,
				Config:                 billingProfile.Profile.WorkflowConfig,
			},
		},
	}

	inputLines := input.Lines.OrEmpty()

	invoice.Lines = billing.NewLineChildren(
		lo.Map(inputLines, func(line *billing.Line, _ int) *billing.Line {
			line.Namespace = input.CustomerID.Namespace
			if line.ID == "" {
				line.ID = ulid.Make().String()
			}
			line.CreatedAt = now
			line.UpdatedAt = now
			line.Currency = input.Currency
			line.InvoiceID = invoice.ID

			return line
		}),
	)

	if err := invoice.Validate(); err != nil {
		return billing.Invoice{}, billing.ValidationError{
			Err: err,
		}
	}

	// Let's update the lines and the detailed lines
	for _, line := range invoice.Lines.OrEmpty() {
		if err := line.Validate(); err != nil {
			return billing.Invoice{}, billing.ValidationError{
				Err: err,
			}
		}

		lineSvc, err := s.lineService.FromEntity(line)
		if err != nil {
			return billing.Invoice{}, fmt.Errorf("creating line service from entity: %w", err)
		}

		if err := lineSvc.Validate(ctx, &invoice); err != nil {
			return billing.Invoice{}, billing.ValidationError{
				Err: err,
			}
		}

		if err := lineSvc.CalculateDetailedLines(); err != nil {
			return billing.Invoice{}, fmt.Errorf("calculating detailed lines: %w", err)
		}

		if err := lineSvc.UpdateTotals(); err != nil {
			return billing.Invoice{}, fmt.Errorf("updating totals: %w", err)
		}
	}

	// Let's simulate a recalculation of the invoice
	if err := s.invoiceCalculator.Calculate(&invoice); err != nil {
		return billing.Invoice{}, err
	}

	for _, validationIssue := range invoice.ValidationIssues {
		if validationIssue.Severity == billing.ValidationIssueSeverityCritical {
			invoice.Status = billing.InvoiceStatusDraftInvalid
			invoice.StatusDetails.Failed = true
			break
		}
	}

	return invoice, nil
}

func (s *Service) UpsertValidationIssues(ctx context.Context, input billing.UpsertValidationIssuesInput) error {
	if err := input.Validate(); err != nil {
		return billing.ValidationError{
			Err: err,
		}
	}

	invoice, err := s.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: input.Invoice,
		Expand:  billing.InvoiceExpandAll,
	})
	if err != nil {
		return fmt.Errorf("fetching invoice: %w", err)
	}

	if !invoice.StatusDetails.Immutable {
		return fmt.Errorf("invoice validation issues can only be manipulated without the state-machine, if the invoice is in immutable state [invoice_id=%s,status=%s]", invoice.ID, invoice.Status)
	}

	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		invoice.ValidationIssues = input.Issues

		_, err := s.adapter.UpdateInvoice(ctx, invoice)
		if err != nil {
			return fmt.Errorf("updating invoice: %w", err)
		}

		return nil
	})
}
