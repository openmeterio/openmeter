package billingservice

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/service/lineservice"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

var _ billing.InvoiceService = (*Service)(nil)

func (s *Service) ListInvoices(ctx context.Context, input billing.ListInvoicesInput) (billing.ListInvoicesResponse, error) {
	invoices, err := s.adapter.ListInvoices(ctx, input)
	if err != nil {
		return billing.ListInvoicesResponse{}, err
	}

	for i := range invoices.Items {
		invoiceID := invoices.Items[i].ID

		invoices.Items[i], err = s.resolveWorkflowApps(ctx, invoices.Items[i])
		if err != nil {
			return billing.ListInvoicesResponse{}, fmt.Errorf("error resolving workflow apps [%s]: %w", invoiceID, err)
		}

		invoices.Items[i], err = s.resolveStatusDetails(ctx, invoices.Items[i])
		if err != nil {
			return billing.ListInvoicesResponse{}, fmt.Errorf("error resolving status details for invoice [%s]: %w", invoiceID, err)
		}

		if input.Expand.RecalculateGatheringInvoice {
			invoices.Items[i], err = s.recalculateGatheringInvoice(ctx, recalculateGatheringInvoiceInput{
				Invoice: invoices.Items[i],
				Expand:  input.Expand,
			})
			if err != nil {
				return billing.ListInvoicesResponse{}, fmt.Errorf("error recalculating gathering invoice [%s]: %w", invoiceID, err)
			}
		}
	}

	return invoices, nil
}

func (s *Service) resolveWorkflowApps(ctx context.Context, invoice billing.Invoice) (billing.Invoice, error) {
	taxApp, err := s.appService.GetApp(ctx, invoice.Workflow.AppReferences.Tax)
	if err != nil {
		return invoice, fmt.Errorf("error getting tax app for invoice [%s]: %w", invoice.ID, err)
	}

	invoicingApp, err := s.appService.GetApp(ctx, invoice.Workflow.AppReferences.Invoicing)
	if err != nil {
		return invoice, fmt.Errorf("error getting invoicing app for invoice [%s]: %w", invoice.ID, err)
	}

	paymentApp, err := s.appService.GetApp(ctx, invoice.Workflow.AppReferences.Payment)
	if err != nil {
		return invoice, fmt.Errorf("error getting payment app for invoice [%s]: %w", invoice.ID, err)
	}

	invoice.Workflow.Apps = &billing.ProfileApps{
		Tax:       taxApp,
		Invoicing: invoicingApp,
		Payment:   paymentApp,
	}

	return invoice, nil
}

func (s *Service) resolveStatusDetails(ctx context.Context, invoice billing.Invoice) (billing.Invoice, error) {
	if invoice.Status == billing.InvoiceStatusGathering {
		// Let's use the default and recalculateGatheringInvoice will fix the gaps
		return invoice, nil
	}

	// If we are not in a time sensitive state and the status details is not empty, we can return the invoice as is, so we
	// don't have to lock the invoice in the DB
	if !lo.IsEmpty(invoice.StatusDetails) &&
		!slices.Contains(
			[]billing.InvoiceStatus{
				billing.InvoiceStatusDraftWaitingForCollection,
				billing.InvoiceStatusDraftWaitingAutoApproval,
			}, invoice.Status) {
		// The status details depends on the current time, so we should recalculate
		return invoice, nil
	}

	// let's resolve the statatus details (the invoice state machine has this side-effect after the callback)
	resolvedInvoice, err := s.WithInvoiceStateMachine(ctx, invoice, func(ctx context.Context, sm *InvoiceStateMachine) error {
		return nil
	})
	if err != nil {
		return invoice, fmt.Errorf("resolving status details: %w", err)
	}

	return resolvedInvoice, nil
}

type recalculateGatheringInvoiceInput struct {
	Invoice billing.Invoice
	Expand  billing.InvoiceExpand
}

