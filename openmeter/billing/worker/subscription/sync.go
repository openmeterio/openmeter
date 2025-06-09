package billingworkersubscription

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
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
	UseUsageBasedFlatFeeLines       bool
}

type Config struct {
	BillingService      billing.Service
	SubscriptionService subscription.Service
	TxCreator           transaction.Creator
	FeatureFlags        FeatureFlags
	Logger              *slog.Logger
	Tracer              trace.Tracer
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
	featureFlags        FeatureFlags
	logger              *slog.Logger
	tracer              trace.Tracer
}

func New(config Config) (*Handler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &Handler{
		billingService:      config.BillingService,
		txCreator:           config.TxCreator,
		featureFlags:        config.FeatureFlags,
		subscriptionService: config.SubscriptionService,
		logger:              config.Logger,
		tracer:              config.Tracer,
	}, nil
}

func (h *Handler) invoicePendingLines(ctx context.Context, customer customer.CustomerID) error {
	ctx, span := tracex.StartWithNoValue(ctx, h.tracer, "billing.worker.subscription.sync.invoicePendingLines", trace.WithAttributes(
		attribute.String("customer_id", customer.ID),
	))

	return span.Wrap(ctx, func(ctx context.Context) error {
		_, err := h.billingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer:                   customer,
			ProgressiveBillingOverride: lo.ToPtr(false),
		})
		if err != nil {
			if errors.Is(err, billing.ErrInvoiceCreateNoLines) {
				return nil
			}

			return err
		}

		return nil
	})
}

func (h *Handler) SyncronizeSubscriptionAndInvoiceCustomer(ctx context.Context, subs subscription.SubscriptionView, asOf time.Time) error {
	ctx, span := tracex.StartWithNoValue(ctx, h.tracer, "billing.worker.subscription.sync.SynchronizeSubscriptionAndInvoiceCustomer", trace.WithAttributes(
		attribute.String("subscription_id", subs.Subscription.ID),
		attribute.String("as_of", asOf.Format(time.RFC3339)),
	))

	return span.Wrap(ctx, func(ctx context.Context) error {
		if err := h.SyncronizeSubscription(ctx, subs, asOf); err != nil {
			return fmt.Errorf("synchronize subscription: %w", err)
		}

		customerID := customer.CustomerID{
			Namespace: subs.Subscription.Namespace,
			ID:        subs.Subscription.CustomerId,
		}
		// Invoice any pending lines invoicable now, so that any in advance fees are invoiced immediately.
		if err := h.invoicePendingLines(ctx, customerID); err != nil {
			return fmt.Errorf("invoice pending lines (post): %w [customer_id=%s]", err, customerID.ID)
		}

		return nil
	})
}

func (h *Handler) SyncronizeSubscription(ctx context.Context, subs subscription.SubscriptionView, asOf time.Time) error {
	ctx, span := tracex.StartWithNoValue(ctx, h.tracer, "billing.worker.subscription.sync.SynchronizeSubscription", trace.WithAttributes(
		attribute.String("subscription_id", subs.Subscription.ID),
		attribute.String("as_of", asOf.Format(time.RFC3339)),
	))

	return span.Wrap(ctx, func(ctx context.Context) error {
		if !subs.Spec.HasBillables() {
			h.logger.InfoContext(ctx, "subscription has no billables, skipping sync", "subscription_id", subs.Subscription.ID)
			return nil
		}

		customerID := customer.CustomerID{
			Namespace: subs.Subscription.Namespace,
			ID:        subs.Subscription.CustomerId,
		}

		// TODO[later]: Right now we are getting the billing profile as a validation step, but later if we allow more collection
		// alignment settings, we should use the collection settings from here to determine the generation end (overriding asof).
		_, err := h.billingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Customer: customerID,
		})
		if err != nil {
			return fmt.Errorf("getting billing profile: %w", err)
		}

		currency, err := subs.Spec.Currency.Calculator()
		if err != nil {
			return fmt.Errorf("getting currency calculator: %w", err)
		}

		return h.billingService.WithLock(ctx, customer.CustomerID{
			Namespace: subs.Subscription.Namespace,
			ID:        subs.Subscription.CustomerId,
		}, func(ctx context.Context) error {
			// Calculate per line patches
			linesDiff, err := h.compareSubscriptionWithExistingLines(ctx, subs, asOf)
			if err != nil {
				return err
			}

			if linesDiff == nil {
				return nil
			}

			patches, err := h.getPatchesFromPlan(linesDiff, subs, currency)
			if err != nil {
				return err
			}

			// Apply the patches to the customer's invoices

			newLinePatches, err := h.getNewUpcomingLinePatches(ctx,
				subs,
				currency,
				linesDiff.NewSubscriptionItems,
			)
			if err != nil {
				return fmt.Errorf("provisioning pending lines: %w", err)
			}

			patches = append(patches, newLinePatches...)

			invoiceUpdater := NewInvoiceUpdater(h.billingService)
			if err := invoiceUpdater.ApplyPatches(ctx, customerID, patches); err != nil {
				return fmt.Errorf("updating invoices: %w", err)
			}

			return nil
		})
	})
}

