package billingworkersubscription

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

const (
	SubscriptionSyncComponentName billing.ComponentName = "subscription-sync"
)

type FeatureFlags struct {
	EnableFlatFeeInAdvanceProrating bool
	EnableFlatFeeInArrearsProrating bool
}

type Config struct {
	BillingService      billing.Service
	SubscriptionService subscription.Service
	TxCreator           transaction.Creator
	Logger              *slog.Logger
	FeatureFlags        FeatureFlags
}

func (c Config) Validate() error {
	if c.BillingService == nil {
		return fmt.Errorf("billing service is required")
	}

	if c.SubscriptionService == nil {
		return fmt.Errorf("subscription service is required")
	}

	if c.TxCreator == nil {
		return fmt.Errorf("transaction creator is required")
	}

	if c.Logger == nil {
		return fmt.Errorf("logger is required")
	}

	return nil
}

type Handler struct {
	billingService      billing.Service
	subscriptionService subscription.Service
	txCreator           transaction.Creator
	logger              *slog.Logger
	featureFlags        FeatureFlags
}

func New(config Config) (*Handler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &Handler{
		billingService:      config.BillingService,
		txCreator:           config.TxCreator,
		logger:              config.Logger,
		featureFlags:        config.FeatureFlags,
		subscriptionService: config.SubscriptionService,
	}, nil
}

func (h *Handler) SyncronizeSubscription(ctx context.Context, subs subscription.SubscriptionView, asOf time.Time) error {
	if subs.Subscription.HasAnnotation(subscription.AnnotationDisableBillingSync) {
		h.logger.Info("subscription sync disabled", "subscription_id", subs.Subscription.ID)
		return nil
	}

	// TODO[later]: Right now we are getting the billing profile as a validation step, but later if we allow more collection
	// alignment settings, we should use the collection settings from here to determine the generation end (overriding asof).
	_, err := h.billingService.GetCustomerOverride(
		ctx,
		billing.GetCustomerOverrideInput{
			Customer: customer.CustomerID{
				Namespace: subs.Subscription.Namespace,
				ID:        subs.Subscription.CustomerId,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("getting billing profile: %w", err)
	}

	currency, err := subs.Spec.Currency.Calculator()
	if err != nil {
		return fmt.Errorf("getting currency calculator: %w", err)
	}

	plan, err := h.calculateSyncPlan(ctx, subs, asOf)
	if err != nil {
		return err
	}

	if plan == nil {
		return nil
	}

	patches, err := h.getPatchesFromPlan(plan, subs, currency)
	if err != nil {
		return nil
	}

	return transaction.RunWithNoValue(ctx, h.txCreator, func(ctx context.Context) error {
		err := h.provisionPendingLines(ctx,
			subs,
			currency,
			plan.NewSubscriptionItems,
		)
		if err != nil {
			return fmt.Errorf("provisioning pending lines: %w", err)
		}

		patchesByInvoiceID := lo.GroupBy(patches, func(p linePatch) string {
			return p.InvoiceID
		})

		invoiceHeadersByID := make(map[string]billing.Invoice, len(patchesByInvoiceID))
		for invoiceID := range patchesByInvoiceID {
			invoice, err := h.billingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
				Invoice: billing.InvoiceID{
					Namespace: subs.Subscription.Namespace,
					ID:        invoiceID,
				},
				Expand: billing.InvoiceExpand{},
			})
			if err != nil {
				return fmt.Errorf("getting invoice[%s]: %w", invoiceID, err)
			}

			invoiceHeadersByID[invoiceID] = invoice
		}

		for invoiceID, patches := range patchesByInvoiceID {
			invoice := invoiceHeadersByID[invoiceID]

			if !invoice.StatusDetails.Immutable {
				if err := h.updateMutableInvoice(ctx, invoice, patches); err != nil {
					return fmt.Errorf("updating mutable invoice[%s]: %w", invoiceID, err)
				}
				continue
			}

			if err := h.updateImmutableInvoice(ctx, invoice, patches); err != nil {
				return fmt.Errorf("updating immutable invoice[%s]: %w", invoiceID, err)
			}
		}

		return nil
	})
}

