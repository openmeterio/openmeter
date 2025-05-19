package billingservice

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/service/lineservice"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

var _ billing.InvoiceLineService = (*Service)(nil)

func (s *Service) CreatePendingInvoiceLines(ctx context.Context, input billing.CreatePendingInvoiceLinesInput) (*billing.CreatePendingInvoiceLinesResult, error) {
	for i := range input.Lines {
		input.Lines[i].Namespace = input.Customer.Namespace
		input.Lines[i].Status = billing.InvoiceLineStatusValid
		input.Lines[i].Currency = input.Currency
	}

	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return transcationForInvoiceManipulation(ctx, s, input.Customer, func(ctx context.Context) (*billing.CreatePendingInvoiceLinesResult, error) {
		if err := s.validateCustomerForUpdate(ctx, input.Customer); err != nil {
			return nil, err
		}

		lineServices, err := s.lineService.FromEntities(lo.Map(input.Lines, func(l *billing.Line, _ int) *billing.Line {
			l.Namespace = input.Customer.Namespace
			l.Status = billing.InvoiceLineStatusValid
			l.Currency = input.Currency

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

		// Create the invoice Lines
		createdLines, err := s.adapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
			Namespace: input.Customer.Namespace,
			Lines:     lines.ToEntities(),
		})
		if err != nil {
			return nil, fmt.Errorf("creating invoice Line: %w", err)
		}

		// Let's reload the invoice after the lines has been assigned
		gatheringInvoice, err = s.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: gatheringInvoice.InvoiceID(),
			Expand:  billing.InvoiceExpandAll,
		})
		if err != nil {
			return nil, fmt.Errorf("fetching invoice[%s]: %w", gatheringInvoice.ID, err)
		}

		// Update invoice if collectionAt field has changed
		collectionConfig := customerProfile.MergedProfile.WorkflowConfig.Collection
		collectionAt := gatheringInvoice.CollectionAt
		if ok := UpdateInvoiceCollectionAt(&gatheringInvoice, collectionConfig); ok {
			s.logger.DebugContext(ctx, "collection time updated for invoice",
				"invoiceID", gatheringInvoice.ID,
				"collectionAt", map[string]interface{}{
					"from":               lo.FromPtr(collectionAt),
					"to":                 lo.FromPtr(gatheringInvoice.CollectionAt),
					"collectionInterval": collectionConfig.Interval.String(),
				},
			)
		}

		if err := s.invoiceCalculator.CalculateGatheringInvoice(&gatheringInvoice); err != nil {
			return nil, fmt.Errorf("calculating invoice[%s]: %w", gatheringInvoice.ID, err)
		}

		gatheringInvoice, err = s.adapter.UpdateInvoice(ctx, gatheringInvoice)
		if err != nil {
			return nil, fmt.Errorf("failed to update invoice[%s]: %w", gatheringInvoice.ID, err)
		}

		gatheringInvoice, err = s.resolveWorkflowApps(ctx, gatheringInvoice)
		if err != nil {
			return nil, fmt.Errorf("error resolving workflow apps for invoice [%s]: %w", gatheringInvoice.ID, err)
		}

		// Let's resolve the created lines from the final invoice
		invoiceLinesByID, _ := slicesx.UniqueGroupBy(gatheringInvoice.Lines.OrEmpty(), func(l *billing.Line) string {
			return l.ID
		})

		finalLines := []*billing.Line{}
		for _, line := range createdLines {
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
				return nil, fmt.Errorf("publishing invoice[%s] created event: %w", gatheringInvoice.ID, err)
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

	if len(pendingInvoiceList.Items) > 1 {
		// Note: Given that we are not using serializable transactions (which is fine), we might
		// have multiple gathering invoices for the same customer.
		// This is a rare case, but we should log it at least, later we can implement a call that
		// merges these invoices (it's fine to just move the Lines to the first invoice)
		s.logger.WarnContext(ctx, "more than one pending invoice found", "customer", customerProfile.Customer.ID, "namespace", customerProfile.Customer.Namespace, "currency", currency)
	}

	invoice := pendingInvoiceList.Items[0]
	if invoice.DeletedAt != nil {
		invoice.DeletedAt = nil

		invoice, err = s.adapter.UpdateInvoice(ctx, invoice)
		if err != nil {
			return nil, fmt.Errorf("restoring deleted invoice[id=%s]: %w", invoice.ID, err)
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

	// Let's create the sub lines as per the meters
	for _, line := range invoiceLines {
		if err := line.SnapshotQuantity(ctx, &invoice); err != nil {
			return invoice, fmt.Errorf("line[%s]: snapshotting quantity: %w", line.ID(), err)
		}
	}

	invoice.QuantitySnapshotedAt = lo.ToPtr(clock.Now().UTC())

	// Let's active the invoice state machine so that calculations can be done
	return s.WithInvoiceStateMachine(ctx, invoice, func(ctx context.Context, ism *InvoiceStateMachine) error {
		return ism.StateMachine.ActivateCtx(ctx)
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
