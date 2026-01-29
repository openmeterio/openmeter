package billingservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/oklog/ulid/v2"
	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/sortx"
	"github.com/samber/lo"
)

var _ billing.InvoiceLineService = (*Service)(nil)

// TODO[SeperatePR]: Move this to gatheringinvoice.go

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
			if line.ServicePeriod.To.After(maxPeriodEnd) {
				errs = append(errs, fmt.Errorf("line[%s]: line period end[%s] is after customer deleted at[%s]", line.ID, line.ServicePeriod.To, maxPeriodEnd))
			}
		}

		if len(errs) > 0 {
			return nil, billing.ValidationError{
				Err: errors.Join(errs...),
			}
		}
	}

	return transcationForInvoiceManipulation(ctx, s, input.Customer, func(ctx context.Context) (*billing.CreatePendingInvoiceLinesResult, error) {
		if len(input.Lines) == 0 {
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

		gatheringInvoice := gatheringInvoiceUpsertResult.Invoice

		linesToCreate, err := slicesx.MapWithErr(input.Lines, func(l billing.GatheringLine) (billing.GatheringLine, error) {
			l.Namespace = input.Customer.Namespace
			l.Currency = input.Currency

			// This is only used to ensure that we know the line IDs before the upsert so that we can return
			// the correct lines to the caller.
			l.ID = ulid.Make().String()
			l.InvoiceID = gatheringInvoice.ID

			normalizedLine, err := l.WithNormalizedValues()
			if err != nil {
				return billing.GatheringLine{}, fmt.Errorf("normalizing line[%s]: %w", l.ID, err)
			}

			if err := normalizedLine.Validate(); err != nil {
				return billing.GatheringLine{}, fmt.Errorf("validating line[%s]: %w", l.ID, err)
			}

			return normalizedLine, nil
		})
		if err != nil {
			return nil, fmt.Errorf("mapping lines: %w", err)
		}

		gatheringInvoice.Lines.Append(linesToCreate...)
		gatheringInvoiceID := gatheringInvoice.ID

		if err := gatheringInvoice.Validate(); err != nil {
			return nil, fmt.Errorf("validating gathering invoice: %w", err)
		}

		if err := s.invoiceCalculator.CalculateGatheringInvoice(&gatheringInvoice); err != nil {
			return nil, fmt.Errorf("calculating invoice[%s]: %w", gatheringInvoiceID, err)
		}

		gatheringInvoice, err = s.adapter.UpdateGatheringInvoice(ctx, gatheringInvoice)
		if err != nil {
			return nil, fmt.Errorf("failed to update invoice[%s]: %w", gatheringInvoiceID, err)
		}

		// Let's resolve the created lines from the final invoice
		invoiceLinesByID := lo.SliceToMap(gatheringInvoice.Lines.OrEmpty(), func(l billing.GatheringLine) (string, billing.GatheringLine) {
			return l.ID, l
		})

		finalLines := []billing.GatheringLine{}
		for _, line := range linesToCreate {
			if line, ok := invoiceLinesByID[line.ID]; ok {
				finalLines = append(finalLines, line)
			}
		}

		// Publish system event for newly created invoices
		if gatheringInvoiceUpsertResult.IsInvoiceNew {
			event, err := billing.NewStandardInvoiceCreatedEvent(gatheringInvoice)
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
	Invoice      billing.GatheringInvoice
	IsInvoiceNew bool
}

func (s *Service) upsertGatheringInvoiceForCurrency(ctx context.Context, currency currencyx.Code, customerProfile billing.CustomerOverrideWithDetails) (*upsertGatheringInvoiceForCurrencyResponse, error) {
	// We would want to stage a pending invoice Line
	pendingInvoiceList, err := s.adapter.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
		Page: pagination.Page{
			PageNumber: 1,
			PageSize:   10,
		},
		Customers:      []string{customerProfile.Customer.ID},
		Namespaces:     []string{customerProfile.Customer.Namespace},
		Currencies:     []currencyx.Code{currency},
		OrderBy:        api.InvoiceOrderByCreatedAt,
		Order:          sortx.OrderAsc,
		IncludeDeleted: true,
		Expand:         []billing.GatheringInvoiceExpand{billing.GatheringInvoiceExpandLines},
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
		invoice, err := s.adapter.CreateGatheringInvoice(ctx, billing.CreateGatheringInvoiceAdapterInput{
			Namespace: customerProfile.Customer.Namespace,
			Customer:  lo.FromPtr(customerProfile.Customer),
			Number:    invoiceNumber,
			Currency:  currency,
		})
		if err != nil {
			return nil, fmt.Errorf("creating invoice: %w", err)
		}

		return &upsertGatheringInvoiceForCurrencyResponse{
			Invoice:      invoice,
			IsInvoiceNew: true,
		}, nil
	}

	invoice := pendingInvoiceList.Items[0]
	if invoice.DeletedAt != nil {
		invoice.DeletedAt = nil

		// If the invoice was deleted, but has non-deleted lines, we need to delete those lines to prevent
		// them from reappearing in the recreated gathering invoice.
		if invoice.Lines.NonDeletedLineCount() > 0 {
			invoice.Lines = invoice.Lines.Map(func(l billing.GatheringLine) billing.GatheringLine {
				if l.DeletedAt == nil {
					l.DeletedAt = lo.ToPtr(clock.Now())
				}
				return l
			})
		}

		invoiceID := invoice.ID

		invoice, err = s.adapter.UpdateGatheringInvoice(ctx, invoice)
		if err != nil {
			return nil, fmt.Errorf("restoring deleted invoice[id=%s]: %w", invoiceID, err)
		}
	}

	return &upsertGatheringInvoiceForCurrencyResponse{
		Invoice: invoice,
	}, nil
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