func (s *Service) recalculateGatheringInvoice(ctx context.Context, in recalculateGatheringInvoiceInput) (billing.Invoice, error) {
	invoice := in.Invoice

	if invoice.Status != billing.InvoiceStatusGathering {
		return invoice, nil
	}

	wasLinesAbsent := invoice.Lines.IsAbsent()

	if wasLinesAbsent {
		// Let's load the lines, if not expanded. This can happen when we are responding to a list request, however
		// this at least allows us to not to expand all the invoices.
		lines, err := s.adapter.ListInvoiceLines(ctx, billing.ListInvoiceLinesAdapterInput{
			Namespace:  invoice.Namespace,
			InvoiceIDs: []string{invoice.ID},
			Statuses: []billing.InvoiceLineStatus{
				billing.InvoiceLineStatusValid,
			},
		})
		if err != nil {
			return invoice, fmt.Errorf("loading lines: %w", err)
		}

		invoice.Lines = billing.NewInvoiceLines(lines)
	}

	hasInvoicableLines := mo.Option[bool]{}
	now := clock.Now()

	customerProfile, err := s.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: invoice.CustomerID(),
		Expand: billing.CustomerOverrideExpand{
			Customer: true,
		},
	})
	if err != nil {
		return invoice, fmt.Errorf("fetching profile: %w", err)
	}

	inScopeLineSvcs, err := s.lineService.FromEntities(
		lo.Filter(invoice.Lines.OrEmpty(), func(line *billing.Line, _ int) bool {
			return line.DeletedAt == nil
		}),
	)
	if err != nil {
		return invoice, fmt.Errorf("creating line services: %w", err)
	}

	if customerProfile.Customer == nil {
		return invoice, fmt.Errorf("customer profile is nil")
	}

	if err := s.snapshotLineQuantitiesInParallel(ctx, billing.NewInvoiceCustomer(*customerProfile.Customer), inScopeLineSvcs); err != nil {
		return invoice, fmt.Errorf("snapshotting lines: %w", err)
	}

	for _, lineSvc := range inScopeLineSvcs {
		period, err := lineSvc.CanBeInvoicedAsOf(ctx, lineservice.CanBeInvoicedAsOfInput{
			AsOf:               now,
			ProgressiveBilling: customerProfile.MergedProfile.WorkflowConfig.Invoicing.ProgressiveBilling,
		})
		if err != nil {
			return invoice, fmt.Errorf("checking if can be invoiced: %w", err)
		}

		if period != nil {
			hasInvoicableLines = mo.Some(true)
		}
	}

	invoice.QuantitySnapshotedAt = lo.ToPtr(now)

	if err := s.invoiceCalculator.CalculateGatheringInvoiceWithLiveData(&invoice); err != nil {
		return invoice, fmt.Errorf("calculating invoice: %w", err)
	}

	if wasLinesAbsent {
		// If the original user intent was to not to receive the lines, let's not send them
		invoice.Lines = billing.InvoiceLines{}
	} else {
		// For calulcations we fetch the split lines, but we don't want to expose them for the response
		invoice.Lines = billing.NewInvoiceLines(
			lo.Filter(invoice.Lines.OrEmpty(), func(line *billing.Line, _ int) bool {
				if !in.Expand.DeletedLines && line.DeletedAt != nil {
					return false
				}

				return true
			}),
		)
	}

	// Let's update the status details based on the lines available
	// TODO[later]: If this sugar is removed due to properly implemented progressive billing stack, we need to cache the when the invoice is first invoicable in the db
	// so that we don't have to fetch all the lines to have proper status details.

	invoice.StatusDetails = billing.InvoiceStatusDetails{
		Immutable: false,
		AvailableActions: billing.InvoiceAvailableActions{
			Invoice: lo.If(hasInvoicableLines.IsPresent(), &billing.InvoiceAvailableActionInvoiceDetails{}).Else(nil),
		},
	}

	return invoice, nil
}

