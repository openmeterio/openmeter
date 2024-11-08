package billingservice

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	lineservice "github.com/openmeterio/openmeter/openmeter/billing/service/lineservice"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

var _ billing.InvoiceLineService = (*Service)(nil)

func (s *Service) CreateInvoiceLines(ctx context.Context, input billing.CreateInvoiceLinesInput) (*billing.CreateInvoiceLinesResponse, error) {
	for i := range input.Lines {
		input.Lines[i].Namespace = input.Namespace
		input.Lines[i].Status = billingentity.InvoiceLineStatusValid
	}

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

			lineSrv, err := s.newLineService(txAdapter)
			if err != nil {
				return nil, fmt.Errorf("creating line service: %w", err)
			}

			lines := make(lineservice.Lines, 0, len(input.Lines))

			// TODO[OM-949]: we should optimize this as this does O(n) queries for invoices per line
			for i, line := range input.Lines {
				line.Namespace = input.Namespace

				updateResult, err := s.upsertLineInvoice(ctx, txAdapter, line, input, customerProfile)
				if err != nil {
					return nil, fmt.Errorf("upserting line[%d]: %w", i, err)
				}

				lineService, err := lineSrv.FromEntity(updateResult.Line)
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
			createdLines, err := txAdapter.CreateInvoiceLines(ctx, lines.ToEntities())
			if err != nil {
				return nil, fmt.Errorf("creating invoice Line: %w", err)
			}

			return createdLines, nil
		})
}

type upsertLineInvoiceResponse struct {
	Line    billingentity.Line
	Invoice *billingentity.Invoice
}

func (s *Service) upsertLineInvoice(ctx context.Context, txAdapter billing.Adapter, line billingentity.Line, input billing.CreateInvoiceLinesInput, customerProfile *billingentity.ProfileWithCustomerDetails) (*upsertLineInvoiceResponse, error) {
	if line.InvoiceID != "" {
		// We would want to attach the line to an existing invoice
		invoice, err := txAdapter.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
			Invoice: billingentity.InvoiceID{
				ID:        line.InvoiceID,
				Namespace: input.Namespace,
			},
		})
		if err != nil {
			return nil, billingentity.ValidationError{
				Err: fmt.Errorf("fetching invoice [%s]: %w", line.InvoiceID, err),
			}
		}

		if !invoice.StatusDetails.Immutable {
			return nil, billingentity.ValidationError{
				Err: fmt.Errorf("invoice [%s] is not mutable", line.InvoiceID),
			}
		}

		if invoice.Currency != line.Currency {
			return nil, billingentity.ValidationError{
				Err: fmt.Errorf("currency mismatch: invoice [%s] currency is %s, but line currency is %s", line.InvoiceID, invoice.Currency, line.Currency),
			}
		}

		line.InvoiceID = invoice.ID
		return &upsertLineInvoiceResponse{
			Line:    line,
			Invoice: &invoice,
		}, nil
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
		return nil, fmt.Errorf("fetching gathering invoices: %w", err)
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
			return nil, fmt.Errorf("creating invoice: %w", err)
		}

		line.InvoiceID = invoice.ID

		return &upsertLineInvoiceResponse{
			Line:    line,
			Invoice: &invoice,
		}, nil
	}

	// Attach to the first pending invoice
	line.InvoiceID = pendingInvoiceList.Items[0].ID

	if len(pendingInvoiceList.Items) > 1 {
		// Note: Given that we are not using serializable transactions (which is fine), we might
		// have multiple gathering invoices for the same customer.
		// This is a rare case, but we should log it at least, later we can implement a call that
		// merges these invoices (it's fine to just move the Lines to the first invoice)
		s.logger.Warn("more than one pending invoice found", "customer", input.CustomerID, "namespace", input.Namespace)
	}

	return &upsertLineInvoiceResponse{
		Line:    line,
		Invoice: &pendingInvoiceList.Items[0],
	}, nil
}

func (s *Service) associateLinesToInvoice(ctx context.Context, lineSrv *lineservice.Service, invoice billingentity.Invoice, lines []lineservice.LineWithBillablePeriod) error {
	for _, line := range lines {
		if line.InvoiceID() == invoice.ID {
			return billingentity.ValidationError{
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
				return fmt.Errorf("line[%s]: line period start[%s] is not equal to billable period start[%s]", line.ID(), line.Period().Start, line.BillablePeriod.Start)
			}

			splitLine, err := line.Split(ctx, line.BillablePeriod.End)
			if err != nil {
				return fmt.Errorf("line[%s]: splitting line: %w", line.ID(), err)
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
		return validationErrors
	}

	// Associate the lines to the invoice
	invoiceLines, err := lineSrv.AssociateLinesToInvoice(ctx, &invoice, invoiceLines)
	if err != nil {
		return fmt.Errorf("associating lines to invoice: %w", err)
	}

	// Let's create the sub lines as per the meters
	for _, line := range invoiceLines {
		snapshot, err := line.SnapshotQuantity(ctx, &invoice)
		if err != nil {
			return fmt.Errorf("line[%s]: snapshotting quantity: %w", line.ID(), err)
		}

		// TODO[OM-980]: detailed lines

		_, err = snapshot.Line.Save(ctx)
		if err != nil {
			return fmt.Errorf("line[%s]: saving line: %w", line.ID(), err)
		}
	}

	return nil
}

func (s *Service) newLineService(adapter billing.Adapter) (*lineservice.Service, error) {
	return lineservice.New(lineservice.Config{
		BillingAdapter:     adapter,
		FeatureService:     s.featureService,
		MeterRepo:          s.meterRepo,
		StreamingConnector: s.streamingConnector,
	})
}
