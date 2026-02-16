package billingservice

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/service/lineservice"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// InvoicePendingLines invoices the pending lines for the customer.
// Flow (overview):
//
// 1) We fetch the gathering invoices of the customer and collect the lines that can be invoiced asOf input.AsOf.
//
// A line is considered billable if it's invoice_at <= input.AsOf OR if progressive billing is enabled and as per the
// line service the line can be invoiced in multiple parts.
//
// 2) We prepare the lines to be billed on the gathering invoice and update it in the database. (prepareLinesToBill)
//
// If a line needs to be split the splitGatheringInvoiceLine method is used, what it does:
//   - It creates a new split line group if it doesn't exist (it groups together the lines on multiple invoices when a single
//     gathering line is billed on multiple invoices)
//   - It creates a new line for the period up to the split at time, decreases the existing line's period end to the split at time.
//   - Note: ChildUniqueReferenceID is set to nil to avoid conflicts on the gathering invoice, the SplitLineGroup owns this unique reference.
//
// 3) We create a new standard invoice from the gathering invoice and associate the lines to it. (createStandardInvoiceFromGatheringLines)
//   - The in-scope lines are moved from the gathering invoice to the new standard invoice (moveLinesToInvoice)
//
// 4) We update the gathering invoice to remove the lines that have been associated to the new invoice. (updateGatheringInvoice)
//
// 5) We publish the invoice created event.
//
// 6) We return the created invoices.
func (s *Service) InvoicePendingLines(ctx context.Context, input billing.InvoicePendingLinesInput) ([]billing.StandardInvoice, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	if slices.Contains(s.fsNamespaceLockdown, input.Customer.Namespace) {
		return nil, billing.ValidationError{
			Err: fmt.Errorf("%w: %s", billing.ErrNamespaceLocked, input.Customer.Namespace),
		}
	}

	return transactionForInvoiceManipulation(
		ctx,
		s,
		input.Customer,
		func(ctx context.Context) ([]billing.StandardInvoice, error) {
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

			asOf := lo.FromPtrOr(input.AsOf, clock.Now())

			// let's fetch the existing gathering invoices for the customer
			existingGatheringInvoices, err := s.adapter.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
				Namespaces: []string{input.Customer.Namespace},
				Customers:  []string{input.Customer.ID},
				Expand: billing.GatheringInvoiceExpands{
					billing.GatheringInvoiceExpandLines,
				},
			})
			if err != nil {
				return nil, fmt.Errorf("fetching existing gathering invoices: %w", err)
			}

			// For consistency, we want to return an error if there are no gathering invoices for the customer. Without this,
			// the caller would get an empty slice of invoices instead of an error which is inconsistent with the error case when
			// there are no lines that are billable.
			if len(existingGatheringInvoices.Items) == 0 {
				return nil, billing.ValidationError{
					Err: billing.ErrInvoiceCreateNoLines,
				}
			}

			invoicesByCurrency := lo.SliceToMap(existingGatheringInvoices.Items, func(i billing.GatheringInvoice) (currencyx.Code, gatheringInvoiceWithFeatureMeters) {
				return i.Currency, gatheringInvoiceWithFeatureMeters{
					Invoice: i,
				}
			})

			// Let's resolve the feature meters for each gathering invoice line for downstream calculations.
			for currency, gatheringInvoiceWithCurrency := range invoicesByCurrency {
				featureMeters, err := s.resolveFeatureMeters(ctx, input.Customer.Namespace, invoicesByCurrency[currency].Invoice.Lines)
				if err != nil {
					return nil, fmt.Errorf("resolving feature meters: %w", err)
				}

				gatheringInvoiceWithCurrency.FeatureMeters = featureMeters
				invoicesByCurrency[currency] = gatheringInvoiceWithCurrency
			}

			if len(invoicesByCurrency) != len(existingGatheringInvoices.Items) {
				return nil, fmt.Errorf("customer has multiple gathering invoices for the same currency: %d", len(invoicesByCurrency))
			}

			// let's gather the in-scope lines and validate it
			inScopeLinesByCurrency, err := s.gatherInScopeLines(ctx, gatherInScopeLineInput{
				GatheringInvoicesByCurrency: invoicesByCurrency,
				LinesToInclude:              input.IncludePendingLines,
				AsOf:                        asOf,
				ProgressiveBilling: lo.FromPtrOr(
					input.ProgressiveBillingOverride,
					customerProfile.MergedProfile.WorkflowConfig.Invoicing.ProgressiveBilling,
				),
			})
			if err != nil {
				return nil, err
			}

			createdInvoices := make([]billing.StandardInvoice, 0, len(inScopeLinesByCurrency))

			for currency, inScopeLines := range inScopeLinesByCurrency {
				// Let's first make sure we have properly split the progressively billed
				// lines into multiple lines on the gathering invoice if needed.
				gatheringInvoice, ok := invoicesByCurrency[currency]
				if !ok {
					return nil, fmt.Errorf("gathering invoice for currency [%s] not found", currency)
				}

				if len(inScopeLines) == 0 {
					return nil, billing.ValidationError{
						Err: billing.ErrInvoiceCreateNoLines,
					}
				}

				createdInvoice, err := s.handleInvoicePendingLinesForCurrency(ctx, handleInvoicePendingLinesForCurrencyInput{
					Currency:                currency,
					Customer:                lo.FromPtr(customerProfile.Customer),
					GatheringInvoice:        gatheringInvoice.Invoice,
					FeatureMeters:           gatheringInvoice.FeatureMeters,
					InScopeLines:            inScopeLines,
					EffectiveBillingProfile: customerProfile.MergedProfile,
				})
				if err != nil {
					return nil, err
				}

				if createdInvoice == nil {
					return nil, fmt.Errorf("created invoice is nil")
				}

				createdInvoices = append(createdInvoices, *createdInvoice)
			}

			for _, invoice := range createdInvoices {
				event, err := billing.NewStandardInvoiceCreatedEvent(invoice)
				if err != nil {
					return nil, fmt.Errorf("creating event: %w", err)
				}

				err = s.publisher.Publish(ctx, event)
				if err != nil {
					return nil, fmt.Errorf("publishing event: %w", err)
				}
			}

			return createdInvoices, nil
		})
}

