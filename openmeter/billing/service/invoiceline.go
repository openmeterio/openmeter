package billingservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/service/lineservice"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

var _ billing.InvoiceLineService = (*Service)(nil)

func (s *Service) CreatePendingInvoiceLines(ctx context.Context, input billing.CreateInvoiceLinesInput) ([]*billing.Line, error) {
	for i := range input.Lines {
		input.Lines[i].Namespace = input.Namespace
		input.Lines[i].Status = billing.InvoiceLineStatusValid

		if input.Lines[i].InvoiceID != "" {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("line[%s]: invoice ID is not allowed for pending lines", input.Lines[i].ID),
			}
		}
	}

	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	// Let's execute the change on a by customer basis. Later we can optimize this to be done in a single batch, but
	// first we need to see if multiple customer staging of invoices is a common use case.

	createByCustomerID := lo.GroupBy(input.Lines, func(line billing.LineWithCustomer) string {
		return line.CustomerID
	})

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) ([]*billing.Line, error) {
		out := make([]*billing.Line, 0, len(input.Lines))
		newInvoiceIDs := []string{}

		for customerID, lineByCustomer := range createByCustomerID {
			if err := s.validateCustomerForUpdate(ctx, customerentity.CustomerID{
				Namespace: input.Namespace,
				ID:        customerID,
			}); err != nil {
				return nil, err
			}

			createdLines, err := TranscationForGatheringInvoiceManipulation(
				ctx,
				s,
				customerentity.CustomerID{
					Namespace: input.Namespace,
					ID:        customerID,
				},
				func(ctx context.Context) ([]*billing.Line, error) {
					// let's resolve the customer's settings
					customerProfile, err := s.GetProfileWithCustomerOverride(ctx, billing.GetProfileWithCustomerOverrideInput{
						Namespace:  input.Namespace,
						CustomerID: customerID,
					})
					if err != nil {
						return nil, fmt.Errorf("fetching customer profile: %w", err)
					}

					lines := make(lineservice.Lines, 0, len(lineByCustomer))

					// TODO[OM-949]: we should optimize this as this does O(n) queries for invoices per line
					for i, line := range lineByCustomer {
						updateResult, err := s.upsertLineInvoice(ctx, line.Line, input, customerProfile)
						if err != nil {
							return nil, fmt.Errorf("upserting line[%d]: %w", i, err)
						}

						if updateResult.IsInvoiceNew {
							newInvoiceIDs = append(newInvoiceIDs, updateResult.Invoice.ID)
						}

						lineService, err := s.lineService.FromEntity(&updateResult.Line)
						if err != nil {
							return nil, fmt.Errorf("creating line service[%d]: %w", i, err)
						}

						if err := lineService.Validate(ctx, updateResult.Invoice); err != nil {
							return nil, fmt.Errorf("validating line[%s]: %w", input.Lines[i].ID, err)
						}

						lineService, err = lineService.PrepareForCreate(ctx)
						if err != nil {
							return nil, fmt.Errorf("modifying line[%s]: %w", input.Lines[i].ID, err)
						}

						lines = append(lines, lineService)
					}

					// Create the invoice Lines
					createdLines, err := s.adapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
						Namespace: input.Namespace,
						Lines:     lines.ToEntities(),
					})
					if err != nil {
						return nil, fmt.Errorf("creating invoice Line: %w", err)
					}

					return createdLines, nil
				})
			if err != nil {
				return nil, err
			}

			out = append(out, createdLines...)
		}

		for _, invoiceID := range newInvoiceIDs {
			invoice, err := s.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
				Invoice: billing.InvoiceID{
					Namespace: input.Namespace,
					ID:        invoiceID,
				},
				Expand: billing.InvoiceExpandAll,
			})
			if err != nil {
				return nil, fmt.Errorf("fetching invoice[%s]: %w", invoiceID, err)
			}

			if err := s.publisher.Publish(ctx, billing.NewInvoiceCreatedEvent(invoice)); err != nil {
				return nil, fmt.Errorf("publishing invoice[%s] created event: %w", invoiceID, err)
			}
		}

		return out, nil
	})
}

type upsertLineInvoiceResponse struct {
	Line         billing.Line
	Invoice      *billing.Invoice
	IsInvoiceNew bool
}

