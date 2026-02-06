package billingservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/service/lineservice"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ billing.GatheringInvoiceService = (*Service)(nil)

func (s *Service) ListGatheringInvoices(ctx context.Context, input billing.ListGatheringInvoicesInput) (pagination.Result[billing.GatheringInvoice], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[billing.GatheringInvoice]{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (pagination.Result[billing.GatheringInvoice], error) {
		return s.adapter.ListGatheringInvoices(ctx, input)
	})
}

func (s *Service) emulateStandardInvoicesGatheringInvoiceFields(ctx context.Context, invoices []billing.StandardInvoice) ([]billing.StandardInvoice, error) {
	mergedProfiles := make(map[customer.CustomerID]billing.CustomerOverrideWithDetails)

	for idx := range invoices {
		invoice := &invoices[idx]

		if invoice.Status != billing.StandardInvoiceStatusGathering {
			continue
		}

		if _, ok := mergedProfiles[invoice.CustomerID()]; !ok {
			expand := billing.CustomerOverrideExpand{
				Customer: true,
				Apps:     true,
			}

			mergedProfile, err := s.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
				Customer: invoice.CustomerID(),
				Expand:   expand,
			})
			if err != nil {
				return nil, err
			}

			mergedProfiles[invoice.CustomerID()] = mergedProfile
		}

		mergedProfile := mergedProfiles[invoice.CustomerID()]

		invoice.Customer = billing.InvoiceCustomer{
			CustomerID:       invoice.CustomerID().ID,
			Name:             mergedProfile.Customer.Name,
			Key:              mergedProfile.Customer.Key,
			UsageAttribution: lo.ToPtr(mergedProfile.Customer.GetUsageAttribution()),
		}

		invoice.Supplier = mergedProfile.MergedProfile.Supplier

		invoice.Workflow = billing.InvoiceWorkflow{
			AppReferences:          lo.FromPtr(mergedProfile.MergedProfile.AppReferences),
			Apps:                   mergedProfile.MergedProfile.Apps,
			SourceBillingProfileID: mergedProfile.MergedProfile.ID,
			Config:                 mergedProfile.MergedProfile.WorkflowConfig,
		}
	}

	return invoices, nil
}

func (s *Service) emulateStandardInvoiceGatheringInvoiceFields(ctx context.Context, invoice billing.StandardInvoice) (billing.StandardInvoice, error) {
	invoices, err := s.emulateStandardInvoicesGatheringInvoiceFields(ctx, []billing.StandardInvoice{invoice})
	if err != nil {
		return billing.StandardInvoice{}, err
	}

	if len(invoices) != 1 {
		return billing.StandardInvoice{}, fmt.Errorf("expected 1 invoice, got %d", len(invoices))
	}

	return invoices[0], nil
}

func (s *Service) UpdateGatheringInvoice(ctx context.Context, input billing.UpdateGatheringInvoiceInput) error {
	if err := input.Validate(); err != nil {
		return billing.ValidationError{
			Err: err,
		}
	}

	gatheringInvoice, err := s.adapter.GetGatheringInvoiceById(ctx, billing.GetGatheringInvoiceByIdInput{
		Invoice: input.Invoice,
	})
	if err != nil {
		return fmt.Errorf("fetching invoice: %w", err)
	}

	return transcationForInvoiceManipulationNoValue(ctx, s, gatheringInvoice.GetCustomerID(), func(ctx context.Context) error {
		expands := billing.GatheringInvoiceExpands{
			billing.GatheringInvoiceExpandLines,
		}
		if input.IncludeDeletedLines {
			expands = expands.With(billing.GatheringInvoiceExpandDeletedLines)
		}

		invoice, err := s.adapter.GetGatheringInvoiceById(ctx, billing.GetGatheringInvoiceByIdInput{
			Invoice: input.Invoice,
			Expand:  expands,
		})
		if err != nil {
			return fmt.Errorf("fetching invoice: %w", err)
		}

		if err := input.EditFn(&invoice); err != nil {
			return fmt.Errorf("editing invoice: %w", err)
		}

		invoice.Lines, err = invoice.Lines.WithNormalizedValues()
		if err != nil {
			return fmt.Errorf("normalizing lines: %w", err)
		}

		if err := s.invoiceCalculator.CalculateGatheringInvoice(&invoice); err != nil {
			return fmt.Errorf("calculating invoice[%s]: %w", invoice.ID, err)
		}

		if err := invoice.Validate(); err != nil {
			return billing.ValidationError{
				Err: err,
			}
		}

		featureMeters, err := s.resolveFeatureMeters(ctx, invoice.Namespace, invoice.Lines)
		if err != nil {
			return fmt.Errorf("resolving feature meters: %w", err)
		}

		customerProfile, err := s.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Customer: invoice.GetCustomerID(),
		})
		if err != nil {
			return fmt.Errorf("fetching profile: %w", err)
		}

		// Check if the new lines are still invoicable
		if err := s.checkIfGatheringLinesAreInvoicable(ctx, invoice, customerProfile.MergedProfile.WorkflowConfig.Invoicing.ProgressiveBilling, featureMeters); err != nil {
			return err
		}

		err = s.adapter.UpdateGatheringInvoice(ctx, invoice)
		if err != nil {
			return fmt.Errorf("updating invoice[%s]: %w", input.Invoice.ID, err)
		}

		// Auto delete the invoice if it has no lines, this needs to happen here, as we are in a
		// TranscationForGatheringInvoiceManipulation

		if invoice.Lines.NonDeletedLineCount() == 0 {
			if err := s.adapter.DeleteGatheringInvoices(ctx, billing.DeleteGatheringInvoicesInput{
				Namespace:  input.Invoice.Namespace,
				InvoiceIDs: []string{invoice.ID},
			}); err != nil {
				return fmt.Errorf("deleting gathering invoice: %w", err)
			}
		}

		return nil
	})
}

func (s Service) checkIfGatheringLinesAreInvoicable(ctx context.Context, invoice billing.GatheringInvoice, progressiveBilling bool, featureMeters billing.FeatureMeters) error {
	linesToCheck := lo.Filter(invoice.Lines.OrEmpty(), func(line billing.GatheringLine, _ int) bool {
		return line.DeletedAt == nil
	})

	return errors.Join(
		lo.Map(linesToCheck, func(line billing.GatheringLine, _ int) error {
			if err := line.Validate(); err != nil {
				return fmt.Errorf("validating line[%s]: %w", line.ID, err)
			}
			period, err := lineservice.ResolveBillablePeriod(lineservice.ResolveBillablePeriodInput[billing.GatheringLine]{
				Line:               line,
				FeatureMeters:      featureMeters,
				ProgressiveBilling: progressiveBilling,
				AsOf:               line.InvoiceAt,
			})
			if err != nil {
				return fmt.Errorf("checking if line[%s] can be invoiced: %w", line.ID, err)
			}

			if period == nil {
				return billing.ValidationError{
					Err: fmt.Errorf("line[%s]: %w as of %s", line.ID, billing.ErrInvoiceLinesNotBillable, line.InvoiceAt),
				}
			}

			return nil
		})...,
	)
}