type gatheringLineWithBillablePeriod = lineservice.LineWithBillablePeriod[billing.GatheringLine]

type handleInvoicePendingLinesForCurrencyInput struct {
	Currency                currencyx.Code
	Customer                customer.Customer
	GatheringInvoice        billing.GatheringInvoice
	FeatureMeters           billing.FeatureMeters
	InScopeLines            []gatheringLineWithBillablePeriod
	EffectiveBillingProfile billing.Profile
}

func (in handleInvoicePendingLinesForCurrencyInput) Validate() error {
	if err := in.Currency.Validate(); err != nil {
		return fmt.Errorf("currency: %w", err)
	}

	if err := in.Customer.Validate(); err != nil {
		return fmt.Errorf("customer: %w", err)
	}

	if err := in.GatheringInvoice.Validate(); err != nil {
		return fmt.Errorf("gathering invoice: %w", err)
	}

	if in.FeatureMeters == nil {
		return fmt.Errorf("feature meters are required")
	}

	return nil
}

func (s *Service) handleInvoicePendingLinesForCurrency(ctx context.Context, in handleInvoicePendingLinesForCurrencyInput) (*billing.StandardInvoice, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	gatheringInvoice := in.GatheringInvoice

	// Step 1: Let's make sure we have lines properly split on the gathering invoice.
	// Invariant: the gathering invoice is updated to contain the new lines if any were split.
	prepareResults, err := s.prepareLinesToBill(ctx, prepareLinesToBillInput{
		GatheringInvoice: gatheringInvoice,
		FeatureMeters:    in.FeatureMeters,
		InScopeLines:     in.InScopeLines,
	})
	if err != nil {
		return nil, fmt.Errorf("gathering lines to bill: %w", err)
	}

	if prepareResults == nil {
		return nil, fmt.Errorf("lines to bill is nil")
	}

	if len(prepareResults.LinesToBill) == 0 {
		return nil, billing.ValidationError{
			Err: billing.ErrInvoiceCreateNoLines,
		}
	}

	gatheringInvoice = prepareResults.GatheringInvoice

	// Step 2: Let's create the standard invoice and move the lines to the new invoice.
	// Invariant:
	// - new invoice: initial calculations are done and persisted to the database
	// - gathering invoice: lines that have been associated to the new invoice are removed from the gathering invoice
	createStandardInvoiceResult, err := s.createStandardInvoiceFromGatheringLines(ctx, createStandardInvoiceFromGatheringLinesInput{
		Customer:                in.Customer,
		Currency:                in.Currency,
		GatheringInvoice:        gatheringInvoice,
		FeatureMeters:           in.FeatureMeters,
		Lines:                   prepareResults.LinesToBill,
		EffectiveBillingProfile: in.EffectiveBillingProfile,
	})
	if err != nil {
		return nil, fmt.Errorf("creating standard invoice and associating lines: %w", err)
	}

	if createStandardInvoiceResult == nil {
		return nil, fmt.Errorf("created invoice is nil")
	}

	return createStandardInvoiceResult, nil
}

type gatheringInvoiceWithFeatureMeters struct {
	Invoice       billing.GatheringInvoice
	FeatureMeters billing.FeatureMeters
}

type gatherInScopeLineInput struct {
	GatheringInvoicesByCurrency map[currencyx.Code]gatheringInvoiceWithFeatureMeters
	// If set restricts the lines to be included to these IDs, otherwise the AsOf is used
	// to determine the lines to be included.
	LinesToInclude     mo.Option[[]string]
	AsOf               time.Time
	ProgressiveBilling bool
}