func (s *Service) upsertLineInvoice(ctx context.Context, line billing.Line, input billing.CreateInvoiceLinesInput, customerProfile *billing.ProfileWithCustomerDetails) (*upsertLineInvoiceResponse, error) {
	// We would want to stage a pending invoice Line
	pendingInvoiceList, err := s.adapter.ListInvoices(ctx, billing.ListInvoicesInput{
		Page: pagination.Page{
			PageNumber: 1,
			PageSize:   10,
		},
		Customers:        []string{customerProfile.Customer.ID},
		Namespaces:       []string{input.Namespace},
		ExtendedStatuses: []billing.InvoiceStatus{billing.InvoiceStatusGathering},
		Currencies:       []currencyx.Code{line.Currency},
		OrderBy:          api.InvoiceOrderByCreatedAt,
		Order:            sortx.OrderAsc,
	})
	if err != nil {
		return nil, fmt.Errorf("fetching gathering invoices: %w", err)
	}

	if len(pendingInvoiceList.Items) == 0 {
		// Create a new invoice
		invoice, err := s.adapter.CreateInvoice(ctx, billing.CreateInvoiceAdapterInput{
			Namespace: input.Namespace,
			Customer:  customerProfile.Customer,
			Profile:   customerProfile.Profile,
			Currency:  line.Currency,
			Status:    billing.InvoiceStatusGathering,
			Type:      billing.InvoiceTypeStandard,
		})
		if err != nil {
			return nil, fmt.Errorf("creating invoice: %w", err)
		}

		line.InvoiceID = invoice.ID

		return &upsertLineInvoiceResponse{
			Line:         line,
			Invoice:      &invoice,
			IsInvoiceNew: true,
		}, nil
	}

	// Attach to the first pending invoice
	line.InvoiceID = pendingInvoiceList.Items[0].ID

	if len(pendingInvoiceList.Items) > 1 {
		// Note: Given that we are not using serializable transactions (which is fine), we might
		// have multiple gathering invoices for the same customer.
		// This is a rare case, but we should log it at least, later we can implement a call that
		// merges these invoices (it's fine to just move the Lines to the first invoice)
		s.logger.WarnContext(ctx, "more than one pending invoice found", "customer", customerProfile.Customer.ID, "namespace", input.Namespace)
	}

	return &upsertLineInvoiceResponse{
		Line:    line,
		Invoice: &pendingInvoiceList.Items[0],
	}, nil
}

func (s *Service) associateLinesToInvoice(ctx context.Context, invoice billing.Invoice, lines []lineservice.LineWithBillablePeriod) (billing.Invoice, error) {
	for _, line := range lines {
		if line.InvoiceID() == invoice.ID {
			return invoice, billing.ValidationError{
				Err: fmt.Errorf("line[%s]: line already associated with invoice[%s]", line.ID(), invoice.ID),
			}
		}
	}

	invoiceLines := make(lineservice.Lines, 0, len(lines))
	// Let's do the line splitting if needed
	for _, line := range lines {
		if !line.Period().Equal(line.BillablePeriod) {
			// We need to split the line into multiple lines
			if !line.Period().Start.Equal(line.BillablePeriod.Start) {
				return invoice, fmt.Errorf("line[%s]: line period start[%s] is not equal to billable period start[%s]", line.ID(), line.Period().Start, line.BillablePeriod.Start)
			}

			splitLine, err := line.Split(ctx, line.BillablePeriod.End)
			if err != nil {
				return invoice, fmt.Errorf("line[%s]: splitting line: %w", line.ID(), err)
			}

			invoiceLines = append(invoiceLines, splitLine.PreSplitAtLine)
		} else {
			invoiceLines = append(invoiceLines, line)
		}
	}

	// Validate that the line can be associated with the invoice
	var validationErrors error
	for _, line := range invoiceLines {
		if err := line.Validate(ctx, &invoice); err != nil {
			validationErrors = fmt.Errorf("line[%s]: %w", line.ID(), err)
		}
	}
	if validationErrors != nil {
		return invoice, validationErrors
	}

	// Associate the lines to the invoice
	invoiceLines, err := s.lineService.AssociateLinesToInvoice(ctx, &invoice, invoiceLines)
	if err != nil {
		return invoice, fmt.Errorf("associating lines to invoice: %w", err)
	}

	// Let's create the sub lines as per the meters
	for _, line := range invoiceLines {
		if err := line.SnapshotQuantity(ctx, &invoice); err != nil {
			return invoice, fmt.Errorf("line[%s]: snapshotting quantity: %w", line.ID(), err)
		}
	}

	// Let's active the invoice state machine so that calculations can be done
	return s.WithInvoiceStateMachine(ctx, invoice, func(ctx context.Context, ism *InvoiceStateMachine) error {
		return ism.StateMachine.ActivateCtx(ctx)
	})
}

func (s *Service) GetInvoiceLine(ctx context.Context, input billing.GetInvoiceLineInput) (*billing.Line, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return s.adapter.GetInvoiceLine(ctx, billing.GetInvoiceLineAdapterInput{
		Namespace: input.Namespace,
		ID:        input.ID,
	})
}