type syncPlan struct {
	NewSubscriptionItems []subscriptionItemWithPeriod
	LinesToDelete        []*billing.Line
	LinesToUpsert        []syncPlanLineUpsert
}

type syncPlanLineUpsert struct {
	Target   subscriptionItemWithPeriod
	Existing *billing.Line
}

func (h *Handler) getPatchesFromPlan(p *syncPlan, subs subscription.SubscriptionView, currency currencyx.Calculator) ([]linePatch, error) {
	patches := make([]linePatch, 0, len(p.LinesToDelete)+len(p.LinesToUpsert))

	// Let's update the existing lines
	for _, line := range p.LinesToDelete {
		patches = append(patches, h.getDeletePatchesForLine(line)...)
	}

	for _, line := range p.LinesToUpsert {
		expectedLine, err := h.lineFromSubscritionRateCard(subs, line.Target, currency)
		if err != nil {
			return nil, fmt.Errorf("generating expected line[%s]: %w", line.Target.UniqueID, err)
		}

		if expectedLine == nil {
			// The line should be deleted, let's see how
			patches = append(patches, h.getDeletePatchesForLine(line.Existing)...)
			continue
		}

		updatePatches, err := h.inScopeLinePatches(line.Existing, expectedLine)
		if err != nil {
			return nil, fmt.Errorf("updating line[%s]: %w", line.Target.UniqueID, err)
		}

		patches = append(patches, updatePatches...)
	}

	return patches, nil
}