type gatherInScopeLinesResult map[currencyx.Code][]gatheringLineWithBillablePeriod

func (s *Service) gatherInScopeLines(ctx context.Context, in gatherInScopeLineInput) (gatherInScopeLinesResult, error) {
	res := make(gatherInScopeLinesResult)

	billableLineIDs := make(map[string]interface{})

	asOfTruncated := in.AsOf.Truncate(streaming.MinimumWindowSizeDuration)

	for currency, invoice := range in.GatheringInvoicesByCurrency {
		linesWithResolvedPeriods, err := lineservice.GetLinesWithBillablePeriods(
			lineservice.GetLinesWithBillablePeriodsInput[billing.GatheringLine]{
				AsOf:               in.AsOf,
				ProgressiveBilling: in.ProgressiveBilling,
				Lines:              invoice.Invoice.Lines.OrEmpty(),
				FeatureMeters:      invoice.FeatureMeters,
			})
		if err != nil {
			return nil, err
		}

		if !in.ProgressiveBilling {
			// Somewhat of a hack: Since we are allowing subscriptions with different billing periods for ratecards, invoiceAt not necessarily equals
			// to the line's period start and end time.

			// So we have two kinds of progressive billing scenarios:
			// 1. the line needs to be split into multiple lines
			// 2. the line does not need to be split but it's invoiceAt is after the line's period end, when the line is technically billable, but
			//    from the user's perspective as they are not requesting progressive billing we should not include it on the invoice.

			linesWithResolvedPeriods = lo.Filter(linesWithResolvedPeriods, func(line lineservice.LineWithBillablePeriod[billing.GatheringLine], _ int) bool {
				invoiceAtTruncated := line.Line.InvoiceAt.Truncate(streaming.MinimumWindowSizeDuration)

				return invoiceAtTruncated.Before(asOfTruncated) || invoiceAtTruncated.Equal(asOfTruncated)
			})
		}

		for _, line := range linesWithResolvedPeriods {
			billableLineIDs[line.Line.ID] = struct{}{}
		}

		res[currency] = linesWithResolvedPeriods
	}

	// If the user has requested specific lines to be included, we need to filter the output to only include those lines
	// but only if all the requested lines are billable.
	if in.LinesToInclude.IsPresent() {
		// Step 1: Let's validate that all the requested lines are billable.

		nonBillableLineIDs := make([]string, 0, len(billableLineIDs))
		for _, lineID := range in.LinesToInclude.OrEmpty() {
			if _, ok := billableLineIDs[lineID]; !ok {
				nonBillableLineIDs = append(nonBillableLineIDs, lineID)
			}
		}

		if len(nonBillableLineIDs) > 0 {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("%w: %s", billing.ErrInvoiceLinesNotBillable, strings.Join(nonBillableLineIDs, ",")),
			}
		}

		// Step 2: Let's filter the output to only include lines the user requested to be billed

		linesShouldBeIncluded := lo.SliceToMap(in.LinesToInclude.OrEmpty(), func(lineID string) (string, interface{}) {
			return lineID, struct{}{}
		})

		for currency, lines := range res {
			res[currency] = lo.Filter(lines, func(line gatheringLineWithBillablePeriod, _ int) bool {
				_, ok := linesShouldBeIncluded[line.Line.ID]
				return ok
			})

			if len(res[currency]) == 0 {
				delete(res, currency)
			}
		}
	}

	return res, nil
}

type hasInvoicableLinesInput struct {
	Invoice            billing.GatheringInvoice
	AsOf               time.Time
	FeatureMeters      billing.FeatureMeters
	ProgressiveBilling bool
}

func (i hasInvoicableLinesInput) Validate() error {
	var errs []error

	if err := i.Invoice.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invoice: %w", err))
	}

	if i.AsOf.IsZero() {
		errs = append(errs, fmt.Errorf("asOf time must not be zero"))
	}

	if i.FeatureMeters == nil {
		errs = append(errs, fmt.Errorf("feature meters are required"))
	}

	return errors.Join(errs...)
}

func (s *Service) hasInvoicableLines(ctx context.Context, in hasInvoicableLinesInput) (bool, error) {
	if err := in.Validate(); err != nil {
		return false, err
	}

	inScopeLines, err := s.gatherInScopeLines(ctx, gatherInScopeLineInput{
		GatheringInvoicesByCurrency: map[currencyx.Code]gatheringInvoiceWithFeatureMeters{
			in.Invoice.Currency: {
				Invoice:       in.Invoice,
				FeatureMeters: in.FeatureMeters,
			},
		},
		AsOf:               in.AsOf,
		ProgressiveBilling: in.ProgressiveBilling,
	})
	if err != nil {
		return false, fmt.Errorf("gathering in scope lines: %w", err)
	}

	res, found := inScopeLines[in.Invoice.Currency]
	if !found {
		return false, nil
	}

	return len(res) > 0, nil
}