func (s *Service) GetInvoiceByID(ctx context.Context, input billing.GetInvoiceByIdInput) (billing.Invoice, error) {
	invoiceID := input.Invoice.ID

	invoice, err := s.adapter.GetInvoiceById(ctx, input)
	if err != nil {
		return billing.Invoice{}, err
	}

	invoice, err = s.resolveWorkflowApps(ctx, invoice)
	if err != nil {
		return billing.Invoice{}, fmt.Errorf("error resolving workload apps for invoice [%s]: %w", invoiceID, err)
	}

	invoice, err = s.resolveStatusDetails(ctx, invoice)
	if err != nil {
		return billing.Invoice{}, fmt.Errorf("error resolving status details for invoice [%s]: %w", invoiceID, err)
	}

	if input.Expand.RecalculateGatheringInvoice {
		invoice, err = s.recalculateGatheringInvoice(ctx, recalculateGatheringInvoiceInput{
			Invoice: invoice,
			Expand:  input.Expand,
		})
		if err != nil {
			return billing.Invoice{}, fmt.Errorf("error recalculating gathering invoice [%s]: %w", invoiceID, err)
		}
	}

	return invoice, nil
}

func (s *Service) advanceUntilStateStable(ctx context.Context, sm *InvoiceStateMachine) error {
	if s.advancementStrategy == billing.QueuedAdvancementStrategy {
		return s.publisher.Publish(ctx, billing.AdvanceInvoiceEvent{
			Invoice:    sm.Invoice.InvoiceID(),
			CustomerID: sm.Invoice.Customer.CustomerID,
		})
	}

	validationIssues, err := billing.ToValidationIssues(
		sm.AdvanceUntilStateStable(ctx),
	)
	if err != nil {
		return fmt.Errorf("activating invoice: %w", err)
	}

	sm.Invoice.ValidationIssues = validationIssues
	return nil
}

type withLockedStateMachineInput struct {
	InvoiceID           billing.InvoiceID
	Callback            InvoiceStateMachineCallback
	IncludeDeletedLines bool
}

func (s *Service) withLockedInvoiceStateMachine(
	ctx context.Context,
	in withLockedStateMachineInput,
) (billing.Invoice, error) {
	invoiceHeader, err := s.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: in.InvoiceID,
		Expand:  billing.InvoiceExpand{}, // We just need the customer ID at this point
	})
	if err != nil {
		return billing.Invoice{}, fmt.Errorf("fetching invoice: %w", err)
	}

	return transcationForInvoiceManipulation(ctx, s, invoiceHeader.CustomerID(), func(ctx context.Context) (billing.Invoice, error) {
		invoice, err := s.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: in.InvoiceID,
			Expand: billing.InvoiceExpandAll.
				SetDeletedLines(in.IncludeDeletedLines),
		})
		if err != nil {
			return billing.Invoice{}, fmt.Errorf("fetching invoice: %w", err)
		}

		return s.WithInvoiceStateMachine(ctx, invoice, in.Callback)
	})
}

func (s *Service) AdvanceInvoice(ctx context.Context, input billing.AdvanceInvoiceInput) (billing.Invoice, error) {
	if err := input.Validate(); err != nil {
		return billing.Invoice{}, billing.ValidationError{
			Err: err,
		}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billing.Invoice, error) {
		invoice, err := s.withLockedInvoiceStateMachine(ctx, withLockedStateMachineInput{
			InvoiceID: input,
			Callback: func(ctx context.Context, sm *InvoiceStateMachine) error {
				preActivationStatus := sm.Invoice.Status

				canAdvance, err := sm.CanFire(ctx, billing.TriggerNext)
				if err != nil {
					return fmt.Errorf("checking if can advance: %w", err)
				}

				if !canAdvance {
					return billing.ValidationError{
						Err: fmt.Errorf("cannot advance invoice in status [%s]: %w", sm.Invoice.Status, billing.ErrInvoiceCannotAdvance),
					}
				}

				if err := s.advanceUntilStateStable(ctx, sm); err != nil {
					return fmt.Errorf("advancing invoice: %w", err)
				}

				s.logger.InfoContext(ctx, "invoice advanced", "invoice", input.ID, "from", preActivationStatus, "to", sm.Invoice.Status)

				return nil
			},
		})
		if err != nil {
			return billing.Invoice{}, err
		}

		// Given the amount of state transitions, we are only saving the invoice after the whole chain
		// this means that some of the intermittent states will not be persisted in the DB.
		return s.updateInvoice(ctx, invoice)
	})
}