type subscriptionSyncPlan struct {
	NewSubscriptionItems []subscriptionItemWithPeriod
	LinesToDelete        []billing.LineOrHierarchy
	LinesToUpsert        []subscriptionSyncPlanLineUpsert
}

type subscriptionSyncPlanLineUpsert struct {
	Target   subscriptionItemWithPeriod
	Existing billing.LineOrHierarchy
}

// calculateSyncPlan calculates the sync plan for the subscription, it returns the lines to delete, the lines to upsert and the new subscription items.
func (h *Handler) compareSubscriptionWithExistingLines(ctx context.Context, subs subscription.SubscriptionView, asOf time.Time) (*subscriptionSyncPlan, error) {
	ctx, span := tracex.Start[*subscriptionSyncPlan](ctx, h.tracer, "billing.worker.subscription.sync.compareSubscriptionWithExistingLines")

	return span.Wrap(ctx, func(ctx context.Context) (*subscriptionSyncPlan, error) {
		// Let's see what's in scope for the subscription
		// TODO: afaik this is already sorted, let's doublecheck that
		slices.SortFunc(subs.Phases, func(i, j subscription.SubscriptionPhaseView) int {
			return timeutil.Compare(i.SubscriptionPhase.ActiveFrom, j.SubscriptionPhase.ActiveFrom)
		})

		inScopeLines, err := h.collectUpcomingLines(ctx, subs, asOf)
		if err != nil {
			return nil, fmt.Errorf("collecting upcoming lines: %w", err)
		}

		// Let's load the existing lines for the subscription
		existingLines, err := h.billingService.GetLinesForSubscription(ctx, billing.GetLinesForSubscriptionInput{
			Namespace:      subs.Subscription.Namespace,
			SubscriptionID: subs.Subscription.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("getting existing lines: %w", err)
		}

		if len(inScopeLines) == 0 && len(existingLines) == 0 {
			// The subscription has no invoicable items, no present lines exist, so there's nothing to do
			return nil, nil
		}

		inScopeLinesByUniqueID, unique := slicesx.UniqueGroupBy(inScopeLines, func(i subscriptionItemWithPeriod) string {
			return i.UniqueID
		})
		if !unique {
			return nil, fmt.Errorf("duplicate unique ids in the upcoming lines")
		}

		existingLinesByUniqueID, unique := slicesx.UniqueGroupBy(
			lo.Filter(existingLines, func(l billing.LineOrHierarchy, _ int) bool {
				return l.ChildUniqueReferenceID() != nil
			}),
			func(l billing.LineOrHierarchy) string {
				return *l.ChildUniqueReferenceID()
			})
		if !unique {
			return nil, fmt.Errorf("duplicate unique ids in the existing lines")
		}

		existingLineUniqueIDs := lo.Keys(existingLinesByUniqueID)
		inScopeLineUniqueIDs := lo.Keys(inScopeLinesByUniqueID)
		// Let's execute the synchronization
		deletedLines, newLines := lo.Difference(existingLineUniqueIDs, inScopeLineUniqueIDs)
		lineIDsToUpsert := lo.Intersect(existingLineUniqueIDs, inScopeLineUniqueIDs)

		linesToDelete, err := slicesx.MapWithErr(deletedLines, func(id string) (billing.LineOrHierarchy, error) {
			line, ok := existingLinesByUniqueID[id]
			if !ok {
				return billing.LineOrHierarchy{}, fmt.Errorf("existing line[%s] not found in the existing lines", id)
			}

			return line, nil
		})
		if err != nil {
			return nil, fmt.Errorf("mapping deleted lines: %w", err)
		}

		linesToUpsert, err := slicesx.MapWithErr(lineIDsToUpsert, func(id string) (subscriptionSyncPlanLineUpsert, error) {
			existingLine, ok := existingLinesByUniqueID[id]
			if !ok {
				return subscriptionSyncPlanLineUpsert{}, fmt.Errorf("existing line[%s] not found in the existing lines", id)
			}

			return subscriptionSyncPlanLineUpsert{
				Target:   inScopeLinesByUniqueID[id],
				Existing: existingLine,
			}, nil
		})
		if err != nil {
			return nil, fmt.Errorf("mapping upsert lines: %w", err)
		}

		return &subscriptionSyncPlan{
			NewSubscriptionItems: lo.Map(newLines, func(id string, _ int) subscriptionItemWithPeriod {
				return inScopeLinesByUniqueID[id]
			}),
			LinesToDelete: linesToDelete,
			LinesToUpsert: linesToUpsert,
		}, nil
	})
}

func (h *Handler) getPatchesFromPlan(p *subscriptionSyncPlan, subs subscription.SubscriptionView, currency currencyx.Calculator) ([]linePatch, error) {
	patches := make([]linePatch, 0, len(p.LinesToDelete)+len(p.LinesToUpsert))

	// Let's update the existing lines
	for _, line := range p.LinesToDelete {
		deletePatches, err := h.getDeletePatchesForLine(line)
		if err != nil {
			return nil, fmt.Errorf("getting delete patches for line: %w", err)
		}

		patches = append(patches, deletePatches...)
	}

	for _, line := range p.LinesToUpsert {
		expectedLine, err := h.lineFromSubscritionRateCard(subs, line.Target, currency)
		if err != nil {
			return nil, fmt.Errorf("generating expected line[%s]: %w", line.Target.UniqueID, err)
		}

		// The line have 0 amount, so we are not going to bill it. This can happen if we are quickly changing subscriptions
		// immediately.
		if expectedLine == nil {
			deletePatches, err := h.getDeletePatchesForLine(line.Existing)
			if err != nil {
				return nil, fmt.Errorf("getting delete patches for line: %w", err)
			}

			patches = append(patches, deletePatches...)
			continue
		}

		updatePatches, err := h.getPatchesForExistingLineOrHierarchy(line.Existing, expectedLine)
		if err != nil {
			return nil, fmt.Errorf("updating line[%s]: %w", line.Target.UniqueID, err)
		}

		patches = append(patches, updatePatches...)
	}

	return patches, nil
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
func (h *Handler) collectUpcomingLines(ctx context.Context, subs subscription.SubscriptionView, asOf time.Time) ([]subscriptionItemWithPeriod, error) {
	ctx, span := tracex.Start[[]subscriptionItemWithPeriod](ctx, h.tracer, "billing.worker.subscription.sync.collectUpcomingLines")

	return span.Wrap(ctx, func(ctx context.Context) ([]subscriptionItemWithPeriod, error) {
		inScopeLines := make([]subscriptionItemWithPeriod, 0, 128)

		for _, phase := range subs.Phases {
			iterator, err := NewPhaseIterator(h.logger, h.tracer, subs, phase.SubscriptionPhase.Key)
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

			items, err := iterator.Generate(ctx, generationLimit)
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
	})
}

func (h *Handler) lineFromSubscritionRateCard(subs subscription.SubscriptionView, item subscriptionItemWithPeriod, currency currencyx.Calculator) (*billing.Line, error) {
	line := &billing.Line{
		LineBase: billing.LineBase{
			Namespace:              subs.Subscription.Namespace,
			Name:                   item.Spec.RateCard.AsMeta().Name,
			Description:            item.Spec.RateCard.AsMeta().Description,
			ManagedBy:              billing.SubscriptionManagedLine,
			Currency:               subs.Spec.Currency,
			Status:                 billing.InvoiceLineStatusValid,
			ChildUniqueReferenceID: &item.UniqueID,
			TaxConfig:              item.Spec.RateCard.AsMeta().TaxConfig,
			Period:                 item.Period,
			RateCardDiscounts:      h.discountsToBillingDiscounts(item.Spec.RateCard.AsMeta().Discounts),

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

	switch item.SubscriptionItem.RateCard.AsMeta().Price.Type() {
	case productcatalog.FlatPriceType:
		price, err := item.SubscriptionItem.RateCard.AsMeta().Price.AsFlat()
		if err != nil {
			return nil, fmt.Errorf("converting price to flat: %w", err)
		}

		// TODO[OM-1040]: We should support rounding errors in prorating calculations (such as 1/3 of a dollar is $0.33, 3*$0.33 is $0.99, if we bill
		// $1.00 in three equal pieces we should charge the customer $0.01 as the last split)
		perUnitAmount := currency.RoundToPrecision(price.Amount)
		if !item.Period.IsEmpty() && h.shouldProrateFlatFee(price) && !item.NonTruncatedPeriod.Equal(item.Period) {
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

		if !h.featureFlags.UseUsageBasedFlatFeeLines {
			line.Type = billing.InvoiceLineTypeFee
			line.FlatFee = &billing.FlatFeeLine{
				PerUnitAmount: perUnitAmount,
				Quantity:      alpacadecimal.NewFromInt(1),
				PaymentTerm:   price.PaymentTerm,
				Category:      billing.FlatFeeCategoryRegular,
			}
		} else {
			line.Type = billing.InvoiceLineTypeUsageBased
			line.UsageBased = &billing.UsageBasedLine{
				Price:      item.SubscriptionItem.RateCard.AsMeta().Price,
				FeatureKey: lo.FromPtr(item.SubscriptionItem.RateCard.AsMeta().FeatureKey),
			}
		}

	default:
		if item.SubscriptionItem.RateCard.AsMeta().Price == nil {
			return nil, fmt.Errorf("price must be defined for usage based price")
		}

		line.Type = billing.InvoiceLineTypeUsageBased
		line.InvoiceAt = inArrearsInvoiceAt
		line.UsageBased = &billing.UsageBasedLine{
			Price:      item.SubscriptionItem.RateCard.AsMeta().Price,
			FeatureKey: *item.SubscriptionItem.RateCard.AsMeta().FeatureKey,
		}
	}

	return line, nil
}

func (h *Handler) discountsToBillingDiscounts(discounts productcatalog.Discounts) billing.Discounts {
	out := billing.Discounts{}

	if discounts.Usage != nil {
		out.Usage = &billing.UsageDiscount{
			UsageDiscount: *discounts.Usage,
		}
	}

	if discounts.Percentage != nil {
		out.Percentage = &billing.PercentageDiscount{
			PercentageDiscount: *discounts.Percentage,
		}
	}

	return out
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

func (h *Handler) getNewUpcomingLinePatches(ctx context.Context, subs subscription.SubscriptionView, currency currencyx.Calculator, subsItems []subscriptionItemWithPeriod) ([]linePatch, error) {
	newLines, err := slicesx.MapWithErr(subsItems, func(subsItem subscriptionItemWithPeriod) (*billing.Line, error) {
		line, err := h.lineFromSubscritionRateCard(subs, subsItem, currency)
		if err != nil {
			return nil, fmt.Errorf("generating line from subscription item [%s]: %w", subsItem.SubscriptionItem.ID, err)
		}

		return line, nil
	})
	if err != nil {
		return nil, fmt.Errorf("creating new lines: %w", err)
	}

	lines := lo.Filter(newLines, func(l *billing.Line, _ int) bool {
		return l != nil
	})

	return lo.Map(lines, func(l *billing.Line, _ int) linePatch {
		return newCreateLinePatch(*l)
	}), nil
}

func (h *Handler) getPatchesForExistingLineOrHierarchy(existingLine billing.LineOrHierarchy, expectedLine *billing.Line) ([]linePatch, error) {
	// TODO/WARNING[later]: This logic should be fine with everything that can be billed progressively, however the following use-cases
	// will behave strangely:
	//
	// - An in advance flat fee cannot be prorated, as that would require void/credit logic.
	// - If a volume based item's tiered are changed, then the old volume based item will be billed at that rate, and the
	//   new volume based item's calculation will start from the new tiered rates beginning. (e.g. if we have a 1 month long tiered
	//   price, then we change the tiered price in the middle of the month, the old tiered price will be billed for the first half
	//   and the new tiered price will be billed for the second half, meaning that the customer will be billed for less sum traffic) [OM]
	// - If a meter is unique counted, then the unique count will be reset at the change's time

	switch existingLine.Type() {
	case billing.LineOrHierarchyTypeLine:
		line, err := existingLine.AsLine()
		if err != nil {
			return nil, fmt.Errorf("getting line: %w", err)
		}

		return h.getPatchesForExistingLine(line, expectedLine)
	case billing.LineOrHierarchyTypeHierarchy:
		group, err := existingLine.AsHierarchy()
		if err != nil {
			return nil, fmt.Errorf("getting hierarchy: %w", err)
		}

		return h.getPatchesForExistingHierarchy(group, expectedLine)
	default:
		return nil, fmt.Errorf("unsupported line or hierarchy type: %s", existingLine.Type())
	}
}

func (h *Handler) getPatchesForExistingLine(existingLine *billing.Line, expectedLine *billing.Line) ([]linePatch, error) {
	// Manual edits prevent resyncronization so that we preserve the user intent
	if existingLine.ManagedBy != billing.SubscriptionManagedLine {
		return nil, nil
	}

	// We assume that only the period can change, maybe some pricing data due to prorating (for flat lines)

	targetLine := existingLine.CloneWithoutChildren()

	wasChange := false

	setIfDoesNotEqual(&targetLine.Period, expectedLine.Period, &wasChange)
	setIfDoesNotEqual(&targetLine.InvoiceAt, expectedLine.InvoiceAt, &wasChange)

	if !isFlatFee(targetLine) {
		// UBP Empty lines are not allowed, let's delete them instead
		if targetLine.Period.Truncate(billing.DefaultMeterResolution).IsEmpty() {
			return []linePatch{
				newDeleteLinePatch(existingLine.LineID(), existingLine.InvoiceID),
			}, nil
		}
	}

	if targetLine.DeletedAt != nil {
		targetLine.DeletedAt = nil
		wasChange = true
	}

	// Let's handle the flat fee prorating (e.g. syncronizing the amount maybe in retrospect)
	if isFlatFee(targetLine) {
		if !isFlatFee(expectedLine) {
			return nil, errors.New("cannot merge flat fee line with usage based line")
		}

		perUnitAmountExisting, err := getFlatFeePerUnitAmount(existingLine)
		if err != nil {
			return nil, fmt.Errorf("getting flat fee per unit amount: %w", err)
		}

		perUnitAmountExpected, err := getFlatFeePerUnitAmount(expectedLine)
		if err != nil {
			return nil, fmt.Errorf("getting flat fee per unit amount: %w", err)
		}

		if !perUnitAmountExisting.Equal(perUnitAmountExpected) {
			if err := setFlatFeePerUnitAmount(targetLine, perUnitAmountExpected); err != nil {
				return nil, fmt.Errorf("setting flat fee per unit amount: %w", err)
			}
			wasChange = true
		}
	}

	if !wasChange {
		return nil, nil
	}

	return []linePatch{
		newUpdateLinePatch(targetLine),
	}, nil
}

func (h *Handler) getPatchesForExistingHierarchy(existingHierarchy *billing.SplitLineHierarchy, expectedLine *billing.Line) ([]linePatch, error) {
	// Parts of the line has been already invoiced using progressive invoicing, so we need to examine the children

	// Nothing to do here, as split lines are UBP lines and thus we don't need the flat fee corrections
	// TODO[later]: When we implement progressive billing based pro-rating, we need to support adjusting flat fee
	// segments here.

	if existingHierarchy.Group.ServicePeriod.End.Equal(expectedLine.Period.End) {
		// The line is already in the expected state, so we can safely return here
		return nil, nil
	}

	patches := []linePatch{}

	// Case #1: The line is being expanded (e.g. continue subscription)
	if existingHierarchy.Group.ServicePeriod.End.Before(expectedLine.Period.End) {
		// Expansion of the line (e.g. continue subscription)

		children := existingHierarchy.Lines
		if len(children) > 0 {
			slices.SortFunc(children, func(i, j billing.LineWithInvoiceHeader) int {
				return timeutil.Compare(i.Line.Period.End, j.Line.Period.End)
			})

			lastChild := children[len(children)-1].Line.CloneWithoutChildren()

			if lastChild.ManagedBy == billing.SubscriptionManagedLine {
				// We are not supporting period changes for children, and we need to maintain the consistency so
				// even for overridden lines we need to update the period

				// We however allow deletions, so we are only un-deleting the line here if it was deleted by the sync engine
				lastChild.DeletedAt = nil
			}

			lastChild.Period.End = expectedLine.Period.End
			lastChild.InvoiceAt = expectedLine.Period.End
			patches = append(patches, newUpdateLinePatch(lastChild))
		}

		// We have already updated the last child, so we need to update at least the periods regardless of managed_by to keep the consistency
		updatedGroup := existingHierarchy.Group.ToUpdate()
		updatedGroup.ServicePeriod.End = expectedLine.Period.End
		patches = append(patches, newUpdateSplitLineGroupPatch(updatedGroup))

		return patches, nil
	}

	// Shrink of the line (e.g. subscription cancled, subscription item edit)

	for _, child := range existingHierarchy.Lines {
		if child.Line.Period.End.Before(expectedLine.Period.End) {
			// The child is not affected by the period shrink, so we can skip it
			continue
		}

		if child.Line.Period.Start.After(expectedLine.Period.End) {
			// The child is after the period shrink, so we need to delete it as it became invalid
			patches = append(patches, newDeleteLinePatch(child.Line.LineID(), child.Line.InvoiceID))
			continue
		}

		// The child is partially affected by the period shrink, so we need to adjust the period
		if !child.Line.Period.End.Equal(expectedLine.Period.End) {
			updatedChild := child.Line.CloneWithoutChildren()
			updatedChild.Period.End = expectedLine.Period.End

			if updatedChild.InvoiceAt.After(expectedLine.Period.End) {
				// The child is invoiced after the period end, so we need to adjust the invoice date
				updatedChild.InvoiceAt = expectedLine.Period.End
			}

			if child.Line.ManagedBy == billing.SubscriptionManagedLine {
				updatedChild.DeletedAt = nil
			}

			if !isFlatFee(updatedChild) {
				// UBP Empty lines are not allowed, let's delete them instead
				if updatedChild.Period.Truncate(billing.DefaultMeterResolution).IsEmpty() {
					patches = append(patches, newDeleteLinePatch(child.Line.LineID(), child.Line.InvoiceID))
					continue
				}
			}

			patches = append(patches, newUpdateLinePatch(updatedChild))
		}
	}

	// Let's make sure that the group's end is updated to the expected period end, so that
	// we can reliably detect last child

	updatedGroup := existingHierarchy.Group.ToUpdate()
	updatedGroup.ServicePeriod.End = expectedLine.Period.End
	patches = append(patches, newUpdateSplitLineGroupPatch(updatedGroup))

	return patches, nil
}