type prepareLinesToBillInput struct {
	GatheringInvoice billing.GatheringInvoice
	FeatureMeters    billing.FeatureMeters
	InScopeLines     []gatheringLineWithBillablePeriod
}

func (i prepareLinesToBillInput) Validate() error {
	var errs []error

	if i.GatheringInvoice.Lines.IsAbsent() {
		errs = append(errs, fmt.Errorf("gathering invoice must have lines expanded"))
	}

	if i.FeatureMeters == nil {
		errs = append(errs, fmt.Errorf("feature meters are required"))
	}

	for _, line := range i.InScopeLines {
		if line.Line.InvoiceID != i.GatheringInvoice.ID {
			errs = append(errs, fmt.Errorf("line[%s]: line is not associated with gathering invoice[%s]", line.Line.ID, i.GatheringInvoice.ID))
		}

		if line.Line.Currency != i.GatheringInvoice.Currency {
			errs = append(errs, fmt.Errorf("line[%s]: line currency[%s] is not equal to gathering invoice currency[%s]", line.Line.ID, line.Line.Currency, i.GatheringInvoice.Currency))
		}
	}

	return errors.Join(errs...)
}

type prepareLinesToBillResult struct {
	LinesToBill      billing.GatheringLines
	GatheringInvoice billing.GatheringInvoice
}