func (s *Service) ApproveInvoice(ctx context.Context, input billing.ApproveInvoiceInput) (billing.Invoice, error) {
	return s.executeTriggerOnInvoice(ctx, input, billing.TriggerApprove)
}

func (s *Service) SnapshotQuantities(ctx context.Context, input billing.SnapshotQuantitiesInput) (billing.Invoice, error) {
	return s.executeTriggerOnInvoice(ctx, input, billing.TriggerSnapshotQuantities)
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

		if _, err := s.updateInvoice(ctx, invoice); err != nil {
			return billing.Invoice{}, fmt.Errorf("updating invoice[%s]: %w", input.ID, err)
		}

		return s.executeTriggerOnInvoice(ctx, input, billing.TriggerRetry)
	})
}

type (
	editCallbackFunc              func(sm *InvoiceStateMachine) error
	executeTriggerApplyOptionFunc func(opts *executeTriggerOnInvoiceOptions)
)

type executeTriggerOnInvoiceOptions struct {
	editCallback        func(sm *InvoiceStateMachine) error
	allowInStates       []billing.InvoiceStatus
	includeDeletedLines bool
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

func ExecuteTriggerWithIncludeDeletedLines(includeDeletedLines bool) executeTriggerApplyOptionFunc {
	return func(opts *executeTriggerOnInvoiceOptions) {
		opts.includeDeletedLines = includeDeletedLines
	}
}

func (s *Service) executeTriggerOnInvoice(ctx context.Context, invoiceID billing.InvoiceID, trigger billing.InvoiceTrigger, opts ...executeTriggerApplyOptionFunc) (billing.Invoice, error) {
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
		invoice, err := s.withLockedInvoiceStateMachine(ctx, withLockedStateMachineInput{
			InvoiceID:           invoiceID,
			IncludeDeletedLines: options.includeDeletedLines,
			Callback: func(ctx context.Context, sm *InvoiceStateMachine) error {
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

					if err := sm.Invoice.Validate(); err != nil {
						return billing.ValidationError{
							Err: err,
						}
					}

					if err := s.checkIfLinesAreInvoicable(ctx, &sm.Invoice, sm.Invoice.Workflow.Config.Invoicing.ProgressiveBilling); err != nil {
						return err
					}

					// This forces line ID generation for new or added lines
					sm.Invoice, err = s.updateInvoice(ctx, sm.Invoice)
					if err != nil {
						return fmt.Errorf("updating invoice[%s]: %w", invoiceID, err)
					}
				}

				if err := sm.FireAndActivate(ctx, trigger); err != nil {
					validationIssues, err := billing.ToValidationIssues(err)
					sm.Invoice.ValidationIssues = validationIssues

					if err != nil {
						return fmt.Errorf("firing %s: %w", trigger, err)
					}

					sm.Invoice, err = s.updateInvoice(ctx, sm.Invoice)
					if err != nil {
						return fmt.Errorf("updating invoice[%s]: %w", invoiceID, err)
					}

					return nil
				}

				if err := s.advanceUntilStateStable(ctx, sm); err != nil {
					return fmt.Errorf("advancing invoice: %w", err)
				}

				sm.Invoice, err = s.updateInvoice(ctx, sm.Invoice)
				if err != nil {
					return fmt.Errorf("updating invoice[%s]: %w", invoiceID, err)
				}

				return nil
			},
		})
		if err != nil {
			return invoice, err
		}

		return invoice, nil
	})
}