func (h *Handler) calculateSyncPlan(ctx context.Context, subs subscription.SubscriptionView, asOf time.Time) (*syncPlan, error) {
	// Let's see what's in scope for the subscription
	slices.SortFunc(subs.Phases, func(i, j subscription.SubscriptionPhaseView) int {
		return timeutil.Compare(i.SubscriptionPhase.ActiveFrom, j.SubscriptionPhase.ActiveFrom)
	})

	inScopeLines, err := h.collectUpcomingLines(subs, asOf)
	if err != nil {
		return nil, fmt.Errorf("collecting upcoming lines: %w", err)
	}

	if len(inScopeLines) == 0 {
		// The subscription has no invoicable items, so we can return early
		return nil, nil
	}

	inScopeLinesByUniqueID, unique := slicesx.UniqueGroupBy(inScopeLines, func(i subscriptionItemWithPeriod) string {
		return i.UniqueID
	})
	if !unique {
		return nil, fmt.Errorf("duplicate unique ids in the upcoming lines")
	}

	// Let's load the existing lines for the subscription
	existingLines, err := h.billingService.GetLinesForSubscription(ctx, billing.GetLinesForSubscriptionInput{
		Namespace:      subs.Subscription.Namespace,
		SubscriptionID: subs.Subscription.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("getting existing lines: %w", err)
	}

	existingLinesByUniqueID, unique := slicesx.UniqueGroupBy(
		lo.Filter(existingLines, func(l *billing.Line, _ int) bool {
			return l.ChildUniqueReferenceID != nil
		}),
		func(l *billing.Line) string {
			return *l.ChildUniqueReferenceID
		})
	if !unique {
		return nil, fmt.Errorf("duplicate unique ids in the existing lines")
	}

	existingLineUniqueIDs := lo.Keys(existingLinesByUniqueID)
	inScopeLineUniqueIDs := lo.Keys(inScopeLinesByUniqueID)
	// Let's execute the synchronization
	deletedLines, newLines := lo.Difference(existingLineUniqueIDs, inScopeLineUniqueIDs)
	lineIDsToUpsert := lo.Intersect(existingLineUniqueIDs, inScopeLineUniqueIDs)

	linesToDelete, err := slicesx.MapWithErr(deletedLines, func(id string) (*billing.Line, error) {
		line, ok := existingLinesByUniqueID[id]
		if !ok {
			return nil, fmt.Errorf("existing line[%s] not found in the existing lines", id)
		}

		return line, nil
	})
	if err != nil {
		return nil, fmt.Errorf("mapping deleted lines: %w", err)
	}

	linesToUpsert, err := slicesx.MapWithErr(lineIDsToUpsert, func(id string) (syncPlanLineUpsert, error) {
		existingLine, ok := existingLinesByUniqueID[id]
		if !ok {
			return syncPlanLineUpsert{}, fmt.Errorf("existing line[%s] not found in the existing lines", id)
		}

		return syncPlanLineUpsert{
			Target:   inScopeLinesByUniqueID[id],
			Existing: existingLine,
		}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("mapping upsert lines: %w", err)
	}

	return &syncPlan{
		NewSubscriptionItems: lo.Map(newLines, func(id string, _ int) subscriptionItemWithPeriod {
			return inScopeLinesByUniqueID[id]
		}),
		LinesToDelete: linesToDelete,
		LinesToUpsert: linesToUpsert,
	}, nil
}

// TODO[OM-1038]: manually deleted lines might come back to draft/gathering invoices (see ticket)

// collectUpcomingLines collects the upcoming lines for the subscription, if it does not return any lines the subscription doesn't
// have any invoicable items.
//
// AsOf is a guideline for the end of generation, but the actual end of generation can be different based on the collection (as we
// always yield at least one line if an invoicable line exists).
//
// This approach allows us to not to have to poll all the subscriptions periodically, but we can act when an invoice is created or when
// a subscription is updated.
func (h *Handler) collectUpcomingLines(subs subscription.SubscriptionView, asOf time.Time) ([]subscriptionItemWithPeriod, error) {
	inScopeLines := make([]subscriptionItemWithPeriod, 0, 128)

	for _, phase := range subs.Phases {
		iterator, err := NewPhaseIterator(subs, phase.SubscriptionPhase.Key)
		if err != nil {
			return nil, fmt.Errorf("creating phase iterator: %w", err)
		}

		if !iterator.HasInvoicableItems() {
			continue
		}

		generationLimit := asOf
		if phaseStart := iterator.PhaseStart(); phaseStart.After(asOf) {
			// We need to have invoicable items, so we need to advance the limit here at least to phaseStart to see
			// if we can have any invoicable items.

			generationLimit = iterator.GetMinimumBillableTime()

			if generationLimit.IsZero() {
				// This should not happen, but if it does, we should skip this phase
				continue
			}
		}

		items, err := iterator.Generate(generationLimit)
		if err != nil {
			return nil, fmt.Errorf("generating items: %w", err)
		}

		inScopeLines = append(inScopeLines, items...)

		if phaseEnd := iterator.PhaseEnd(); phaseEnd != nil && !phaseEnd.Before(asOf) {
			// we are done with the generation, as the phase end is after the asOf, and we have invoicable items
			break
		}
	}

	return inScopeLines, nil
}

func (h *Handler) getDeletePatchesForLine(line *billing.Line) []linePatch {
	patches := make([]linePatch, 0, 1+len(line.Children.OrEmpty()))

	if line.DeletedAt == nil {
		patches = append(patches, patchFromLine(patchOpDelete, line))
	}

	if line.Status == billing.InvoiceLineStatusSplit {
		// let's delete all the children
		for _, child := range line.Children.OrEmpty() {
			if child.DeletedAt != nil {
				continue
			}

			patches = append(patches, patchFromLine(patchOpDelete, child))
		}
	}

	return patches
}

func (h *Handler) lineFromSubscritionRateCard(subs subscription.SubscriptionView, item subscriptionItemWithPeriod, currency currencyx.Calculator) (*billing.Line, error) {
	line := &billing.Line{
		LineBase: billing.LineBase{
			Namespace:              subs.Subscription.Namespace,
			Name:                   item.Spec.RateCard.Name,
			Description:            item.Spec.RateCard.Description,
			ManagedBy:              billing.SubscriptionManagedLine,
			Currency:               subs.Spec.Currency,
			Status:                 billing.InvoiceLineStatusValid,
			ChildUniqueReferenceID: &item.UniqueID,
			TaxConfig:              item.Spec.RateCard.TaxConfig,
			Period:                 item.Period,

			Subscription: &billing.SubscriptionReference{
				SubscriptionID: subs.Subscription.ID,
				PhaseID:        item.PhaseID,
				ItemID:         item.SubscriptionItem.ID,
			},
		},
	}

	// In advance changes should always be invoiced immediately
	inAdvanceInvoiceAt := item.Period.Start

	inArrearsInvoiceAt := item.Period.End
	if item.InvoiceAligned {
		inArrearsInvoiceAt = item.NonTruncatedPeriod.End
	}

	switch item.SubscriptionItem.RateCard.Price.Type() {
	case productcatalog.FlatPriceType:
		price, err := item.SubscriptionItem.RateCard.Price.AsFlat()
		if err != nil {
			return nil, fmt.Errorf("converting price to flat: %w", err)
		}

		// TODO[OM-1040]: We should support rounding errors in prorating calculations (such as 1/3 of a dollar is $0.33, 3*$0.33 is $0.99, if we bill
		// $1.00 in three equal pieces we should charge the customer $0.01 as the last split)
		perUnitAmount := currency.RoundToPrecision(price.Amount)
		if !item.Period.IsEmpty() && h.shouldProrateFlatFee(price) {
			perUnitAmount = currency.RoundToPrecision(price.Amount.Mul(item.PeriodPercentage()))
		}

		switch price.PaymentTerm {
		case productcatalog.InArrearsPaymentTerm:
			line.InvoiceAt = inArrearsInvoiceAt
		case productcatalog.InAdvancePaymentTerm:
			// In case of inAdvance we should always invoice at the start of the period and if there's a change
			// prorating should void the item and credit the customer.
			//
			// Warning: We are not supporting voiding or crediting right now, so we are going to overcharge on
			// inAdvance items in case of a change on a finalized invoice.
			line.InvoiceAt = inAdvanceInvoiceAt
		default:
			return nil, fmt.Errorf("unsupported payment term: %v", price.PaymentTerm)
		}

		if perUnitAmount.IsZero() {
			// We don't need to bill the customer for zero amount items (zero amount items are not allowed on the lines
			// either, so we can safely return here)
			return nil, nil
		}

		line.Type = billing.InvoiceLineTypeFee
		line.FlatFee = &billing.FlatFeeLine{
			PerUnitAmount: perUnitAmount,
			Quantity:      alpacadecimal.NewFromInt(1),
			PaymentTerm:   price.PaymentTerm,
			Category:      billing.FlatFeeCategoryRegular,
		}

	case productcatalog.UnitPriceType, productcatalog.TieredPriceType:
		// Should not happen, but let's be safe
		if item.SubscriptionItem.RateCard.FeatureKey == nil {
			return nil, fmt.Errorf("feature must be defined for usage based price")
		}

		if item.SubscriptionItem.RateCard.Price == nil {
			return nil, fmt.Errorf("price must be defined for usage based price")
		}

		line.Type = billing.InvoiceLineTypeUsageBased
		line.InvoiceAt = inArrearsInvoiceAt
		line.UsageBased = &billing.UsageBasedLine{
			Price:      item.SubscriptionItem.RateCard.Price,
			FeatureKey: *item.SubscriptionItem.RateCard.FeatureKey,
		}

	default:
		return nil, fmt.Errorf("unsupported price type: %v", item.SubscriptionItem.RateCard.Price.Type())
	}

	return line, nil
}

func (h *Handler) shouldProrateFlatFee(price productcatalog.FlatPrice) bool {
	switch price.PaymentTerm {
	case productcatalog.InAdvancePaymentTerm:
		return h.featureFlags.EnableFlatFeeInAdvanceProrating
	case productcatalog.InArrearsPaymentTerm:
		return h.featureFlags.EnableFlatFeeInArrearsProrating
	default:
		return false
	}
}

func (h *Handler) provisionPendingLines(ctx context.Context, subs subscription.SubscriptionView, currency currencyx.Calculator, line []subscriptionItemWithPeriod) error {
	newLines, err := slicesx.MapWithErr(line, func(subsItem subscriptionItemWithPeriod) (*billing.LineWithCustomer, error) {
		line, err := h.lineFromSubscritionRateCard(subs, subsItem, currency)
		if err != nil {
			return nil, fmt.Errorf("generating line[%s]: %w", line.ID, err)
		}

		if line == nil {
			// We map all items to ItemWithPeriod that have a Price defined, however, we might deliberater later that for some de-facto 0 prices
			// we don't want to generate lines. This is the case for instance with 0 flat prices
			return nil, nil
		}

		return &billing.LineWithCustomer{
			Line:       *line,
			CustomerID: subs.Subscription.CustomerId,
		}, nil
	})
	if err != nil {
		return fmt.Errorf("creating new lines: %w", err)
	}

	_, err = h.billingService.CreatePendingInvoiceLines(ctx, billing.CreateInvoiceLinesInput{
		Namespace: subs.Subscription.Namespace,
		Lines: func() []billing.LineWithCustomer {
			lines := make([]billing.LineWithCustomer, 0, len(newLines))

			for _, l := range newLines {
				if l != nil {
					lines = append(lines, *l)
				}
			}

			return lines
		}(),
	})
	if err != nil {
		return fmt.Errorf("creating pending invoice lines: %w", err)
	}

	return nil
}

func (h *Handler) inScopeLinePatches(existingLine *billing.Line, expectedLine *billing.Line) ([]linePatch, error) {
	// TODO/WARNING[later]: This logic should be fine with everything that can be billed progressively, however the following use-cases
	// will behave strangely:
	//
	// - An in advance flat fee cannot be prorated, as that would require void/credit logic.
	// - If a volume based item's tiered are changed, then the old volume based item will be billed at that rate, and the
	//   new volume based item's calculation will start from the new tiered rates beginning. (e.g. if we have a 1 month long tiered
	//   price, then we change the tiered price in the middle of the month, the old tiered price will be billed for the first half
	//   and the new tiered price will be billed for the second half, meaning that the customer will be billed for less sum traffic) [OM]
	// - If a meter is unique counted, then the unique count will be reset at the change's time

	// This is a non-split line, so it's either assigned to a gathering invoice or an already paid invoice, we can just update the line
	// and let billing service handle the rest
	if existingLine.Status == billing.InvoiceLineStatusValid {
		if existingLine.ManagedBy != billing.SubscriptionManagedLine {
			return nil, nil
		}

		mergedLine, wasChange := h.mergeChangesFromLine(h.cloneLineForUpsert(existingLine), expectedLine)
		if !wasChange {
			return nil, nil
		}

		return []linePatch{
			patchFromLine(patchOpUpdate, mergedLine),
		}, nil
	}

	// Parts of the line has been already invoiced using progressive invoicing, so we need to examine the children
	if existingLine.Status == billing.InvoiceLineStatusSplit {
		if existingLine.Type == billing.InvoiceLineTypeFee {
			// This is defensive programming, as the existing split logic should not allow flat fee lines to be split
			// but if we implement progressive billing for flat fee lines, we need to handle the split here
			return nil, fmt.Errorf("flat fee lines should not be split [lineID=%s]", existingLine.ID)
		}

		// Nothing to do here, as split lines are UBP lines and thus we don't need the flat fee corrections
		// TODO[later]: When we implement progressive billing based pro-rating, we need to support adjusting flat fee
		// segments here.

		if existingLine.Period.End.Equal(expectedLine.Period.End) {
			// The line is already in the expected state, so we can safely return here
			return nil, nil
		}

		patches := []linePatch{}

		switch {
		case existingLine.Period.End.Before(expectedLine.Period.End):

			// Expansion of the line (e.g. continue subscription)

			children := existingLine.Children.OrEmpty()
			if len(children) > 0 {
				slices.SortFunc(children, func(i, j *billing.Line) int {
					return timeutil.Compare(i.Period.End, j.Period.End)
				})

				lastChild := h.cloneLineForUpsert(children[len(children)-1])

				if lastChild.ManagedBy == billing.SubscriptionManagedLine {
					// We are not supporting period changes for children, and we need to maintain the consistency so
					// even for overridden lines we need to update the period

					// We however allow deletions, so we are only un-deleting the line here if it was deleted by the sync engine
					lastChild.DeletedAt = nil
				}

				lastChild.Period.End = expectedLine.Period.End
				lastChild.InvoiceAt = expectedLine.Period.End
				patches = append(patches, patchFromLine(patchOpUpdate, lastChild))
			}

			// We have already updated the last child, so we need to update at least the periods regardless of managed_by to keep the consistency
			updatedExistingLine := h.cloneLineForUpsert(existingLine)
			updatedExistingLine.Period.End = expectedLine.Period.End
			updatedExistingLine.InvoiceAt = expectedLine.Period.End

			if updatedExistingLine.ManagedBy == billing.SubscriptionManagedLine {
				updatedExistingLine.DeletedAt = nil
			}

			patches = append(patches, patchFromLine(patchOpUpdate, updatedExistingLine))
		case existingLine.Period.End.After(expectedLine.Period.End):
			// Shrink of the line (e.g. subscription cancled, subscription item edit)

			for _, child := range existingLine.Children.OrEmpty() {
				if child.Period.End.Before(expectedLine.Period.End) {
					// The child is not affected by the period shrink, so we can skip it
					continue
				}

				if child.Period.Start.After(expectedLine.Period.End) {
					// The child is after the period shrink, so we need to delete it as it became invalid
					patches = append(patches, patchFromLine(patchOpDelete, child))
					continue
				}

				// The child is partially affected by the period shrink, so we need to adjust the period
				if !child.Period.End.Equal(expectedLine.Period.End) {
					updatedChild := h.cloneLineForUpsert(child)
					updatedChild.Period.End = expectedLine.Period.End

					if updatedChild.InvoiceAt.After(expectedLine.Period.End) {
						// The child is invoiced after the period end, so we need to adjust the invoice date
						updatedChild.InvoiceAt = expectedLine.Period.End
					}

					if child.ManagedBy == billing.SubscriptionManagedLine {
						updatedChild.DeletedAt = nil
					}

					patches = append(patches, patchFromLine(patchOpUpdate, updatedChild))
				}
			}
			// Split lines are always associated with gathering invoices, so we can safely update the line without checking for
			// snapshot update requirements

			updatedExistingLine := h.cloneLineForUpsert(existingLine)
			updatedExistingLine.Period.End = expectedLine.Period.End
			updatedExistingLine.InvoiceAt = expectedLine.Period.End

			patches = append(patches, patchFromLine(patchOpUpdate, updatedExistingLine))
		default:
			return nil, fmt.Errorf("could not handle split line update [lineID=%s, status=%s]", existingLine.ID, existingLine.Status)
		}

		return patches, nil
	}

	// There is no other state in which a line can be in, so we can safely return an error here
	return nil, fmt.Errorf("could not handle line update [lineID=%s, status=%s]", existingLine.ID, existingLine.Status)
}

type typeWithEqual[T any] interface {
	Equal(T) bool
}

func setIfDoesNotEqual[T typeWithEqual[T]](existing *T, expected T, wasChange *bool) {
	if !(*existing).Equal(expected) {
		*existing = expected
		*wasChange = true
	}
}

func (h *Handler) mergeChangesFromLine(existingLine *billing.Line, expectedLine *billing.Line) (*billing.Line, bool) {
	// We assume that only the period can change, maybe some pricing data due to prorating (for flat lines)

	wasChange := false

	setIfDoesNotEqual(&existingLine.Period, expectedLine.Period, &wasChange)
	setIfDoesNotEqual(&existingLine.InvoiceAt, expectedLine.InvoiceAt, &wasChange)

	if existingLine.DeletedAt != nil {
		existingLine.DeletedAt = nil
		wasChange = true
	}

	// Let's handle the flat fee prorating
	if existingLine.Type == billing.InvoiceLineTypeFee {
		setIfDoesNotEqual(&existingLine.FlatFee.PerUnitAmount, expectedLine.FlatFee.PerUnitAmount, &wasChange)
	}

	return existingLine, wasChange
}

func (h *Handler) updateMutableInvoice(ctx context.Context, invoice billing.Invoice, patches []linePatch) error {
	updatedInvoice, err := h.billingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
		Invoice:             invoice.InvoiceID(),
		IncludeDeletedLines: true,
		EditFn: func(invoice *billing.Invoice) error {
			for _, patch := range patches {
				line := invoice.Lines.GetByID(patch.LineID)
				if line == nil {
					return fmt.Errorf("line[%s] not found in the invoice, cannot delete", patch.LineID)
				}

				if patch.Op == patchOpDelete {
					line.DeletedAt = lo.ToPtr(clock.Now())
					continue
				}

				// update
				updatedLine := patch.TargetState

				if invoice.Status != billing.InvoiceStatusGathering && patch.TargetState.Type == billing.InvoiceLineTypeUsageBased {
					// We need to update the quantities of the usage based lines, to compensate for any changes in the period
					// of the line

					updatedQtyLine, err := h.billingService.SnapshotLineQuantity(ctx, billing.SnapshotLineQuantityInput{
						Invoice: invoice,
						Line:    updatedLine,
					})
					if err != nil {
						return fmt.Errorf("recalculating line[%s]: %w", updatedLine.ID, err)
					}

					updatedLine = updatedQtyLine
				}

				if ok := invoice.Lines.ReplaceByID(patch.LineID, updatedLine); !ok {
					return fmt.Errorf("line[%s/%s] not found in the invoice, cannot update", patch.LineID, lo.FromPtrOr(patch.TargetState.ChildUniqueReferenceID, "nil"))
				}
			}

			return nil
		},
	})
	if err != nil {
		return fmt.Errorf("updating invoice: %w", err)
	}

	if updatedInvoice.Lines.NonDeletedLineCount() == 0 {
		if updatedInvoice.Status == billing.InvoiceStatusGathering {
			// Gathering invoice deletion is handled by the service layer if they are empty
			return nil
		}

		// The invoice has no lines, so let's just delete it
		if err := h.billingService.DeleteInvoice(ctx, updatedInvoice.InvoiceID()); err != nil {
			return fmt.Errorf("deleting empty invoice: %w", err)
		}
	}

	return err
}