// prepareLinesToBill prepares the lines to be billed from the gathering invoice, if needed
// lines are split into multiple lines for progressively billed lines on the gathering invoice.
func (s *Service) prepareLinesToBill(ctx context.Context, input prepareLinesToBillInput) (*prepareLinesToBillResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	gatheringInvoice := input.GatheringInvoice

	invoiceLines := make([]billing.GatheringLine, 0, len(input.InScopeLines))
	wasSplit := false

	for _, line := range input.InScopeLines {
		if !line.Line.ServicePeriod.Equal(line.BillablePeriod) {
			// We need to split the line into multiple lines
			if !line.Line.ServicePeriod.From.Equal(line.BillablePeriod.From) {
				return nil, fmt.Errorf("line[%s]: line period start[%s] is not equal to billable period start[%s]", line.Line.ID, line.Line.ServicePeriod.From, line.BillablePeriod.From)
			}

			splitLine, err := s.splitGatheringInvoiceLine(ctx, splitGatheringInvoiceLineInput{
				GatheringInvoice: gatheringInvoice,
				FeatureMeters:    input.FeatureMeters,
				LineID:           line.Line.ID,
				SplitAt:          line.BillablePeriod.To,
			})
			if err != nil {
				return nil, fmt.Errorf("line[%s]: splitting line: %w", line.Line.ID, err)
			}

			if splitLine.PreSplitAtLine.DeletedAt != nil {
				wasSplit = true

				s.logger.WarnContext(ctx, "pre split line is nil, skipping collection",
					"line", line.Line.ID,
					"original_period_start", line.Line.ServicePeriod.From,
					"original_period_end", line.Line.ServicePeriod.To,
					"split_at", line.BillablePeriod.To)
				continue
			}

			gatheringInvoice = splitLine.GatheringInvoice
			invoiceLines = append(invoiceLines, splitLine.PreSplitAtLine)
			wasSplit = true
		} else {
			invoiceLines = append(invoiceLines, line.Line)
		}
	}

	if wasSplit {
		// Let's update the gathering invoice to contain the new lines that we have split
		err := s.adapter.UpdateGatheringInvoice(ctx, gatheringInvoice)
		if err != nil {
			return nil, fmt.Errorf("updating gathering invoice: %w", err)
		}

		updatedInvoice, err := s.adapter.GetGatheringInvoiceById(ctx, billing.GetGatheringInvoiceByIdInput{
			Invoice: gatheringInvoice.GetInvoiceID(),
			Expand: billing.GatheringInvoiceExpands{
				billing.GatheringInvoiceExpandLines,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("getting gathering invoice: %w", err)
		}

		gatheringInvoice = updatedInvoice
	}

	return &prepareLinesToBillResult{
		LinesToBill:      invoiceLines,
		GatheringInvoice: gatheringInvoice,
	}, nil
}

type splitGatheringInvoiceLineInput struct {
	GatheringInvoice billing.GatheringInvoice
	FeatureMeters    billing.FeatureMeters
	LineID           string
	SplitAt          time.Time
}

func (i splitGatheringInvoiceLineInput) Validate() error {
	var errs []error

	if i.LineID == "" {
		errs = append(errs, fmt.Errorf("line ID is required"))
	}

	if i.SplitAt.IsZero() {
		errs = append(errs, fmt.Errorf("split at is required"))
	}

	if i.GatheringInvoice.Lines.IsAbsent() {
		errs = append(errs, fmt.Errorf("gathering invoice must have lines expanded"))
	}

	if i.FeatureMeters == nil {
		errs = append(errs, fmt.Errorf("feature meters are required"))
	}

	return errors.Join(errs...)
}

type splitGatheringInvoiceLineResult struct {
	PreSplitAtLine   billing.GatheringLine
	PostSplitAtLine  billing.GatheringLine
	GatheringInvoice billing.GatheringInvoice
}

// splitGatheringInvoiceLine splits a gathering invoice line into two lines, one will be from the
// line's period start up to the split at time, the other will be from the split at time to the line's period end.
//
// The gathering invoice's lines will be updated to contain both lines.

func (s *Service) splitGatheringInvoiceLine(ctx context.Context, in splitGatheringInvoiceLineInput) (splitGatheringInvoiceLineResult, error) {
	res := splitGatheringInvoiceLineResult{}

	if err := in.Validate(); err != nil {
		return res, err
	}

	gatheringInvoice := in.GatheringInvoice

	line, found := gatheringInvoice.Lines.GetByID(in.LineID)
	if !found {
		return res, fmt.Errorf("line[%s]: line not found in gathering invoice", in.LineID)
	}

	if !line.ServicePeriod.Contains(in.SplitAt) {
		return res, fmt.Errorf("line[%s]: splitAt is not within the line period", line.ID)
	}

	var splitLineGroupID string
	if line.SplitLineGroupID == nil {
		splitLineGroup, err := s.adapter.CreateSplitLineGroup(ctx, billing.CreateSplitLineGroupAdapterInput{
			Namespace: line.Namespace,

			SplitLineGroupMutableFields: billing.SplitLineGroupMutableFields{
				Name:        line.Name,
				Description: line.Description,

				ServicePeriod:     billing.Period{Start: line.ServicePeriod.From, End: line.ServicePeriod.To},
				RatecardDiscounts: line.RateCardDiscounts,
				TaxConfig:         line.TaxConfig,
			},

			UniqueReferenceID: line.ChildUniqueReferenceID,

			Currency: line.Currency,

			Price:      lo.ToPtr(line.Price),
			FeatureKey: lo.EmptyableToPtr(line.FeatureKey),

			Subscription: line.Subscription,
		})
		if err != nil {
			return res, fmt.Errorf("creating split line group: %w", err)
		}

		splitLineGroupID = splitLineGroup.ID
	} else {
		splitLineGroupID = lo.FromPtr(line.SplitLineGroupID)
		if splitLineGroupID == "" {
			return res, fmt.Errorf("split line group id is empty")
		}
	}

	// We have already split the line once, we just need to create a new line and update the existing line
	postSplitAtLine, err := line.CloneForCreate(func(l *billing.GatheringLine) {
		l.ServicePeriod.From = in.SplitAt
		l.SplitLineGroupID = lo.ToPtr(splitLineGroupID)

		l.ChildUniqueReferenceID = nil
	})
	if err != nil {
		return res, fmt.Errorf("cloning post split line: %w", err)
	}

	postSplitAtLineEmpty, err := lineservice.IsPeriodEmptyConsideringTruncations(postSplitAtLine)
	if err != nil {
		return res, fmt.Errorf("checking if post split line is empty: %w", err)
	}

	if !postSplitAtLineEmpty {
		if err := postSplitAtLine.Validate(); err != nil {
			return res, fmt.Errorf("validating post split line: %w", err)
		}

		gatheringInvoice.Lines.Append(postSplitAtLine)
	}

	// Let's update the original line to only contain the period up to the splitAt time
	line.ServicePeriod.To = in.SplitAt
	line.InvoiceAt = in.SplitAt
	line.SplitLineGroupID = lo.ToPtr(splitLineGroupID)
	line.ChildUniqueReferenceID = nil

	preSplitAtLine := line

	preSplitAtLineEmpty, err := lineservice.IsPeriodEmptyConsideringTruncations(preSplitAtLine)
	if err != nil {
		return res, fmt.Errorf("checking if pre split line is empty: %w", err)
	}

	// If the line became empty, due to the split, let's remove it from the gathering invoice
	if preSplitAtLineEmpty {
		line.DeletedAt = lo.ToPtr(clock.Now())
	} else {
		if err := preSplitAtLine.Validate(); err != nil {
			return res, fmt.Errorf("validating pre split line: %w", err)
		}
	}

	if err := gatheringInvoice.Lines.ReplaceByID(preSplitAtLine); err != nil {
		return res, fmt.Errorf("setting pre split line: %w", err)
	}

	return splitGatheringInvoiceLineResult{
		PreSplitAtLine:   preSplitAtLine,
		PostSplitAtLine:  postSplitAtLine,
		GatheringInvoice: gatheringInvoice,
	}, nil
}

type createStandardInvoiceFromGatheringLinesInput struct {
	Customer                customer.Customer
	Currency                currencyx.Code
	GatheringInvoice        billing.GatheringInvoice
	FeatureMeters           billing.FeatureMeters
	Lines                   billing.GatheringLines
	EffectiveBillingProfile billing.Profile
}

func (in createStandardInvoiceFromGatheringLinesInput) Validate() error {
	var errs []error

	if err := in.Customer.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	if err := in.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if err := in.EffectiveBillingProfile.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("effective billing profile: %w", err))
	}

	if len(in.Lines) == 0 {
		errs = append(errs, fmt.Errorf("lines must contain at least one line"))
	}

	for _, line := range in.Lines {
		if err := line.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("line[%s]: %w", line.ID, err))
		}
	}

	if in.FeatureMeters == nil {
		errs = append(errs, fmt.Errorf("feature meters are required"))
	}

	return errors.Join(errs...)
}