func (s *Service) UpdateInvoiceLine(ctx context.Context, input billing.UpdateInvoiceLineInput) (*billing.Line, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	// Let's get the customer's active billing profile
	lineOwnership, err := s.adapter.GetInvoiceLineOwnership(ctx, input.Line)
	if err != nil {
		return nil, err
	}

	billingProfile, err := s.GetProfileWithCustomerOverride(ctx, billing.GetProfileWithCustomerOverrideInput{
		Namespace:  lineOwnership.Namespace,
		CustomerID: lineOwnership.CustomerID,
	})
	if err != nil {
		return nil, fmt.Errorf("fetching customer profile: %w", err)
	}

	triggeredInvoice, err := transaction.Run(ctx, s.adapter, func(ctx context.Context) (*billing.Invoice, error) {
		existingLine, err := s.adapter.GetInvoiceLine(ctx, input.Line)
		if err != nil {
			return nil, err
		}

		invoice, err := s.executeTriggerOnInvoice(
			ctx,
			billing.InvoiceID{
				ID:        existingLine.InvoiceID,
				Namespace: existingLine.Namespace,
			},
			triggerUpdated,
			ExecuteTriggerWithAllowInStates(billing.InvoiceStatusDraftUpdating),
			ExecuteTriggerWithEditCallback(func(sm *InvoiceStateMachine) error {
				targetState, err := input.Apply(existingLine)
				if err != nil {
					return err
				}

				if err := targetState.Validate(); err != nil {
					return billing.ValidationError{
						Err: err,
					}
				}

				targetStateLineSrv, err := s.lineService.FromEntity(targetState)
				if err != nil {
					return fmt.Errorf("creating line service: %w", err)
				}

				period, err := targetStateLineSrv.CanBeInvoicedAsOf(ctx, lineservice.CanBeInvoicedAsOfInput{
					AsOf:               targetState.Period.End,
					ProgressiveBilling: billingProfile.Profile.WorkflowConfig.Invoicing.ProgressiveBilling,
				})
				if err != nil {
					return fmt.Errorf("line[%s]: can be invoiced as of: %w", targetState.ID, err)
				}

				if period == nil {
					return billing.ValidationError{
						Err: fmt.Errorf("line[%s]: %w as of %s", targetState.ID, billing.ErrInvoiceLinesNotBillable, targetState.Period.End),
					}
				}

				if ok := sm.Invoice.Lines.ReplaceByID(existingLine.ID, targetState); !ok {
					return fmt.Errorf("line[%s]: not found in invoice", existingLine.ID)
				}

				return nil
			}),
		)

		return &invoice, err
	})
	if err != nil {
		return nil, err
	}

	invoice, err := s.AdvanceInvoice(ctx, triggeredInvoice.InvoiceID())
	// We don't care if we cannot advance the invoice, as most probably we ended up in a failed state
	if errors.Is(err, billing.ErrInvoiceCannotAdvance) {
		invoice = *triggeredInvoice
	} else if err != nil {
		return nil, fmt.Errorf("advancing invoice: %w", err)
	}

	updatedLine := lo.FindOrElse(invoice.Lines.OrEmpty(), nil, func(l *billing.Line) bool {
		return l.ID == input.Line.ID
	})

	if updatedLine == nil {
		return nil, billing.NotFoundError{
			Err: fmt.Errorf("line[%s]: not found in invoice", input.Line.ID),
		}
	}

	return updatedLine, nil
}

func (s *Service) DeleteInvoiceLine(ctx context.Context, input billing.DeleteInvoiceLineInput) error {
	if err := input.Validate(); err != nil {
		return billing.ValidationError{
			Err: err,
		}
	}

	existingLine, err := s.adapter.GetInvoiceLine(ctx, input)
	if err != nil {
		return err
	}

	if existingLine.Status != billing.InvoiceLineStatusValid {
		return billing.ValidationError{
			Err: fmt.Errorf("line[%s]: %w", existingLine.ID, billing.ErrInvoiceLineDeleteInvalidStatus),
		}
	}

	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		_, err := s.executeTriggerOnInvoice(
			ctx,
			billing.InvoiceID{
				ID:        existingLine.InvoiceID,
				Namespace: existingLine.Namespace,
			},
			triggerUpdated,
			ExecuteTriggerWithAllowInStates(billing.InvoiceStatusDraftUpdating),
			ExecuteTriggerWithEditCallback(func(sm *InvoiceStateMachine) error {
				line := sm.Invoice.Lines.GetByID(existingLine.ID)
				if line == nil || line.DeletedAt != nil {
					return billing.NotFoundError{
						Err: fmt.Errorf("line[%s]: not found in invoice", existingLine.ID),
					}
				}

				line.DeletedAt = lo.ToPtr(clock.Now())

				return nil
			}),
		)

		return err
	})
}

func (s *Service) GetLinesForSubscription(ctx context.Context, input billing.GetLinesForSubscriptionInput) ([]*billing.Line, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return s.adapter.GetLinesForSubscription(ctx, input)
}

func (s *Service) SnapshotLineQuantity(ctx context.Context, input billing.SnapshotLineQuantityInput) (*billing.Line, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	lineSvc, err := s.lineService.FromEntity(input.Line)
	if err != nil {
		return nil, fmt.Errorf("creating line service: %w", err)
	}

	err = lineSvc.SnapshotQuantity(ctx, input.Invoice)
	if err != nil {
		return nil, fmt.Errorf("snapshotting line quantity: %w", err)
	}

	return lineSvc.ToEntity(), nil
}