func (h *Handler) updateImmutableInvoice(ctx context.Context, invoice billing.Invoice, patches []linePatch) error {
	invoice, err := h.billingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: invoice.InvoiceID(),
		Expand:  billing.InvoiceExpandAll,
	})
	if err != nil {
		return fmt.Errorf("getting invoice: %w", err)
	}

	validationIssues := make([]billing.ValidationIssue, 0, len(patches))

	for _, patch := range patches {
		if patch.Op == patchOpDelete {
			validationIssues = append(validationIssues,
				newValidationIssueOnLine(invoice.Lines.GetByID(patch.LineID), "line should be voided, but the invoice is immutable"),
			)
			continue
		}

		existingLine := invoice.Lines.GetByID(patch.LineID)
		if existingLine == nil {
			return fmt.Errorf("line[%s] not found in the invoice, cannot update", patch.LineID)
		}

		targetState := patch.TargetState

		if targetState.Type != existingLine.Type {
			validationIssues = append(validationIssues,
				newValidationIssueOnLine(existingLine, "line type cannot be changed (new type: %s)", targetState.Type),
			)
			continue
		}

		switch targetState.Type {
		case billing.InvoiceLineTypeFee:
			if !targetState.FlatFee.PerUnitAmount.Equal(existingLine.FlatFee.PerUnitAmount) {
				validationIssues = append(validationIssues,
					newValidationIssueOnLine(existingLine, "flat fee amount cannot be changed on immutable invoice (new amount: %s)",
						targetState.FlatFee.PerUnitAmount.String(),
					),
				)
			}
		case billing.InvoiceLineTypeUsageBased:
			if !targetState.Period.Equal(existingLine.Period) {
				// The period of the line has changed => we need to refetch the quantity
				targetStateWithUpdatedQty, err := h.billingService.SnapshotLineQuantity(ctx, billing.SnapshotLineQuantityInput{
					Invoice: &invoice,
					Line:    targetState,
				})
				if err != nil {
					return fmt.Errorf("recalculating line[%s]: %w", targetStateWithUpdatedQty.ID, err)
				}

				if !targetStateWithUpdatedQty.UsageBased.Quantity.Equal(lo.FromPtrOr(existingLine.UsageBased.Quantity, alpacadecimal.Zero)) {
					validationIssues = append(validationIssues,
						newValidationIssueOnLine(existingLine, "usage based line's quantity cannot be changed on immutable invoice (new qty: %s)",
							targetStateWithUpdatedQty.UsageBased.Quantity.String()),
					)
				}
			}
		}
	}

	if len(validationIssues) > 0 {
		// These calculations are not idempontent, as we are only executing it against the in-scope part of the
		// subscription, so we cannot rely on the component based replace features of the validation issues member
		// of the invoice, so let's manually merge the issues.

		mergedValidationIssues, wasChange := h.mergeValidationIssues(invoice, validationIssues)
		if !wasChange {
			return nil
		}

		return h.billingService.UpsertValidationIssues(ctx, billing.UpsertValidationIssuesInput{
			Invoice: invoice.InvoiceID(),
			Issues:  mergedValidationIssues,
		})
	}

	return nil
}