// createStandardInvoiceFromGatheringLines creates a standard invoice from the gathering invoice lines.
// Invariant:
// - the standard invoice is in draft.created state, and is calculated and persisted to the database
// - the gathering invoice's lines are deleted, and persisted to the database
func (s *Service) createStandardInvoiceFromGatheringLines(ctx context.Context, in createStandardInvoiceFromGatheringLinesInput) (*billing.StandardInvoice, error) {
	if err := in.Validate(); err != nil {
		return nil, fmt.Errorf("validating input: %w", err)
	}

	invoiceNumber, err := s.GenerateInvoiceSequenceNumber(ctx,
		billing.SequenceGenerationInput{
			Namespace:    in.Customer.Namespace,
			CustomerName: in.Customer.Name,
			Currency:     in.Currency,
		},
		billing.DraftInvoiceSequenceNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("generating invoice number: %w", err)
	}

	// let's create the invoice
	invoice, err := s.adapter.CreateInvoice(ctx, billing.CreateInvoiceAdapterInput{
		Namespace: in.Customer.Namespace,
		Customer:  in.Customer,
		Profile:   in.EffectiveBillingProfile,

		Currency: in.Currency,
		Number:   invoiceNumber,
		Status:   billing.StandardInvoiceStatusDraftCreated,

		Type: billing.InvoiceTypeStandard,
	})
	if err != nil {
		return nil, fmt.Errorf("creating invoice: %w", err)
	}

	invoiceID := invoice.ID

	// let's resolve the workflow apps as some checks such as CanDraftSyncAdvance depends on the apps
	invoice, err = s.resolveWorkflowApps(ctx, invoice)
	if err != nil {
		return nil, fmt.Errorf("error resolving workflow apps for invoice [%s]: %w", invoiceID, err)
	}

	convertResults, err := s.convertGatheringLinesToStandardLines(ctx, convertGatheringLinesToStandardLinesInput{
		TargetInvoice:           invoice,
		FeatureMeters:           in.FeatureMeters,
		GatheringLinesToConvert: in.Lines,
	})
	if err != nil {
		return nil, fmt.Errorf("moving lines to invoice: %w", err)
	}

	invoice = convertResults.TargetInvoice

	// Let's first update the gathering invoice to make sure deleted lines are synced, as the standard invoice will have expanded split line hierarchies
	// and we need to make sure that gathering invoice lines that are already yielded the standard invoice lines are excluded from the split line hierarchy.
	//
	// Note: this is a hack, on the long term we need to have a Charge type that encapsulates all of this logic.
	err = s.removeLinesFromGatheringInvoice(ctx, in.GatheringInvoice, in.Lines)
	if err != nil {
		return nil, fmt.Errorf("updating gathering invoice: %w", err)
	}

	// Prerequisite: we should have the split line group headers expanded so that snapshotting can determine if the preLine
	// queries are needed.
	if err := s.resolveSplitLineGroupHeadersForLines(ctx, in.Customer.Namespace, convertResults.LinesAssociated); err != nil {
		return nil, fmt.Errorf("resolving split line group headers for lines: %w", err)
	}

	// Let's snapshot the quantities for the lines that we have converted to standard lines so that calculations can be performed
	if err := s.snapshotLineQuantitiesInParallel(ctx, invoice.Customer, convertResults.LinesAssociated, in.FeatureMeters); err != nil {
		return nil, fmt.Errorf("snapshotting lines: %w", err)
	}

	// Let's persist the snapshotted values to the database as the state machine always reloads the invoice from the database to make
	// sure we don't have any manual modifications inside the invoice structure.
	invoice, err = s.updateInvoice(ctx, invoice)
	if err != nil {
		return nil, fmt.Errorf("updating target invoice: %w", err)
	}

	// Let's make sure that the invoice is in an up-to-date state
	invoice, err = s.withLockedInvoiceStateMachine(ctx, withLockedStateMachineInput{
		InvoiceID: invoice.GetInvoiceID(),
		Callback: func(ctx context.Context, sm *InvoiceStateMachine) error {
			// Let's activate the state machine so that the created state's calculation is triggered
			if err := sm.StateMachine.ActivateCtx(ctx); err != nil {
				return fmt.Errorf("activating invoice state machine: %w", err)
			}

			// If the invoice has critical validation issues => trigger a failed state
			if sm.Invoice.HasCriticalValidationIssues() {
				return sm.TriggerFailed(ctx)
			}

			invoiceID := sm.Invoice.ID

			// If we have reached this point, we need to persist the invoice to the database so that all the
			// entities have IDs available for the app.
			sm.Invoice, err = s.updateInvoice(ctx, sm.Invoice)
			if err != nil {
				return fmt.Errorf("updating invoice[%s]: %w", invoiceID, err)
			}

			// Otherwise, let's advance the invoice to the next final state
			if err := s.advanceUntilStateStable(ctx, sm); err != nil {
				return fmt.Errorf("activating invoice: %w", err)
			}

			sm.Invoice, err = s.updateInvoice(ctx, sm.Invoice)
			if err != nil {
				return fmt.Errorf("updating invoice[%s]: %w", invoiceID, err)
			}

			return nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("activating invoice: %w", err)
	}

	return &invoice, nil
}

type convertGatheringLinesToStandardLinesInput struct {
	FeatureMeters           billing.FeatureMeters
	TargetInvoice           billing.StandardInvoice
	GatheringLinesToConvert billing.GatheringLines
}

func (in convertGatheringLinesToStandardLinesInput) Validate() error {
	if err := in.TargetInvoice.Validate(); err != nil {
		return fmt.Errorf("target invoice: %w", err)
	}

	if in.TargetInvoice.Status != billing.StandardInvoiceStatusDraftCreated {
		return fmt.Errorf("target invoice must be in draft created status")
	}

	if len(in.GatheringLinesToConvert) == 0 {
		return fmt.Errorf("line IDs to move is required")
	}

	if in.TargetInvoice.ID == "" {
		return fmt.Errorf("target invoice ID is required")
	}

	for _, line := range in.GatheringLinesToConvert {
		if err := line.Validate(); err != nil {
			return fmt.Errorf("validating gathering line: %w", err)
		}

		if line.Currency != in.TargetInvoice.Currency {
			return fmt.Errorf("gathering line[%s]: currency[%s] is not equal to target invoice currency[%s]", line.ID, line.Currency, in.TargetInvoice.Currency)
		}

		if line.Namespace != in.TargetInvoice.Namespace {
			return fmt.Errorf("gathering line[%s]: namespace[%s] is not equal to target invoice namespace[%s]", line.ID, line.Namespace, in.TargetInvoice.Namespace)
		}
	}

	if in.FeatureMeters == nil {
		return fmt.Errorf("feature meters are required")
	}

	return nil
}

type convertGatheringLinesToStandardLinesResult struct {
	TargetInvoice   billing.StandardInvoice
	LinesAssociated billing.StandardLines
}

// convertGatheringLinesToStandardLines converts the gathering lines to standard lines and adds them to the target invoice.
// Invariants:
// - the target invoice is updated by adding the standard lines that have been converted from the gathering lines
// - no database changes are made
func (s *Service) convertGatheringLinesToStandardLines(ctx context.Context, in convertGatheringLinesToStandardLinesInput) (*convertGatheringLinesToStandardLinesResult, error) {
	if err := in.Validate(); err != nil {
		return nil, fmt.Errorf("validating input: %w", err)
	}

	newStandardLines, err := slicesx.MapWithErr(in.GatheringLinesToConvert, func(gatheringLine billing.GatheringLine) (*billing.StandardLine, error) {
		newStandardLine, err := convertGatheringLineToNewStandardLine(gatheringLine, in.TargetInvoice.ID)
		if err != nil {
			return nil, fmt.Errorf("converting gathering line to new standard line: %w", err)
		}

		if err := newStandardLine.Validate(); err != nil {
			return nil, fmt.Errorf("validating new standard line: %w", err)
		}

		return newStandardLine, nil
	})
	if err != nil {
		return nil, fmt.Errorf("converting gathering lines to standard lines: %w", err)
	}

	// Let's add the lines to the target invoice
	in.TargetInvoice.Lines.Append(newStandardLines...)

	return &convertGatheringLinesToStandardLinesResult{
		TargetInvoice:   in.TargetInvoice,
		LinesAssociated: newStandardLines,
	}, nil
}

func convertGatheringLineToNewStandardLine(line billing.GatheringLine, invoiceID string) (*billing.StandardLine, error) {
	clonedAnnotations, err := line.Annotations.Clone()
	if err != nil {
		return nil, fmt.Errorf("cloning annotations: %w", err)
	}

	var taxConfig *productcatalog.TaxConfig
	if line.TaxConfig != nil {
		taxConfig = lo.ToPtr(line.TaxConfig.Clone())
	}

	var subscription *billing.SubscriptionReference
	if line.Subscription != nil {
		subscription = line.Subscription.Clone()
	}

	convertedLine := &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: line.ManagedResource,
			Metadata:        line.Metadata.Clone(),
			Annotations:     clonedAnnotations,
			ManagedBy:       line.ManagedBy,
			InvoiceID:       invoiceID,
			Currency:        line.Currency,

			Period: billing.Period{
				Start: line.ServicePeriod.From,
				End:   line.ServicePeriod.To,
			},
			InvoiceAt: line.InvoiceAt,

			TaxConfig:              taxConfig,
			RateCardDiscounts:      line.RateCardDiscounts.Clone(),
			ChildUniqueReferenceID: line.ChildUniqueReferenceID,
			Subscription:           subscription,
			SplitLineGroupID:       line.SplitLineGroupID,
			ChargeID:               line.ChargeID,
		},
		UsageBased: &billing.UsageBasedLine{
			Price:      lo.ToPtr(line.Price),
			FeatureKey: line.FeatureKey,
		},

		DBState: nil, // We don't want to reuse the state from the gathering line (so let's make it explicit)
	}

	return convertedLine, nil
}

