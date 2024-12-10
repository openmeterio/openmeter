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
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timex"
)

type Config struct {
	BillingService billing.Service
	TxCreator      transaction.Creator
	Logger         *slog.Logger
}

func (c Config) Validate() error {
	if c.BillingService == nil {
		return fmt.Errorf("billing service is required")
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
	billingService billing.Service
	txCreator      transaction.Creator
	logger         *slog.Logger
}

func New(config Config) (*Handler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &Handler{
		billingService: config.BillingService,
		txCreator:      config.TxCreator,
		logger:         config.Logger,
	}, nil
}

func (h *Handler) SyncronizeSubscription(ctx context.Context, subs subscription.SubscriptionView, asOf time.Time) error {
	// TODO[later]: Right now we are getting the billing profile as a validation step, but later if we allow more collection
	// alignment settings, we should use the collection settings from here to determine the generation end (overriding asof).
	_, err := h.billingService.GetProfileWithCustomerOverride(
		ctx,
		billing.GetProfileWithCustomerOverrideInput{
			Namespace:  subs.Subscription.Namespace,
			CustomerID: subs.Subscription.CustomerId,
		},
	)
	if err != nil {
		return fmt.Errorf("getting billing profile: %w", err)
	}

	// Let's see what's in scope for the subscription
	slices.SortFunc(subs.Phases, func(i, j subscription.SubscriptionPhaseView) int {
		return timex.Compare(i.SubscriptionPhase.ActiveFrom, j.SubscriptionPhase.ActiveFrom)
	})

	inScopeLines, err := h.collectUpcomingLines(subs, asOf)
	if err != nil {
		return fmt.Errorf("collecting upcoming lines: %w", err)
	}

	if len(inScopeLines) == 0 {
		// The subscription has no invoicable items, so we can return early
		return nil
	}

	inScopeLinesByUniqueID, unique := slicesx.UniqueGroupBy(inScopeLines, func(i subscriptionItemWithPeriod) string {
		return i.UniqueID
	})
	if !unique {
		return fmt.Errorf("duplicate unique ids in the upcoming lines")
	}

	// Let's load the existing lines for the subscription
	existingLines, err := h.billingService.GetLinesForSubscription(ctx, billing.GetLinesForSubscriptionInput{
		Namespace:      subs.Subscription.Namespace,
		SubscriptionID: subs.Subscription.ID,
	})
	if err != nil {
		return fmt.Errorf("getting existing lines: %w", err)
	}

	existingLinesByUniqueID, unique := slicesx.UniqueGroupBy(
		lo.Filter(existingLines, func(l *billing.Line, _ int) bool {
			return l.ChildUniqueReferenceID != nil
		}),
		func(l *billing.Line) string {
			return *l.ChildUniqueReferenceID
		})
	if !unique {
		return fmt.Errorf("duplicate unique ids in the existing lines")
	}

	existingLineUniqueIDs := lo.Keys(existingLinesByUniqueID)
	inScopeLineUniqueIDs := lo.Keys(inScopeLinesByUniqueID)
	// Let's execute the synchronization
	deletedLines, newLines := lo.Difference(existingLineUniqueIDs, inScopeLineUniqueIDs)
	linesToUpsert := lo.Intersect(existingLineUniqueIDs, inScopeLineUniqueIDs)

	currency, err := currencyx.Code(subs.Spec.Currency).Calculator()
	if err != nil {
		return fmt.Errorf("getting currency calculator: %w", err)
	}

	return transaction.RunWithNoValue(ctx, h.txCreator, func(ctx context.Context) error {
		// Let's stage new lines
		newLines, err := slicesx.MapWithErr(newLines, func(id string) (billing.LineWithCustomer, error) {
			line, err := h.lineFromSubscritionRateCard(subs, inScopeLinesByUniqueID[id], currency)
			if err != nil {
				return billing.LineWithCustomer{}, fmt.Errorf("generating line[%s]: %w", id, err)
			}

			return billing.LineWithCustomer{
				Line:       *line,
				CustomerID: subs.Subscription.CustomerId,
			}, nil
		})
		if err != nil {
			return fmt.Errorf("creating new lines: %w", err)
		}

		_, err = h.billingService.CreatePendingInvoiceLines(ctx, billing.CreateInvoiceLinesInput{
			Namespace: subs.Subscription.Namespace,
			Lines:     newLines,
		})
		if err != nil {
			return fmt.Errorf("creating pending invoice lines: %w", err)
		}

		// Let's flag deleted lines deleted
		nowPtr := lo.ToPtr(clock.Now())
		for _, uniqueID := range deletedLines {
			existingLinesByUniqueID[uniqueID].DeletedAt = nowPtr
		}

		// Let's update the existing lines
		for _, uniqueID := range linesToUpsert {
			expectedLine, err := h.lineFromSubscritionRateCard(subs, inScopeLinesByUniqueID[uniqueID], currency)
			if err != nil {
				return fmt.Errorf("generating expected line[%s]: %w", uniqueID, err)
			}

			if err := h.updateInScopeLine(existingLinesByUniqueID[uniqueID], expectedLine); err != nil {
				return fmt.Errorf("updating line[%s]: %w", uniqueID, err)
			}
		}

		return h.billingService.UpdateInvoiceLinesInternal(ctx, billing.UpdateInvoiceLinesInternalInput{
			Namespace:  subs.Subscription.Namespace,
			CustomerID: subs.Subscription.CustomerId,
			Lines:      existingLines,
		})
	})
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

func (h *Handler) lineFromSubscritionRateCard(subs subscription.SubscriptionView, item subscriptionItemWithPeriod, currency currencyx.Calculator) (*billing.Line, error) {
	line := &billing.Line{
		LineBase: billing.LineBase{
			Namespace:              subs.Subscription.Namespace,
			Name:                   item.Spec.RateCard.Name,
			Description:            item.Spec.RateCard.Description,
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

	switch item.SubscriptionItem.RateCard.Price.Type() {
	case productcatalog.FlatPriceType:
		price, err := item.SubscriptionItem.RateCard.Price.AsFlat()
		if err != nil {
			return nil, fmt.Errorf("converting price to flat: %w", err)
		}

		perUnitAmount := price.Amount
		switch price.PaymentTerm {
		case productcatalog.InArrearsPaymentTerm:
			line.InvoiceAt = item.Period.End
			// TODO[OM-1040]: We should support rounding errors in prorating calculations (such as 1/3 of a dollar is $0.33, 3*$0.33 is $0.99, if we bill
			// $1.00 in three equal pieces we should charge the customer $0.01 as the last split)
			perUnitAmount = currency.RoundToPrecision(price.Amount.Mul(item.PeriodPercentage()))
		case productcatalog.InAdvancePaymentTerm:
			// In case of inAdvance we should always invoice at the start of the period and if there's a change
			// prorating should void the item and credit the customer.
			//
			// Warning: We are not supporting voiding or crediting right now, so we are going to overcharge on
			// inAdvance items in case of a change.
			line.InvoiceAt = item.Period.Start
		default:
			return nil, fmt.Errorf("unsupported payment term: %v", price.PaymentTerm)
		}

		line.Type = billing.InvoiceLineTypeFee
		line.FlatFee = billing.FlatFeeLine{
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
		line.InvoiceAt = item.Period.End
		line.UsageBased = billing.UsageBasedLine{
			Price:      *item.SubscriptionItem.RateCard.Price,
			FeatureKey: *item.SubscriptionItem.RateCard.FeatureKey,
		}

	default:
		return nil, fmt.Errorf("unsupported price type: %v", item.SubscriptionItem.RateCard.Price.Type())
	}

	return line, nil
}

func (h *Handler) updateInScopeLine(existingLine *billing.Line, expectedLine *billing.Line) error {
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
		h.mergeChangesFromLine(existingLine, expectedLine)

		return nil
	}

	// Parts of the line has been already invoiced using progressive invoicing, so we need to examine the children
	if existingLine.Status == billing.InvoiceLineStatusSplit {
		// Nothing to do here, as split lines are UBP lines and thus we don't need the flat fee corrections
		// TODO[later]: When we implement progressive billing based pro-rating, we need to support adjusting flat fee
		// segments here.

		if existingLine.Period.End.Before(expectedLine.Period.End) {
			// Expansion of the line (e.g. continue subscription)

			children := existingLine.Children.OrEmpty()
			if len(children) > 0 {
				slices.SortFunc(children, func(i, j *billing.Line) int {
					return timex.Compare(i.Period.End, j.Period.End)
				})

				lastChild := children[len(children)-1]
				lastChild.Period.End = expectedLine.Period.End
				lastChild.InvoiceAt = expectedLine.Period.End
			}

			existingLine.Period.End = expectedLine.Period.End
			existingLine.InvoiceAt = expectedLine.Period.End
		} else {
			// Shrink of the line (e.g. subscription cancled, subscription item edit)

			for _, child := range existingLine.Children.OrEmpty() {
				if child.Period.End.Before(expectedLine.Period.End) {
					// The child is not affected by the period shrink, so we can skip it
					continue
				}

				if child.Period.Start.After(expectedLine.Period.End) {
					// The child is after the period shrink, so we need to delete it as it became invalid
					child.DeletedAt = lo.ToPtr(clock.Now())
					continue
				}

				// The child is partially affected by the period shrink, so we need to adjust the period
				if !child.Period.End.Equal(expectedLine.Period.End) {
					child.Period.End = expectedLine.Period.End

					if child.InvoiceAt.After(expectedLine.Period.End) {
						// The child is invoiced after the period end, so we need to adjust the invoice date
						child.InvoiceAt = expectedLine.Period.End
					}
				}
			}
			// Split lines are always associated with gathering invoices, so we can safely update the line without checking for
			// snapshot update requirements

			existingLine.Period.End = expectedLine.Period.End
			existingLine.InvoiceAt = expectedLine.Period.End
		}

		return nil
	}

	// There is no other state in which a line can be in, so we can safely return an error here
	return fmt.Errorf("could not handle line update [lineID=%s, status=%s]", existingLine.ID, existingLine.Status)
}

func (h *Handler) mergeChangesFromLine(existingLine *billing.Line, expectedLine *billing.Line) {
	// We assume that only the period can change, maybe some pricing data due to prorating (for flat lines)

	existingLine.Period = expectedLine.Period

	existingLine.InvoiceAt = expectedLine.InvoiceAt

	// Let's handle the flat fee prorating
	if existingLine.Type == billing.InvoiceLineTypeFee {
		existingLine.FlatFee.PerUnitAmount = expectedLine.FlatFee.PerUnitAmount
	}
}