func (s *Service) DeleteInvoice(ctx context.Context, input billing.DeleteInvoiceInput) (billing.Invoice, error) {
	if err := input.Validate(); err != nil {
		return billing.Invoice{}, billing.ValidationError{
			Err: err,
		}
	}

	// Let's see if we are talking about a gathering invoice
	invoice, err := s.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: input,
	})
	if err != nil {
		return billing.Invoice{}, err
	}

	if invoice.Status == billing.InvoiceStatusGathering {
		// TODO: If this becomes a UX problem we can always edit the invoice to delete all the gathering lines
		return billing.Invoice{}, billing.ValidationError{
			Err: fmt.Errorf("gathering invoice[%s]: %w", invoice.ID, billing.ErrInvoiceCannotDeleteGathering),
		}
	}

	return s.executeTriggerOnInvoice(ctx, input, billing.TriggerDelete)
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
		customerProfile, err := s.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Customer: invoice.CustomerID(),
		})
		if err != nil {
			return billing.Invoice{}, fmt.Errorf("fetching profile: %w", err)
		}

		return transcationForInvoiceManipulation(ctx, s, invoice.CustomerID(), func(ctx context.Context) (billing.Invoice, error) {
			invoice, err := s.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
				Invoice: input.Invoice,
				Expand: billing.InvoiceExpandAll.
					SetDeletedLines(input.IncludeDeletedLines),
			})
			if err != nil {
				return billing.Invoice{}, fmt.Errorf("fetching invoice: %w", err)
			}

			if err := input.EditFn(&invoice); err != nil {
				return billing.Invoice{}, fmt.Errorf("editing invoice: %w", err)
			}

			lineServices, err := s.lineService.FromEntities(invoice.Lines.OrEmpty())
			if err != nil {
				return billing.Invoice{}, fmt.Errorf("creating line services: %w", err)
			}

			for idx := range lineServices {
				lineSvc, err := lineServices[idx].PrepareForCreate(ctx)
				if err != nil {
					return billing.Invoice{}, fmt.Errorf("preparing line[%s] for create: %w", lineServices[idx].ID(), err)
				}

				lineServices[idx] = lineSvc
			}

			invoice.Lines = billing.NewInvoiceLines(lineServices.ToEntities())

			if err := s.invoiceCalculator.CalculateGatheringInvoice(&invoice); err != nil {
				return billing.Invoice{}, fmt.Errorf("calculating invoice[%s]: %w", invoice.ID, err)
			}

			if err := invoice.Validate(); err != nil {
				return billing.Invoice{}, billing.ValidationError{
					Err: err,
				}
			}

			// Check if the new lines are still invoicable
			if err := s.checkIfLinesAreInvoicable(ctx, &invoice, customerProfile.MergedProfile.WorkflowConfig.Invoicing.ProgressiveBilling); err != nil {
				return billing.Invoice{}, err
			}

			invoice, err = s.updateInvoice(ctx, invoice)
			if err != nil {
				return billing.Invoice{}, fmt.Errorf("updating invoice[%s]: %w", input.Invoice.ID, err)
			}

			// Auto delete the invoice if it has no lines, this needs to happen here, as we are in a
			// TranscationForGatheringInvoiceManipulation

			if invoice.Lines.NonDeletedLineCount() == 0 {
				if err := s.adapter.DeleteGatheringInvoices(ctx, billing.DeleteGatheringInvoicesInput{
					Namespace:  input.Invoice.Namespace,
					InvoiceIDs: []string{invoice.ID},
				}); err != nil {
					return billing.Invoice{}, fmt.Errorf("deleting gathering invoice: %w", err)
				}
			}

			return invoice, nil
		})
	}

	return s.executeTriggerOnInvoice(
		ctx,
		input.Invoice,
		billing.TriggerUpdated,
		ExecuteTriggerWithIncludeDeletedLines(input.IncludeDeletedLines),
		ExecuteTriggerWithAllowInStates(billing.InvoiceStatusDraftUpdating),
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

// updateInvoice calls the adapter to update the invoice and returns the updated invoice including any expands that are
// the responsibility of the service
func (s Service) updateInvoice(ctx context.Context, in billing.UpdateInvoiceAdapterInput) (billing.Invoice, error) {
	invoice, err := s.resolveStatusDetails(ctx, in)
	if err != nil {
		return billing.Invoice{}, fmt.Errorf("error resolving status details for invoice [%s]: %w", in.ID, err)
	}

	invoice, err = s.adapter.UpdateInvoice(ctx, invoice)
	if err != nil {
		return billing.Invoice{}, err
	}

	invoice, err = s.resolveWorkflowApps(ctx, invoice)
	if err != nil {
		return billing.Invoice{}, fmt.Errorf("error resolving workflow apps for invoice [%s]: %w", in.ID, err)
	}

	return invoice, nil
}

func (s Service) checkIfLinesAreInvoicable(ctx context.Context, invoice *billing.Invoice, progressiveBilling bool) error {
	inScopeLineServices, err := s.lineService.FromEntities(
		lo.Filter(invoice.Lines.OrEmpty(), func(line *billing.Line, _ int) bool {
			return line.DeletedAt == nil
		}),
	)
	if err != nil {
		return fmt.Errorf("creating line services: %w", err)
	}

	return errors.Join(
		lo.Map(inScopeLineServices, func(lineSvc lineservice.Line, _ int) error {
			if err := lineSvc.Validate(ctx, invoice); err != nil {
				return fmt.Errorf("validating line[%s]: %w", lineSvc.ID(), err)
			}

			period, err := lineSvc.CanBeInvoicedAsOf(ctx, lineservice.CanBeInvoicedAsOfInput{
				AsOf:               lineSvc.InvoiceAt(),
				ProgressiveBilling: progressiveBilling,
			})
			if err != nil {
				return fmt.Errorf("checking if line[%s] can be invoiced: %w", lineSvc.ID(), err)
			}

			if period == nil {
				return billing.ValidationError{
					Err: fmt.Errorf("line[%s]: %w as of %s", lineSvc.ID(), billing.ErrInvoiceLinesNotBillable, lineSvc.Period().End),
				}
			}

			return nil
		})...,
	)
}

func (s Service) SimulateInvoice(ctx context.Context, input billing.SimulateInvoiceInput) (billing.Invoice, error) {
	if err := input.Validate(); err != nil {
		return billing.Invoice{}, fmt.Errorf("validating input: %w", err)
	}

	var customerProfile billing.CustomerOverrideWithDetails

	if input.CustomerID != nil {
		var err error
		customerProfile, err = s.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Customer: customer.CustomerID{
				Namespace: input.Namespace,
				ID:        *input.CustomerID,
			},
			Expand: billing.CustomerOverrideExpand{Customer: true},
		})
		if err != nil {
			return billing.Invoice{}, fmt.Errorf("getting profile with customer override: %w", err)
		}
	}

	if input.Customer != nil {
		var err error

		customerProfile, err = s.buildSimulatedCustomerProfile(ctx, input.Namespace, *input.Customer)
		if err != nil {
			return billing.Invoice{}, fmt.Errorf("building simulated customer profile: %w", err)
		}
	}

	now := clock.Now()

	invoice := billing.Invoice{
		InvoiceBase: billing.InvoiceBase{
			Namespace: input.Namespace,
			ID:        ulid.Make().String(),

			Number: lo.FromPtrOr(input.Number, "INV-SIMULATED"),

			Type: billing.InvoiceTypeStandard,

			Currency:      input.Currency,
			Status:        billing.InvoiceStatusDraftCreated,
			StatusDetails: billing.InvoiceStatusDetails{},
			CreatedAt:     now,
			UpdatedAt:     now,

			Customer: billing.NewInvoiceCustomer(*customerProfile.Customer),

			Supplier: billing.SupplierContact{
				ID:      customerProfile.MergedProfile.Supplier.ID,
				Name:    customerProfile.MergedProfile.Supplier.Name,
				Address: customerProfile.MergedProfile.Supplier.Address,
				TaxCode: customerProfile.MergedProfile.Supplier.TaxCode,
			},

			Workflow: billing.InvoiceWorkflow{
				AppReferences:          lo.FromPtr(customerProfile.MergedProfile.AppReferences),
				Apps:                   customerProfile.MergedProfile.Apps,
				SourceBillingProfileID: customerProfile.MergedProfile.ID,
				Config:                 customerProfile.MergedProfile.WorkflowConfig,
			},
		},
	}

	inputLines := input.Lines.OrEmpty()

	invoice.Lines = billing.NewInvoiceLines(
		lo.Map(inputLines, func(line *billing.Line, _ int) *billing.Line {
			line.Namespace = input.Namespace
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

	err := errors.Join(lo.Map(invoice.Lines.OrEmpty(), func(line *billing.Line, _ int) error {
		return line.Validate()
	})...)
	if err != nil {
		return billing.Invoice{}, billing.ValidationError{
			Err: err,
		}
	}

	inScopeLineSvcs, err := s.lineService.FromEntities(invoice.Lines.OrEmpty())
	if err != nil {
		return billing.Invoice{}, fmt.Errorf("creating line services: %w", err)
	}

	// Let's update the lines and the detailed lines
	for _, lineSvc := range inScopeLineSvcs {
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

func (s *Service) buildSimulatedCustomerProfile(ctx context.Context, namespace string, simulatedCustomer customer.Customer) (billing.CustomerOverrideWithDetails, error) {
	profile, err := s.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
		Namespace: namespace,
	})
	if err != nil {
		return billing.CustomerOverrideWithDetails{}, fmt.Errorf("getting default profile: %w", err)
	}

	return billing.CustomerOverrideWithDetails{
		MergedProfile: *profile,
		Customer:      &simulatedCustomer,
	}, nil
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

		_, err := s.updateInvoice(ctx, invoice)
		if err != nil {
			return fmt.Errorf("updating invoice[%s]: %w", invoice.ID, err)
		}

		return nil
	})
}