func newValidationIssueOnLine(line *billing.Line, message string, a ...any) billing.ValidationIssue {
	return billing.ValidationIssue{
		// We use warning here, to prevent the state machine from being locked up due to present
		// validation errors
		Severity:  billing.ValidationIssueSeverityWarning,
		Message:   fmt.Sprintf(message, a...),
		Code:      billing.ImmutableInvoiceHandlingNotSupportedErrorCode,
		Component: SubscriptionSyncComponentName,
		Path:      fmt.Sprintf("lines/%s", line.ID),
	}
}

func (h *Handler) mergeValidationIssues(invoice billing.Invoice, issues []billing.ValidationIssue) (billing.ValidationIssues, bool) {
	changed := false

	// We don't expect much issues, and this is temporary until we have credits so let's just
	// use this simple approach.

	for _, issue := range issues {
		_, found := lo.Find(invoice.ValidationIssues, func(i billing.ValidationIssue) bool {
			return i.Path == issue.Path && i.Component == SubscriptionSyncComponentName && i.Code == billing.ImmutableInvoiceHandlingNotSupportedErrorCode &&
				i.Message == issue.Message
		})

		if found {
			continue
		}

		changed = true

		invoice.ValidationIssues = append(invoice.ValidationIssues, issue)
	}

	return invoice.ValidationIssues, changed
}

func (h *Handler) cloneLineForUpsert(line *billing.Line) *billing.Line {
	clone := line.CloneWithoutChildren()

	// We need to maintain the parent line relationship, so that we can update the qty snapshots on updated usage-based lines
	clone.ParentLine = line.ParentLine
	return clone
}
