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
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
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
			billableLines, err := s.PrepareBillableLines(ctx, input)
			if err != nil {
				return nil, fmt.Errorf("preparing billable lines: %w", err)
			}

			if billableLines == nil {
				// Should not happen, but we want to be defensive, but we are not surfacing this error to the caller.
				return nil, fmt.Errorf("billable lines are nil")
			}

			createdInvoices := make([]billing.StandardInvoice, 0, len(billableLines.LinesByCurrency))

			for currency, inScopeLines := range billableLines.LinesByCurrency {
				createdInvoice, err := s.CreateStandardInvoiceFromGatheringLines(ctx, billing.CreateStandardInvoiceFromGatheringLinesInput{
					Customer: input.Customer,
					Currency: currency,
					Lines:    inScopeLines,
				})
				if err != nil {
					return nil, fmt.Errorf("creating standard invoice from gathering lines: %w", err)
				}

				createdInvoices = append(createdInvoices, *createdInvoice)
			}

			return createdInvoices, nil
		})
}

func (s *Service) PrepareBillableLines(ctx context.Context, input billing.PrepareBillableLinesInput) (*billing.PrepareBillableLinesResult, error) {
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
		func(ctx context.Context) (*billing.PrepareBillableLinesResult, error) {
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

			linesToBeBilledByCurrency := make(map[currencyx.Code]billing.GatheringLines)

			for currency, inScopeLines := range inScopeLinesByCurrency {
				// Let's first make sure we have properly split the progressively billed
				// lines into multiple lines on the gathering invoice if needed.
				gatheringInvoice, ok := invoicesByCurrency[currency]
				if !ok {
					return nil, fmt.Errorf("gathering invoice for currency [%s] not found", currency)
				}

				if len(inScopeLines) == 0 {
					continue
				}

				// Step 1: Let's make sure we have lines properly split on the gathering invoice.
				// Invariant: the gathering invoice is updated to contain the new lines if any were split.
				prepareResults, err := s.prepareLinesToBill(ctx, prepareLinesToBillInput{
					GatheringInvoice: gatheringInvoice.Invoice,
					FeatureMeters:    gatheringInvoice.FeatureMeters,
					InScopeLines:     inScopeLines,
				})
				if err != nil {
					return nil, fmt.Errorf("gathering lines to bill: %w", err)
				}

				if prepareResults == nil {
					continue
				}

				linesToBeBilledByCurrency[currency] = prepareResults.LinesToBill
			}

			totalLinesToBeBilled := 0
			for _, lines := range linesToBeBilledByCurrency {
				totalLinesToBeBilled += len(lines)
			}

			if totalLinesToBeBilled == 0 {
				return nil, billing.ValidationError{
					Err: billing.ErrInvoiceCreateNoLines,
				}
			}

			return &billing.PrepareBillableLinesResult{
				LinesByCurrency: linesToBeBilledByCurrency,
			}, nil
		})
}

type gatheringLineWithBillablePeriod struct {
	Line           billing.GatheringLine
	BillablePeriod timeutil.ClosedPeriod
	Engine         billing.LineEngine
}