// updateGatheringInvoice updates the gathering invoice's state and if it contains no lines, it will be deleted.
// Invariant:
// - the invoice is recalculated
// - the invoice is updated to the database
func (s *Service) removeLinesFromGatheringInvoice(ctx context.Context, invoice billing.GatheringInvoice, linesToRemove billing.GatheringLines) error {
	lineIDsToRemove := lo.Map(linesToRemove, func(l billing.GatheringLine, _ int) string { return l.ID })

	nrLinesRemoved := 0
	invoiceLinesWithoutRemovedLines := lo.Filter(invoice.Lines.OrEmpty(), func(l billing.GatheringLine, _ int) bool {
		if slices.Contains(lineIDsToRemove, l.ID) {
			nrLinesRemoved++
			return false
		}

		return true
	})

	// This makes sure that all the IDs are present on the gathering invoice before invoking the hard delete.
	if nrLinesRemoved != len(lineIDsToRemove) {
		return fmt.Errorf("lines to remove[%d] must contain the same number of lines as line IDs to remove[%d]", nrLinesRemoved, len(lineIDsToRemove))
	}

	invoice.Lines = billing.NewGatheringInvoiceLines(invoiceLinesWithoutRemovedLines)

	// We need to hard delete the lines from the gathering invoice as now the standard lines are taking their place with the same IDs.
	// If we would soft-delete the lines, all downstream services would assume that the line was deleted due to synchronization and
	// would recreate it.
	if err := s.adapter.HardDeleteGatheringInvoiceLines(ctx, invoice.GetInvoiceID(), lineIDsToRemove); err != nil {
		return fmt.Errorf("hard deleting gathering invoice lines: %w", err)
	}

	// Let's update the invoice's state
	if err := s.invoiceCalculator.CalculateGatheringInvoice(&invoice); err != nil {
		return fmt.Errorf("calculating gathering invoice: %w", err)
	}

	// The gathering invoice has no lines => delete the invoice
	if invoice.Lines.NonDeletedLineCount() == 0 {
		invoice.DeletedAt = lo.ToPtr(clock.Now())
	}

	err := s.adapter.UpdateGatheringInvoice(ctx, invoice)
	if err != nil {
		return fmt.Errorf("updating gathering invoice: %w", err)
	}

	return nil
}

