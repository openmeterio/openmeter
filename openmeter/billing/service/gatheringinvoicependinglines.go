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
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
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

	return transcationForInvoiceManipulation(
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
			existingGatheringInvoices, err := s.ListInvoices(ctx, billing.ListInvoicesInput{
				Namespaces:       []string{input.Customer.Namespace},
				Customers:        []string{input.Customer.ID},
				ExtendedStatuses: []billing.StandardInvoiceStatus{billing.StandardInvoiceStatusGathering},
				Expand: billing.InvoiceExpand{
					Lines: true,
				},
			})
			if err != nil {
				return nil, fmt.Errorf("fetching existing gathering invoices: %w", err)
			}

			invoicesByCurrency := lo.SliceToMap(existingGatheringInvoices.Items, func(i billing.StandardInvoice) (currencyx.Code, gatheringInvoiceWithFeatureMeters) {
				return i.Currency, gatheringInvoiceWithFeatureMeters{
					Invoice: i,
				}
			})

			// Let's resolve the feature meters for each gathering invoice line for downstream calculations.
			for currency, gatheringInvoiceWithCurrency := range invoicesByCurrency {
				featureMeters, err := s.resolveFeatureMeters(ctx, invoicesByCurrency[currency].Invoice.Lines.OrEmpty())
				if err != nil {
					return nil, fmt.Errorf("resolving feature meters: %w", err)
				}

				gatheringInvoiceWithCurrency.FeatureMeters = featureMeters
				invoicesByCurrency[currency] = gatheringInvoiceWithCurrency
			}

			if len(invoicesByCurrency) != len(existingGatheringInvoices.Items) {
				return nil, fmt.Errorf("customer has multiple gathering invoices for the same currency: %d", len(invoicesByCurrency))
			}

			if len(invoicesByCurrency) == 0 {
				return nil, billing.ErrInvoiceCreateNoLines
			}

			// let's gather the in-scope lines and validate it
			inScopeLines, err := s.gatherInScopeLines(ctx, gatherInScopeLineInput{
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

			if len(inScopeLines) == 0 {
				return nil, billing.ErrInvoiceCreateNoLines
			}

			createdInvoices := make([]billing.StandardInvoice, 0, len(inScopeLines))

			for currency, inScopeLines := range inScopeLines {
				// Let's first make sure we have properly split the progressively billed
				// lines into multiple lines on the gathering invoice if needed.
				gatheringInvoice, ok := invoicesByCurrency[currency]
				if !ok {
					return nil, fmt.Errorf("gathering invoice for currency [%s] not found", currency)
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
					return nil, fmt.Errorf("handling invoice pending lines for currency: %w", err)
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

type handleInvoicePendingLinesForCurrencyInput struct {
	Currency                currencyx.Code
	Customer                customer.Customer
	GatheringInvoice        billing.StandardInvoice
	FeatureMeters           billing.FeatureMeters
	InScopeLines            []lineservice.LineWithBillablePeriod
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

	if len(in.InScopeLines) == 0 {
		return fmt.Errorf("in scope lines must contain at least one line")
	}

	if in.FeatureMeters == nil {
		return fmt.Errorf("feature meters are required")
	}

	return nil
}

func (s *Service) handleInvoicePendingLinesForCurrency(ctx context.Context, in handleInvoicePendingLinesForCurrencyInput) (*billing.StandardInvoice, error) {
	if err := in.Validate(); err != nil {
		return nil, fmt.Errorf("validating input: %w", err)
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

	if len(prepareResults.LineIDsToBill) == 0 {
		return nil, fmt.Errorf("no lines to bill")
	}

	gatheringInvoice = prepareResults.GatheringInvoice

	// Step 2: Let's create the standard invoice and move the lines to the new invoice.
	linesToBill := lo.Filter(gatheringInvoice.Lines.OrEmpty(), func(line *billing.StandardLine, _ int) bool {
		return slices.Contains(prepareResults.LineIDsToBill, line.ID)
	})

	if len(linesToBill) != len(prepareResults.LineIDsToBill) {
		return nil, fmt.Errorf("lines to associate[%d] must contain the same number of lines as lines to bill[%d]", len(linesToBill), len(prepareResults.LineIDsToBill))
	}

	// Let's create the invoice and associate the lines to it
	// Invariant:
	// - new invoice: initial calculations are done and persisted to the database
	// - gathering invoice: lines that have been associated to the new invoice are removed from the gathering invoice
	createStandardInvoiceResult, err := s.createStandardInvoiceFromGatheringLines(ctx, createStandardInvoiceFromGatheringLinesInput{
		Customer:                in.Customer,
		Currency:                in.Currency,
		GatheringInvoice:        gatheringInvoice,
		FeatureMeters:           in.FeatureMeters,
		Lines:                   linesToBill,
		EffectiveBillingProfile: in.EffectiveBillingProfile,
	})
	if err != nil {
		return nil, fmt.Errorf("creating standard invoice and associating lines: %w", err)
	}

	gatheringInvoice = createStandardInvoiceResult.GatheringInvoice

	_, err = s.updateGatheringInvoice(ctx, gatheringInvoice)
	if err != nil {
		return nil, fmt.Errorf("updating gathering invoice: %w", err)
	}

	return &createStandardInvoiceResult.CreatedInvoice, nil
}

type gatheringInvoiceWithFeatureMeters struct {
	Invoice       billing.StandardInvoice
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

type gatherInScopeLinesResult map[currencyx.Code][]lineservice.LineWithBillablePeriod

func (s *Service) gatherInScopeLines(ctx context.Context, in gatherInScopeLineInput) (gatherInScopeLinesResult, error) {
	res := make(gatherInScopeLinesResult)

	billableLineIDs := make(map[string]interface{})

	for currency, invoice := range in.GatheringInvoicesByCurrency {
		lineSrvs, err := s.lineService.FromEntities(invoice.Invoice.Lines.OrEmpty(), invoice.FeatureMeters)
		if err != nil {
			return nil, err
		}

		linesWithResolvedPeriods, err := lineSrvs.ResolveBillablePeriod(ctx, lineservice.ResolveBillablePeriodInput{
			AsOf:               in.AsOf,
			ProgressiveBilling: in.ProgressiveBilling,
		})
		if err != nil {
			return nil, err
		}

		for _, line := range linesWithResolvedPeriods {
			billableLineIDs[line.ID()] = struct{}{}
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
			return nil, billing.NotFoundError{
				ID:     strings.Join(nonBillableLineIDs, ","),
				Entity: billing.EntityInvoiceLine,
				Err:    billing.ErrInvoiceLinesNotBillable,
			}
		}

		// Step 2: Let's filter the output to only include lines the user requested to be billed

		linesShouldBeIncluded := lo.SliceToMap(in.LinesToInclude.OrEmpty(), func(lineID string) (string, interface{}) {
			return lineID, struct{}{}
		})

		for currency, lines := range res {
			res[currency] = lo.Filter(lines, func(line lineservice.LineWithBillablePeriod, _ int) bool {
				_, ok := linesShouldBeIncluded[line.ID()]
				return ok
			})

			if len(res[currency]) == 0 {
				delete(res, currency)
			}
		}
	}

	return res, nil
}

type prepareLinesToBillInput struct {
	GatheringInvoice billing.StandardInvoice
	FeatureMeters    billing.FeatureMeters
	InScopeLines     []lineservice.LineWithBillablePeriod
}

func (i prepareLinesToBillInput) Validate() error {
	var errs []error

	if i.GatheringInvoice.Status != billing.StandardInvoiceStatusGathering {
		errs = append(errs, fmt.Errorf("gathering invoice is not in gathering status"))
	}

	if len(i.InScopeLines) == 0 {
		errs = append(errs, fmt.Errorf("no lines to bill"))
	}

	if i.GatheringInvoice.Lines.IsAbsent() {
		errs = append(errs, fmt.Errorf("gathering invoice must have lines expanded"))
	}

	if i.FeatureMeters == nil {
		errs = append(errs, fmt.Errorf("feature meters are required"))
	}

	for _, line := range i.InScopeLines {
		if line.InvoiceID() != i.GatheringInvoice.ID {
			errs = append(errs, fmt.Errorf("line[%s]: line is not associated with gathering invoice[%s]", line.ID(), i.GatheringInvoice.ID))
		}

		if line.Currency() != i.GatheringInvoice.Currency {
			errs = append(errs, fmt.Errorf("line[%s]: line currency[%s] is not equal to gathering invoice currency[%s]", line.ID(), line.Currency(), i.GatheringInvoice.Currency))
		}
	}

	return errors.Join(errs...)
}

type prepareLinesToBillResult struct {
	LineIDsToBill    []string
	GatheringInvoice billing.StandardInvoice
}

// prepareLinesToBill prepares the lines to be billed from the gathering invoice, if needed
// lines are split into multiple lines for progressively billed lines on the gathering invoice.
func (s *Service) prepareLinesToBill(ctx context.Context, input prepareLinesToBillInput) (*prepareLinesToBillResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	gatheringInvoice := input.GatheringInvoice

	invoiceLines := make(lineservice.Lines, 0, len(input.InScopeLines))
	wasSplit := false

	for _, line := range input.InScopeLines {
		if !line.Period().Equal(line.BillablePeriod) {
			// We need to split the line into multiple lines
			if !line.Period().Start.Equal(line.BillablePeriod.Start) {
				return nil, fmt.Errorf("line[%s]: line period start[%s] is not equal to billable period start[%s]", line.ID(), line.Period().Start, line.BillablePeriod.Start)
			}

			splitLine, err := s.splitGatheringInvoiceLine(ctx, splitGatheringInvoiceLineInput{
				GatheringInvoice: gatheringInvoice,
				FeatureMeters:    input.FeatureMeters,
				LineID:           line.ID(),
				SplitAt:          line.BillablePeriod.End,
			})
			if err != nil {
				return nil, fmt.Errorf("line[%s]: splitting line: %w", line.ID(), err)
			}

			if splitLine.PreSplitAtLine == nil {
				s.logger.WarnContext(ctx, "pre split line is nil, we are not creating empty lines", "line", line.ID(), "period_start", line.Period().Start, "period_end", line.Period().End)
				continue
			}

			gatheringInvoice = splitLine.GatheringInvoice
			invoiceLines = append(invoiceLines, splitLine.PreSplitAtLine)
			wasSplit = true
		} else {
			invoiceLines = append(invoiceLines, line)
		}
	}

	if wasSplit {
		// Let's update the gathering invoice to contain the new lines that we have split
		updatedInvoice, err := s.adapter.UpdateInvoice(ctx, gatheringInvoice)
		if err != nil {
			return nil, fmt.Errorf("updating gathering invoice: %w", err)
		}

		gatheringInvoice = updatedInvoice
	}

	return &prepareLinesToBillResult{
		LineIDsToBill:    lo.Map(invoiceLines, func(l lineservice.Line, _ int) string { return l.ID() }),
		GatheringInvoice: gatheringInvoice,
	}, nil
}

type splitGatheringInvoiceLineInput struct {
	GatheringInvoice billing.StandardInvoice
	FeatureMeters    billing.FeatureMeters
	LineID           string
	SplitAt          time.Time
}

func (i splitGatheringInvoiceLineInput) Validate() error {
	var errs []error

	if i.GatheringInvoice.Status != billing.StandardInvoiceStatusGathering {
		errs = append(errs, fmt.Errorf("gathering invoice is not in gathering status"))
	}

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
	PreSplitAtLine   lineservice.Line
	PostSplitAtLine  lineservice.Line
	GatheringInvoice billing.StandardInvoice
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

	line := gatheringInvoice.Lines.GetByID(in.LineID)
	if line == nil {
		return res, fmt.Errorf("line[%s]: line not found in gathering invoice", in.LineID)
	}
	if !line.Period.Contains(in.SplitAt) {
		return res, fmt.Errorf("line[%s]: splitAt is not within the line period", line.ID)
	}

	var splitLineGroupID string
	if line.SplitLineGroupID == nil {
		splitLineGroup, err := s.adapter.CreateSplitLineGroup(ctx, billing.CreateSplitLineGroupAdapterInput{
			Namespace: line.Namespace,

			SplitLineGroupMutableFields: billing.SplitLineGroupMutableFields{
				Name:        line.Name,
				Description: line.Description,

				ServicePeriod:     line.Period,
				RatecardDiscounts: line.RateCardDiscounts,
				TaxConfig:         line.TaxConfig,
			},

			UniqueReferenceID: line.ChildUniqueReferenceID,

			Currency: line.Currency,

			Price:      line.UsageBased.Price,
			FeatureKey: lo.EmptyableToPtr(line.UsageBased.FeatureKey),

			Subscription: line.Subscription,
		})
		if err != nil {
			return res, fmt.Errorf("creating split line group: %w", err)
		}

		splitLineGroupID = splitLineGroup.ID
	} else {
		splitLineGroupID = lo.FromPtr(line.SplitLineGroupID)
	}

	// We have alredy split the line once, we just need to create a new line and update the existing line
	postSplitAtLine := line.CloneWithoutDependencies(func(l *billing.StandardLine) {
		l.Period.Start = in.SplitAt
		l.SplitLineGroupID = lo.ToPtr(splitLineGroupID)

		l.ChildUniqueReferenceID = nil
	})

	postSplitAtLineSvc, err := s.lineService.FromEntity(postSplitAtLine, in.FeatureMeters)
	if err != nil {
		return res, fmt.Errorf("creating line service: %w", err)
	}

	if !postSplitAtLineSvc.IsPeriodEmptyConsideringTruncations() {
		gatheringInvoice.Lines.Append(postSplitAtLine)

		if err := postSplitAtLineSvc.Validate(ctx, &gatheringInvoice); err != nil {
			return res, fmt.Errorf("validating post split line: %w", err)
		}
	}

	// Let's update the original line to only contain the period up to the splitAt time
	line.Period.End = in.SplitAt
	line.InvoiceAt = in.SplitAt
	line.SplitLineGroupID = lo.ToPtr(splitLineGroupID)
	line.ChildUniqueReferenceID = nil

	preSplitAtLineSvc, err := s.lineService.FromEntity(line, in.FeatureMeters)
	if err != nil {
		return res, fmt.Errorf("creating line service: %w", err)
	}

	// If the line became empty, due to the split, let's remove it from the gathering invoice
	if preSplitAtLineSvc.IsPeriodEmptyConsideringTruncations() {
		line.DeletedAt = lo.ToPtr(clock.Now())
	} else {
		if err := preSplitAtLineSvc.Validate(ctx, &gatheringInvoice); err != nil {
			return res, fmt.Errorf("validating pre split line: %w", err)
		}
	}

	return splitGatheringInvoiceLineResult{
		PreSplitAtLine:   preSplitAtLineSvc,
		PostSplitAtLine:  postSplitAtLineSvc,
		GatheringInvoice: gatheringInvoice,
	}, nil
}

type createStandardInvoiceFromGatheringLinesInput struct {
	Customer                customer.Customer
	Currency                currencyx.Code
	GatheringInvoice        billing.StandardInvoice
	FeatureMeters           billing.FeatureMeters
	Lines                   billing.StandardLines
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

type createStandardInvoiceFromGatheringLinesResult struct {
	CreatedInvoice   billing.StandardInvoice
	GatheringInvoice billing.StandardInvoice
}

// createStandardInvoiceFromGatheringLines creates a standard invoice from the gathering invoice lines.
// Invariant:
// - the standard invoice is in draft.created state, and is calculated and persisted to the database
// - the gathering invoice's lines are removed, but not persisted to the database
func (s *Service) createStandardInvoiceFromGatheringLines(ctx context.Context, in createStandardInvoiceFromGatheringLinesInput) (*createStandardInvoiceFromGatheringLinesResult, error) {
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

	moveResults, err := s.moveLinesToInvoice(ctx, moveLinesToInvoiceInput{
		SourceGatheringInvoice: in.GatheringInvoice,
		TargetInvoice:          invoice,
		FeatureMeters:          in.FeatureMeters,
		LineIDsToMove:          lo.Map(in.Lines, func(l *billing.StandardLine, _ int) string { return l.ID }),
	})
	if err != nil {
		return nil, fmt.Errorf("moving lines to invoice: %w", err)
	}

	// Let's create the sub lines as per the meters (we are not setting the QuantitySnapshotedAt field just now, to signal that this is not the final snapshot)
	if err := s.snapshotLineQuantitiesInParallel(ctx, invoice.Customer, moveResults.LinesAssociated, in.FeatureMeters); err != nil {
		return nil, fmt.Errorf("snapshotting lines: %w", err)
	}

	// Let's persist the target invoice as the state machine always reloads the invoice from the database to make
	// sure we don't have any manual modifications inside the invoice structure.
	_, err = s.updateInvoice(ctx, moveResults.TargetInvoice)
	if err != nil {
		return nil, fmt.Errorf("updating target invoice: %w", err)
	}

	// Let's make sure that the invoice is in an up-to-date state
	invoice, err = s.withLockedInvoiceStateMachine(ctx, withLockedStateMachineInput{
		InvoiceID: invoice.InvoiceID(),
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

	return &createStandardInvoiceFromGatheringLinesResult{
		CreatedInvoice:   invoice,
		GatheringInvoice: moveResults.GatheringInvoice,
	}, nil
}

type moveLinesToInvoiceInput struct {
	SourceGatheringInvoice billing.StandardInvoice
	FeatureMeters          billing.FeatureMeters
	TargetInvoice          billing.StandardInvoice
	LineIDsToMove          []string
}

func (in moveLinesToInvoiceInput) Validate() error {
	if err := in.SourceGatheringInvoice.Validate(); err != nil {
		return fmt.Errorf("source gathering invoice: %w", err)
	}

	if in.SourceGatheringInvoice.Status != billing.StandardInvoiceStatusGathering {
		return fmt.Errorf("source gathering invoice must be in gathering status")
	}

	if err := in.TargetInvoice.Validate(); err != nil {
		return fmt.Errorf("target invoice: %w", err)
	}

	if in.TargetInvoice.Status != billing.StandardInvoiceStatusDraftCreated {
		return fmt.Errorf("target invoice must be in draft created status")
	}

	if len(in.LineIDsToMove) == 0 {
		return fmt.Errorf("line IDs to move is required")
	}

	if in.TargetInvoice.Currency != in.SourceGatheringInvoice.Currency {
		return fmt.Errorf("target invoice currency must be the same as source gathering invoice currency")
	}

	if in.TargetInvoice.ID == "" {
		return fmt.Errorf("target invoice ID is required")
	}

	if in.TargetInvoice.Namespace != in.SourceGatheringInvoice.Namespace {
		return fmt.Errorf("target invoice namespace must be the same as source gathering invoice namespace")
	}

	if in.FeatureMeters == nil {
		return fmt.Errorf("feature meters are required")
	}

	return nil
}

type moveLinesToInvoiceResult struct {
	GatheringInvoice billing.StandardInvoice
	TargetInvoice    billing.StandardInvoice
	LinesAssociated  billing.StandardLines
}

// moveLinesToInvoice moves the lines from the source gathering invoice to the target invoice, invariants:
// - the source gathering invoice is updated by removing the lines that have been moved to the target invoice
// - the target invoice is updated by adding the lines that have been moved from the source gathering invoice
// - neither invoices are saved to the database, they are returned as is
func (s *Service) moveLinesToInvoice(ctx context.Context, in moveLinesToInvoiceInput) (*moveLinesToInvoiceResult, error) {
	if err := in.Validate(); err != nil {
		return nil, fmt.Errorf("validating input: %w", err)
	}

	srcInvoice := in.SourceGatheringInvoice
	dstInvoice := in.TargetInvoice

	// Let's find the lines to move from the source gathering invoice
	linesToMove := lo.Filter(srcInvoice.Lines.OrEmpty(), func(line *billing.StandardLine, _ int) bool {
		return slices.Contains(in.LineIDsToMove, line.ID)
	})

	if len(linesToMove) != len(in.LineIDsToMove) {
		return nil, fmt.Errorf("lines to move[%d] must contain the same number of lines as line IDs to move[%d]", len(linesToMove), len(in.LineIDsToMove))
	}

	linesToAssociate, err := s.lineService.FromEntities(linesToMove, in.FeatureMeters)
	if err != nil {
		return nil, fmt.Errorf("creating line services for lines to move: %w", err)
	}

	if err := linesToAssociate.ValidateForInvoice(ctx, &dstInvoice); err != nil {
		return nil, fmt.Errorf("validating lines to move: %w", err)
	}

	// Let's set the invoice ID of the lines to move to the target invoice ID
	for _, line := range linesToMove {
		line.InvoiceID = dstInvoice.ID
	}

	// Let's add the lines to the target invoice
	dstInvoice.Lines.Append(linesToMove...)

	// Let's remove the lines from the source gathering invoice
	for _, line := range linesToMove {
		if !srcInvoice.Lines.RemoveByID(line.ID) {
			return nil, fmt.Errorf("line[%s] not found in source gathering invoice", line.ID)
		}
	}

	return &moveLinesToInvoiceResult{
		GatheringInvoice: srcInvoice,
		TargetInvoice:    dstInvoice,
		LinesAssociated:  linesToMove,
	}, nil
}

// updateGatheringInvoice updates the gathering invoice's state and if it contains no lines, it will be deleted.
// Invariant:
// - the invoice is recalculated
// - the invoice is updated to the database
func (s *Service) updateGatheringInvoice(ctx context.Context, invoice billing.StandardInvoice) (billing.StandardInvoice, error) {
	// Let's update the invoice's state
	if err := s.invoiceCalculator.CalculateGatheringInvoice(&invoice); err != nil {
		return billing.StandardInvoice{}, fmt.Errorf("calculating gathering invoice: %w", err)
	}

	// The gathering invoice has no lines => delete the invoice
	if invoice.Lines.NonDeletedLineCount() == 0 {
		invoice.DeletedAt = lo.ToPtr(clock.Now())
	}

	invoice, err := s.adapter.UpdateInvoice(ctx, invoice)
	if err != nil {
		return billing.StandardInvoice{}, fmt.Errorf("updating gathering invoice: %w", err)
	}

	return invoice, nil
}