func (s *Service) RecalculateGatheringInvoices(ctx context.Context, input billing.RecalculateGatheringInvoicesInput) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		gatheringInvoices, err := s.adapter.ListInvoices(ctx, billing.ListInvoicesInput{
			Namespaces: []string{input.Namespace},
			Customers:  []string{input.ID},
			Statuses:   []string{string(billing.InvoiceStatusGathering)},
			Expand:     billing.InvoiceExpand{Lines: true},
		})
		if err != nil {
			return fmt.Errorf("listing invoices: %w", err)
		}

		for _, invoice := range gatheringInvoices.Items {
			var err error

			if err = s.invoiceCalculator.CalculateGatheringInvoice(&invoice); err != nil {
				return fmt.Errorf("calculating gathering invoice: %w", err)
			}

			invoiceID := invoice.ID

			invoice, err = s.updateInvoice(ctx, invoice)
			if err != nil {
				return fmt.Errorf("updating invoice[%s]: %w", invoiceID, err)
			}

			if invoice.Lines.NonDeletedLineCount() == 0 {
				if err := s.adapter.DeleteGatheringInvoices(ctx, billing.DeleteGatheringInvoicesInput{
					Namespace:  input.Namespace,
					InvoiceIDs: []string{invoiceID},
				}); err != nil {
					return fmt.Errorf("deleting gathering invoice: %w", err)
				}
			}
		}

		return nil
	})
}