// resolveSplitLineGroupHeadersForLines resolves the split line group headers for the given lines.
// Warning: this will not fetch the lines from the database, so only use this if you are sure that
// only the headers are needed. (e.g. don't use it for invoice calculations or usage discounts
// will be off)
func (s *Service) resolveSplitLineGroupHeadersForLines(ctx context.Context, ns string, lines billing.StandardLines) error {
	splitLineGroupIDs := lo.Uniq(
		lo.Filter(
			lo.Map(lines, func(line *billing.StandardLine, _ int) string { return lo.FromPtr(line.SplitLineGroupID) }),
			func(id string, _ int) bool { return id != "" },
		),
	)

	if len(splitLineGroupIDs) == 0 {
		return nil
	}

	splitLineGroupHeaders, err := s.adapter.GetSplitLineGroupHeaders(ctx, billing.GetSplitLineGroupHeadersInput{
		Namespace:         ns,
		SplitLineGroupIDs: splitLineGroupIDs,
	})
	if err != nil {
		return fmt.Errorf("getting split line group headers: %w", err)
	}

	splitLineGroupHeadersByID := lo.SliceToMap(splitLineGroupHeaders, func(header billing.SplitLineGroup) (string, billing.SplitLineGroup) { return header.ID, header })

	for idx := range lines {
		if lines[idx].SplitLineGroupID == nil {
			continue
		}

		splitLineGroupHeader, ok := splitLineGroupHeadersByID[lo.FromPtr(lines[idx].SplitLineGroupID)]
		if !ok {
			return fmt.Errorf("split line group header not found for line[%s]: id[%s]", lines[idx].ID, lo.FromPtr(lines[idx].SplitLineGroupID))
		}

		lines[idx].SplitLineHierarchy = &billing.SplitLineHierarchy{
			Group: splitLineGroupHeader,
		}
	}

	return nil
}
