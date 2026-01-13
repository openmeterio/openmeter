package billingservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/service/lineservice"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

var _ billing.InvoiceLineService = (*Service)(nil)

func (s *Service) CreatePendingInvoiceLines(ctx context.Context, input billing.CreatePendingInvoiceLinesInput) (*billing.CreatePendingInvoiceLinesResult, error) {
	for i := range input.Lines {
		input.Lines[i].Namespace = input.Customer.Namespace
		input.Lines[i].Currency = input.Currency
	}

	cust, err := s.customerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &input.Customer,
	})
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	maxPeriodEnd := lo.FromPtr(cust.DeletedAt)
	if !maxPeriodEnd.IsZero() {
		var errs []error
		for _, line := range input.Lines {
			if line.Period.End.After(maxPeriodEnd) {
				errs = append(errs, fmt.Errorf("line[%s]: line period end[%s] is after customer deleted at[%s]", line.ID, line.Period.End, maxPeriodEnd))
			}
		}

		if len(errs) > 0 {
			return nil, billing.ValidationError{
				Err: errors.Join(errs...),
			}
		}
	}

	return transcationForInvoiceManipulation(ctx, s, input.Customer, func(ctx context.Context) (*billing.CreatePendingInvoiceLinesResult, error) {
		lineServices, err := s.lineService.FromEntities(lo.Map(input.Lines, func(l *billing.Line, _ int) *billing.Line {
			l.Namespace = input.Customer.Namespace
			l.Currency = input.Currency

			// This is only used to ensure that we know the line IDs before the upsert so that we can return
			// the correct lines to the caller.
			l.ID = ulid.Make().String()

			return l
		}))
		if err != nil {
			return nil, fmt.Errorf("creating line services: %w", err)
		}

		if len(lineServices) == 0 {
			return nil, nil
		}

		// let's resolve the customer's settings
		customerProfile, err := s.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Customer: input.Customer,
			Expand: billing.CustomerOverrideExpand{
				Customer: true,
				Apps:     true,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("fetching customer profile: %w", err)
		}

		gatheringInvoiceUpsertResult, err := s.upsertGatheringInvoiceForCurrency(ctx, input.Currency, customerProfile)
		if err != nil {
			return nil, fmt.Errorf("upserting gathering invoice: %w", err)
		}

		if gatheringInvoiceUpsertResult.Invoice == nil {
			return nil, fmt.Errorf("gathering invoice is nil")
		}
		gatheringInvoice := *gatheringInvoiceUpsertResult.Invoice

		lines := make(lineservice.Lines, 0, len(input.Lines))

		for i, lineSvc := range lineServices {
			line := lineSvc.ToEntity()
			line.InvoiceID = gatheringInvoice.ID

			if err := lineSvc.Validate(ctx, &gatheringInvoice); err != nil {
				return nil, fmt.Errorf("validating line[%d]: %w", i, err)
			}

			lineSvc, err = lineSvc.PrepareForCreate(ctx)
			if err != nil {
				return nil, fmt.Errorf("modifying line[%d]: %w", i, err)
			}

			lines = append(lines, lineSvc)
		}

		linesToCreate := lines.ToEntities()

		gatheringInvoice.Lines.Append(linesToCreate...)

		gatheringInvoice, err = s.adapter.UpdateInvoice(ctx, gatheringInvoice)
		if err != nil {
			return nil, fmt.Errorf("updating invoice: %w", err)
		}

		gatheringInvoiceID := gatheringInvoice.ID

		if err := s.invoiceCalculator.CalculateGatheringInvoice(&gatheringInvoice); err != nil {
			return nil, fmt.Errorf("calculating invoice[%s]: %w", gatheringInvoiceID, err)
		}

		gatheringInvoice, err = s.adapter.UpdateInvoice(ctx, gatheringInvoice)
		if err != nil {
			return nil, fmt.Errorf("failed to update invoice[%s]: %w", gatheringInvoiceID, err)
		}

		gatheringInvoice, err = s.resolveWorkflowApps(ctx, gatheringInvoice)
		if err != nil {
			return nil, fmt.Errorf("error resolving workflow apps for invoice [%s]: %w", gatheringInvoiceID, err)
		}

		// Let's resolve the created lines from the final invoice
		invoiceLinesByID := lo.SliceToMap(gatheringInvoice.Lines.OrEmpty(), func(l *billing.Line) (string, *billing.Line) {
			return l.ID, l
		})

		finalLines := []*billing.Line{}
		for _, line := range linesToCreate {
			if line, ok := invoiceLinesByID[line.ID]; ok {
				finalLines = append(finalLines, line)
			}
		}

		// Publish system event for newly created invoices
		if gatheringInvoiceUpsertResult.IsInvoiceNew {
			event, err := billing.NewInvoiceCreatedEvent(gatheringInvoice)
			if err != nil {
				return nil, fmt.Errorf("creating event: %w", err)
			}

			if err := s.publisher.Publish(ctx, event); err != nil {
				return nil, fmt.Errorf("publishing invoice[%s] created event: %w", gatheringInvoiceID, err)
			}
		}

		return &billing.CreatePendingInvoiceLinesResult{
			Invoice:      gatheringInvoice,
			IsInvoiceNew: gatheringInvoiceUpsertResult.IsInvoiceNew,
			Lines:        finalLines,
		}, nil
	})
}

type upsertGatheringInvoiceForCurrencyResponse struct {
	Invoice      *billing.Invoice
	IsInvoiceNew bool
}