type gatheringInvoiceWithFeatureMeters struct {
	Invoice       billing.GatheringInvoice
	FeatureMeters feature.FeatureMeters
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
		linesWithResolvedPeriods, err := slicesx.MapWithErr(invoice.Invoice.Lines.OrEmpty(), func(line billing.GatheringLine) (gatheringLineWithBillablePeriod, error) {
			period, err := s.ratingService.ResolveBillablePeriod(rating.ResolveBillablePeriodInput{
				Line:               line,
				FeatureMeters:      invoice.FeatureMeters,
				ProgressiveBilling: in.ProgressiveBilling,
				AsOf:               in.AsOf,
			})
			if err != nil {
				return gatheringLineWithBillablePeriod{}, fmt.Errorf("resolving billable period[%s]: %w", line.ID, err)
			}

			eng, err := s.lineEngines.Get(line.Engine)
			if err != nil {
				return gatheringLineWithBillablePeriod{}, fmt.Errorf("getting engine[%s]: %w", line.ID, err)
			}

			engineInput := billing.IsLineBillableAsOfInput{
				Line:                   line,
				AsOf:                   in.AsOf,
				ProgressiveBilling:     in.ProgressiveBilling,
				FeatureMeters:          invoice.FeatureMeters,
				ResolvedBillablePeriod: lo.FromPtr(period),
			}
			if err := engineInput.Validate(); err != nil {
				return gatheringLineWithBillablePeriod{}, fmt.Errorf("validating billable status input[%s]: %w", line.ID, err)
			}

			isBillable, err := eng.IsLineBillableAsOf(ctx, engineInput)
			if err != nil {
				return gatheringLineWithBillablePeriod{}, fmt.Errorf("checking billable status[%s]: %w", line.ID, err)
			}

			if !isBillable {
				return gatheringLineWithBillablePeriod{
					Line:   line,
					Engine: eng,
				}, nil
			}

			return gatheringLineWithBillablePeriod{
				Line:           line,
				BillablePeriod: lo.FromPtr(period),
				Engine:         eng,
			}, nil
		})
		if err != nil {
			return nil, fmt.Errorf("gathering lines with resolved periods: %w", err)
		}

		linesWithResolvedPeriods = lo.Filter(linesWithResolvedPeriods, func(line gatheringLineWithBillablePeriod, _ int) bool {
			return !lo.IsEmpty(line.BillablePeriod)
		})

		if !in.ProgressiveBilling {
			// Somewhat of a hack: Since we are allowing subscriptions with different billing periods for ratecards, invoiceAt not necessarily equals
			// to the line's period start and end time.

			// So we have two kinds of progressive billing scenarios:
			// 1. the line needs to be split into multiple lines
			// 2. the line does not need to be split but it's invoiceAt is after the line's period end, when the line is technically billable, but
			//    from the user's perspective as they are not requesting progressive billing we should not include it on the invoice.

			linesWithResolvedPeriods = lo.Filter(linesWithResolvedPeriods, func(line gatheringLineWithBillablePeriod, _ int) bool {
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
	FeatureMeters      feature.FeatureMeters
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
	FeatureMeters    feature.FeatureMeters
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
		currentLine, found := gatheringInvoice.Lines.GetByID(line.Line.ID)
		if !found {
			return nil, fmt.Errorf("line[%s]: line not found in gathering invoice[%s]", line.Line.ID, gatheringInvoice.ID)
		}

		if !currentLine.ServicePeriod.Equal(line.BillablePeriod) {
			// We need to split the line into multiple lines
			if !currentLine.ServicePeriod.From.Equal(line.BillablePeriod.From) {
				return nil, fmt.Errorf("line[%s]: line period start[%s] is not equal to billable period start[%s]", currentLine.ID, currentLine.ServicePeriod.From, line.BillablePeriod.From)
			}

			engineInput := billing.SplitGatheringLineInput{
				Line:          currentLine,
				FeatureMeters: input.FeatureMeters,
				SplitAt:       line.BillablePeriod.To,
			}
			if err := engineInput.Validate(); err != nil {
				return nil, fmt.Errorf("line[%s]: validating split input: %w", currentLine.ID, err)
			}

			splitLine, err := line.Engine.SplitGatheringLine(ctx, engineInput)
			if err != nil {
				return nil, fmt.Errorf("line[%s]: splitting line: %w", currentLine.ID, err)
			}

			if err := splitLine.Validate(); err != nil {
				return nil, fmt.Errorf("line[%s]: validating split output: %w", currentLine.ID, err)
			}

			if splitLine.PreSplitAtLine.DeletedAt != nil {
				if err := gatheringInvoice.Lines.ReplaceByID(splitLine.PreSplitAtLine); err != nil {
					return nil, fmt.Errorf("line[%s]: merging deleted pre split line: %w", currentLine.ID, err)
				}

				if splitLine.PostSplitAtLine != nil {
					gatheringInvoice.Lines.Append(*splitLine.PostSplitAtLine)
				}

				wasSplit = true

				s.logger.WarnContext(ctx, "pre split line is nil, skipping collection",
					"line", currentLine.ID,
					"original_period_start", currentLine.ServicePeriod.From,
					"original_period_end", currentLine.ServicePeriod.To,
					"split_at", line.BillablePeriod.To)
				continue
			}

			if err := gatheringInvoice.Lines.ReplaceByID(splitLine.PreSplitAtLine); err != nil {
				return nil, fmt.Errorf("line[%s]: merging pre split line: %w", currentLine.ID, err)
			}

			if splitLine.PostSplitAtLine != nil {
				gatheringInvoice.Lines.Append(*splitLine.PostSplitAtLine)
			}

			invoiceLines = append(invoiceLines, splitLine.PreSplitAtLine)
			wasSplit = true
		} else {
			invoiceLines = append(invoiceLines, currentLine)
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

// createStandardInvoiceFromGatheringLines creates a standard invoice from the gathering invoice lines.
// Invariant:
// - the standard invoice is in draft.created state, and is calculated and persisted to the database
// - the gathering invoice's lines are deleted, and persisted to the database
func (s *Service) CreateStandardInvoiceFromGatheringLines(ctx context.Context, in billing.CreateStandardInvoiceFromGatheringLinesInput) (*billing.StandardInvoice, error) {
	if err := in.Validate(); err != nil {
		return nil, fmt.Errorf("validating input: %w", err)
	}

	if err := validateUniqueChargeIDs(in.Lines); err != nil {
		return nil, fmt.Errorf("validating gathering lines: %w", err)
	}

	profile, err := s.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: in.Customer,
		Expand: billing.CustomerOverrideExpand{
			Customer: true,
			Apps:     true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("fetching customer profile: %w", err)
	}

	invoiceNumber, err := s.GenerateInvoiceSequenceNumber(ctx,
		billing.SequenceGenerationInput{
			Namespace:    in.Customer.Namespace,
			CustomerName: profile.Customer.Name,
			Currency:     in.Currency,
		},
		billing.DraftInvoiceSequenceNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("generating invoice number: %w", err)
	}

	if err := s.resolveDefaultTaxCode(ctx, in.Customer.Namespace, profile.MergedProfile.WorkflowConfig.Invoicing.DefaultTaxConfig); err != nil {
		return nil, fmt.Errorf("resolving default tax code: %w", err)
	}

	// let's create the invoice
	invoice, err := s.adapter.CreateInvoice(ctx, billing.CreateInvoiceAdapterInput{
		Namespace: in.Customer.Namespace,
		Customer:  lo.FromPtr(profile.Customer),
		Profile:   profile.MergedProfile,

		Currency:    in.Currency,
		Number:      invoiceNumber,
		Status:      billing.StandardInvoiceStatusDraftCreated,
		Description: in.Description,

		Type: billing.InvoiceTypeStandard,
	})
	if err != nil {
		return nil, fmt.Errorf("creating invoice: %w", err)
	}

	// let's set the workflow apps as some checks such as CanDraftSyncAdvance depends on the apps
	invoice.Workflow.Apps = profile.MergedProfile.Apps

	linesWithEngines, err := s.lineEngines.groupGatheringLinesByEngine(in.Lines)
	if err != nil {
		return nil, fmt.Errorf("grouping lines by engine: %w", err)
	}

	for _, item := range linesWithEngines {
		engineInput := billing.BuildStandardInvoiceLinesInput{
			Invoice:        invoice,
			GatheringLines: item.Lines,
		}
		if err := engineInput.Validate(); err != nil {
			return nil, fmt.Errorf("validating build standard invoice lines input for engine %s: %w", item.Engine.GetLineEngineType(), err)
		}

		stdLines, err := item.Engine.BuildStandardInvoiceLines(ctx, engineInput)
		if err != nil {
			return nil, fmt.Errorf("building standard invoice lines for engine %s: %w", item.Engine.GetLineEngineType(), err)
		}

		if err := stdLines.Validate(); err != nil {
			return nil, fmt.Errorf("validating build standard invoice lines output for engine %s: %w", item.Engine.GetLineEngineType(), err)
		}

		expectedIDs := lo.Map(item.Lines, func(line billing.GatheringLine, _ int) string {
			return line.ID
		})
		actualIDs := lo.Map(stdLines, func(line *billing.StandardLine, _ int) string {
			return line.ID
		})
		if !lo.ElementsMatch(expectedIDs, actualIDs) {
			return nil, fmt.Errorf(
				"build standard invoice lines ids mismatch for engine %s: expected %v, got %v",
				item.Engine.GetLineEngineType(),
				expectedIDs,
				actualIDs,
			)
		}

		invoice.Lines.Append(stdLines...)
	}

	affectedGatheringInvoiceIDs := lo.Uniq(lo.Map(in.Lines, func(line billing.GatheringLine, _ int) billing.InvoiceID {
		return billing.InvoiceID{Namespace: line.Namespace, ID: line.InvoiceID}
	}))

	for _, gatheringInvoiceID := range affectedGatheringInvoiceIDs {
		gatheringInvoice, err := s.adapter.GetGatheringInvoiceById(ctx, billing.GetGatheringInvoiceByIdInput{
			Invoice: gatheringInvoiceID,
			Expand: billing.GatheringInvoiceExpands{
				billing.GatheringInvoiceExpandLines,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("fetching gathering invoice: %w", err)
		}

		invoiceLinesToRemove := lo.Filter(in.Lines, func(line billing.GatheringLine, _ int) bool {
			return billing.InvoiceID{Namespace: gatheringInvoice.Namespace, ID: line.InvoiceID} == gatheringInvoiceID
		})

		// Let's first update the gathering invoice to make sure deleted lines are synced, as the standard invoice will have expanded split line hierarchies
		// and we need to make sure that gathering invoice lines that are already yielded the standard invoice lines are excluded from the split line hierarchy.
		//
		// Note: this is a hack, on the long term we need to have a Charge type that encapsulates all of this logic.
		err = s.removeLinesFromGatheringInvoice(ctx, gatheringInvoice, invoiceLinesToRemove)
		if err != nil {
			return nil, fmt.Errorf("updating gathering invoice: %w", err)
		}
	}

	// Prerequisite: we should have the split line group headers expanded so that snapshotting can determine if the preLine
	// queries are needed.
	// Let's persist the snapshotted values to the database as the state machine always reloads the invoice from the database to make
	// sure we don't have any manual modifications inside the invoice structure.
	invoice, err = s.updateInvoice(ctx, invoice)
	if err != nil {
		return nil, fmt.Errorf("updating target invoice: %w", err)
	}

	invoice, err = s.invokeOnStandardInvoiceCreated(ctx, invoice)
	if err != nil {
		return nil, fmt.Errorf("processing standard invoice created hooks: %w", err)
	}

	invoice, err = s.recalculateStandardInvoice(ctx, invoice)
	if err != nil {
		return nil, fmt.Errorf("recalculating target invoice after standard invoice created hooks: %w", err)
	}

	if err := s.standardInvoiceHooks.PostCreate(ctx, &invoice); err != nil {
		return nil, fmt.Errorf("invoking post create hooks: %w", err)
	}

	// Let's make sure that the invoice is in an up-to-date state
	invoice, err = s.withLockedInvoiceStateMachine(ctx, withLockedStateMachineInput{
		InvoiceID: invoice.GetInvoiceID(),
		Callback: func(ctx context.Context, sm *InvoiceStateMachine) error {
			// Let's activate the state machine so that the created state's calculation is triggered
			if err := sm.StateMachine.ActivateCtx(ctx); err != nil {
				return fmt.Errorf("activating invoice state machine: %w", err)
			}

			if in.PostCreationCalculationHook != nil {
				err := s.invokePostCreationHooks(sm.Invoice, in.PostCreationCalculationHook)
				if err != nil {
					return fmt.Errorf("invoking post creation calculation hook: %w", err)
				}

				// Let's recalculate the invoice so that any adjustments made in the hook are respresented in the calculations.
				if err := sm.calculateInvoice(ctx); err != nil {
					return fmt.Errorf("recalculating invoice: %w", err)
				}
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

	// Let's publish the created event
	event, err := billing.NewStandardInvoiceCreatedEvent(invoice)
	if err != nil {
		return nil, fmt.Errorf("creating event: %w", err)
	}

	err = s.publisher.Publish(ctx, event)
	if err != nil {
		return nil, fmt.Errorf("publishing event: %w", err)
	}

	return &invoice, nil
}

func (s *Service) invokeOnStandardInvoiceCreated(ctx context.Context, invoice billing.StandardInvoice) (billing.StandardInvoice, error) {
	groupedStandardLines, err := s.lineEngines.groupStandardLinesByEngine(invoice.Lines.OrEmpty())
	if err != nil {
		return billing.StandardInvoice{}, fmt.Errorf("grouping standard lines by engine: %w", err)
	}

	for _, grouped := range groupedStandardLines {
		input := billing.OnStandardInvoiceCreatedInput{
			Invoice: invoice,
			Lines:   grouped.Lines,
		}
		if err := input.Validate(); err != nil {
			return billing.StandardInvoice{}, fmt.Errorf("validating standard invoice created input for engine %s: %w", grouped.Engine.GetLineEngineType(), err)
		}

		lines, err := grouped.Engine.OnStandardInvoiceCreated(ctx, input)
		if err != nil {
			return billing.StandardInvoice{}, fmt.Errorf("standard invoice created for engine %s: %w", grouped.Engine.GetLineEngineType(), err)
		}

		if err := lines.Validate(); err != nil {
			return billing.StandardInvoice{}, fmt.Errorf("validating standard invoice created output for engine %s: %w", grouped.Engine.GetLineEngineType(), err)
		}

		if err := billing.ValidateStandardLineIDsMatchExactly(grouped.Lines, lines); err != nil {
			return billing.StandardInvoice{}, fmt.Errorf("validating standard invoice created line ids for engine %s: %w", grouped.Engine.GetLineEngineType(), err)
		}

		if err := invoice.Lines.ReplaceLinesByID(lines...); err != nil {
			return billing.StandardInvoice{}, fmt.Errorf("replacing standard invoice created lines for engine %s: %w", grouped.Engine.GetLineEngineType(), err)
		}
	}

	return invoice, nil
}

func (s *Service) recalculateStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) (billing.StandardInvoice, error) {
	featureMeters, err := s.resolveFeatureMeters(ctx, invoice.Namespace, invoice.Lines)
	if err != nil {
		return billing.StandardInvoice{}, fmt.Errorf("resolving feature meters: %w", err)
	}

	taxCodes, err := s.resolveTaxCodes(ctx, resolveTaxCodesInput{
		Namespace: invoice.Namespace,
		Invoice:   &invoice,
		ReadOnly:  false,
	})
	if err != nil {
		return billing.StandardInvoice{}, fmt.Errorf("resolving tax codes: %w", err)
	}

	if err := s.invoiceCalculator.Calculate(&invoice, invoicecalc.CalculatorDependencies{
		FeatureMeters: featureMeters,
		RatingService: s.ratingService,
		TaxCodes:      taxCodes,
		LineEngines:   s.lineEngines,
	}); err != nil {
		return billing.StandardInvoice{}, fmt.Errorf("recalculating target invoice: %w", err)
	}

	invoice, err = s.updateInvoice(ctx, invoice)
	if err != nil {
		return billing.StandardInvoice{}, fmt.Errorf("updating target invoice after line engine hook: %w", err)
	}

	return invoice, nil
}

func (s *Service) invokePostCreationHooks(invoice billing.StandardInvoice, hook billing.PostCreationCalculationHook) error {
	for _, line := range invoice.Lines.OrEmpty() {
		ops, err := hook(invoice, lo.FromPtr(line))
		if err != nil {
			return fmt.Errorf("invoking post creation hook: %w", err)
		}

		if len(ops) == 0 {
			continue
		}

		for _, op := range ops {
			if err := op(line); err != nil {
				return fmt.Errorf("invoking post creation hook: %w", err)
			}
		}
	}

	return nil
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

func validateUniqueChargeIDs(lines billing.GatheringLines) error {
	chargeIDs := lo.FilterMap(lines, func(line billing.GatheringLine, _ int) (string, bool) {
		if line.ChargeID == nil {
			return "", false
		}

		return *line.ChargeID, true
	})

	if len(chargeIDs) != len(lo.Uniq(chargeIDs)) {
		return fmt.Errorf("duplicate charge ids found: %v", chargeIDs)
	}

	return nil
}