func (s *Service) upsertGatheringInvoiceForCurrency(ctx context.Context, currency currencyx.Code, customerProfile billing.CustomerOverrideWithDetails) (*upsertGatheringInvoiceForCurrencyResponse, error) {
	// We would want to stage a pending invoice Line
	pendingInvoiceList, err := s.adapter.ListInvoices(ctx, billing.ListInvoicesInput{
		Page: pagination.Page{
			PageNumber: 1,
			PageSize:   10,
		},
		Customers:        []string{customerProfile.Customer.ID},
		Namespaces:       []string{customerProfile.Customer.Namespace},
		ExtendedStatuses: []billing.InvoiceStatus{billing.InvoiceStatusGathering},
		Currencies:       []currencyx.Code{currency},
		OrderBy:          api.InvoiceOrderByCreatedAt,
		Order:            sortx.OrderAsc,
		IncludeDeleted:   true,
		Expand: billing.InvoiceExpand{
			Lines: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("fetching gathering invoices: %w", err)
	}

	if len(pendingInvoiceList.Items) == 0 {
		invoiceNumber, err := s.GenerateInvoiceSequenceNumber(ctx,
			billing.SequenceGenerationInput{
				Namespace:    customerProfile.Customer.Namespace,
				CustomerName: customerProfile.Customer.Name,
				Currency:     currency,
			},
			billing.GatheringInvoiceSequenceNumber)
		if err != nil {
			return nil, fmt.Errorf("generating invoice sequence number: %w", err)
		}

		// Create a new invoice
		invoice, err := s.adapter.CreateInvoice(ctx, billing.CreateInvoiceAdapterInput{
			Namespace: customerProfile.Customer.Namespace,
			Customer:  lo.FromPtr(customerProfile.Customer),
			Profile:   customerProfile.MergedProfile,
			Number:    invoiceNumber,
			Currency:  currency,
			Status:    billing.InvoiceStatusGathering,
			Type:      billing.InvoiceTypeStandard,
		})
		if err != nil {
			return nil, fmt.Errorf("creating invoice: %w", err)
		}

		return &upsertGatheringInvoiceForCurrencyResponse{
			Invoice:      &invoice,
			IsInvoiceNew: true,
		}, nil
	}

	invoice := pendingInvoiceList.Items[0]
	if invoice.DeletedAt != nil {
		invoice.DeletedAt = nil

		// If the invoice was deleted, but has lines, we need to delete those lines to prevent
		// them from being associated with the deleted invoice.
		if invoice.Lines.NonDeletedLineCount() > 0 {
			invoice.Lines = invoice.Lines.Map(func(l *billing.Line) *billing.Line {
				if l.DeletedAt == nil {
					l.DeletedAt = lo.ToPtr(clock.Now())
				}
				return l
			})
		}

		invoiceID := invoice.ID

		invoice, err = s.adapter.UpdateInvoice(ctx, invoice)
		if err != nil {
			return nil, fmt.Errorf("restoring deleted invoice[id=%s]: %w", invoiceID, err)
		}
	}

	return &upsertGatheringInvoiceForCurrencyResponse{
		Invoice: &invoice,
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

			if splitLine.PreSplitAtLine == nil {
				s.logger.WarnContext(ctx, "pre split line is nil, we are not creating empty lines", "line", line.ID(), "period_start", line.Period().Start, "period_end", line.Period().End, "period_end", line.Period().End)
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

	// Let's create the sub lines as per the meters (we are not setting the QuantitySnapshotedAt field just now, to signal that this is not the final snapshot)
	if err := s.snapshotLineQuantitiesInParallel(ctx, invoice.Customer, invoiceLines); err != nil {
		return invoice, fmt.Errorf("snapshotting lines: %w", err)
	}

	// Let's active the invoice state machine so that calculations can be done
	return s.WithInvoiceStateMachine(ctx, invoice, func(ctx context.Context, ism *InvoiceStateMachine) error {
		return ism.StateMachine.ActivateCtx(ctx)
	})
}

func (s *Service) snapshotLineQuantitiesInParallel(ctx context.Context, customer billing.InvoiceCustomer, lines lineservice.Lines) error {
	linesCh := make(chan lineservice.Line, len(lines))
	errCh := make(chan error, len(lines))
	doneCh := make(chan struct{})

	// Feed the channel
	for _, line := range lines {
		linesCh <- line
	}
	close(linesCh)

	// Start workers
	for range s.maxParallelQuantitySnapshots {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					errCh <- fmt.Errorf("snapshotting line quantity: %v", r)
				}
				doneCh <- struct{}{}
			}()

			for line := range linesCh {
				if ctx.Err() != nil {
					errCh <- ctx.Err()
					return
				}
				if err := line.SnapshotQuantity(ctx, customer); err != nil {
					errCh <- fmt.Errorf("line[%s]: snapshotting quantity: %w", line.ID(), err)
				}
			}
		}()
	}

	// Wait for all workers to finish
	for range s.maxParallelQuantitySnapshots {
		<-doneCh
	}

	close(errCh)

	// Collect snapshot errors
	errs := []error{}
	for err := range errCh {
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (s *Service) GetLinesForSubscription(ctx context.Context, input billing.GetLinesForSubscriptionInput) ([]billing.LineOrHierarchy, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) ([]billing.LineOrHierarchy, error) {
		return s.adapter.GetLinesForSubscription(ctx, input)
	})
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

	err = lineSvc.SnapshotQuantity(ctx, input.Invoice.Customer)
	if err != nil {
		return nil, fmt.Errorf("snapshotting line quantity: %w", err)
	}

	return lineSvc.ToEntity(), nil
}
