package targetstate

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type State struct {
	Items                  []SubscriptionItemWithPeriods
	MaxGenerationTimeLimit time.Time
}

type Builder struct {
	logger *slog.Logger
	tracer trace.Tracer
}

func NewBuilder(logger *slog.Logger, tracer trace.Tracer) Builder {
	return Builder{logger: logger, tracer: tracer}
}

func (b Builder) Build(ctx context.Context, subs subscription.SubscriptionView, asOf time.Time, persisted persistedstate.State) (State, error) {
	span := tracex.Start[State](ctx, b.tracer, "billing.worker.subscription.sync.targetstate.Build")

	return span.Wrap(func(ctx context.Context) (State, error) {
		slices.SortFunc(subs.Phases, func(i, j subscription.SubscriptionPhaseView) int {
			return timeutil.Compare(i.SubscriptionPhase.ActiveFrom, j.SubscriptionPhase.ActiveFrom)
		})

		upcomingLinesResult, err := b.collectUpcomingLines(ctx, subs, asOf)
		if err != nil {
			return State{}, fmt.Errorf("collecting upcoming lines: %w", err)
		}

		inScopeLines, err := b.correctPeriodStartForUpcomingLines(ctx, subs.Subscription.ID, upcomingLinesResult.Lines, persisted)
		if err != nil {
			return State{}, fmt.Errorf("correcting period start for upcoming lines: %w", err)
		}

		return State{
			Items:                  inScopeLines,
			MaxGenerationTimeLimit: upcomingLinesResult.SubscriptionMaxGenerationTimeLimit,
		}, nil
	})
}

type collectResult struct {
	Lines                              []SubscriptionItemWithPeriods
	SubscriptionMaxGenerationTimeLimit time.Time
}

func (b Builder) collectUpcomingLines(ctx context.Context, subs subscription.SubscriptionView, asOf time.Time) (collectResult, error) {
	span := tracex.Start[collectResult](ctx, b.tracer, "billing.worker.subscription.sync.collectUpcomingLines")

	return span.Wrap(func(ctx context.Context) (collectResult, error) {
		inScopeLines := make([]SubscriptionItemWithPeriods, 0, 128)
		maxGenerationTimeLimit := time.Time{}

		for _, phase := range subs.Phases {
			iterator, err := NewPhaseIterator(b.logger, b.tracer, subs, phase.SubscriptionPhase.Key)
			if err != nil {
				return collectResult{}, fmt.Errorf("creating phase iterator: %w", err)
			}

			if !iterator.HasInvoicableItems() {
				continue
			}

			generationLimit := asOf

			currBillingPeriod, err := subs.Spec.GetAlignedBillingPeriodAt(asOf)
			if err != nil {
				switch {
				case subscription.IsValidationIssueWithCode(err, subscription.ErrCodeSubscriptionBillingPeriodQueriedBeforeSubscriptionStart):
					b.logger.InfoContext(ctx, "asOf is before subscription start, advancing generation time to subscription start", "subscription_id", subs.Subscription.ID, "as_of", asOf, "subscription_start", subs.Spec.ActiveFrom)
					generationLimit = subs.Subscription.ActiveFrom
				default:
					return collectResult{}, fmt.Errorf("getting aligned billing period: %w", err)
				}
			}

			if !currBillingPeriod.From.IsZero() && !generationLimit.Equal(currBillingPeriod.From) {
				generationLimit = currBillingPeriod.To
			}

			if phaseStart := iterator.PhaseStart(); phaseStart.After(generationLimit) {
				generationLimit = iterator.GetMinimumBillableTime()
				if generationLimit.IsZero() {
					continue
				}
			}

			items, err := iterator.Generate(ctx, generationLimit)
			if err != nil {
				return collectResult{}, fmt.Errorf("generating items: %w", err)
			}

			if maxGenerationTimeLimit.Before(generationLimit) {
				maxGenerationTimeLimit = generationLimit
			}

			inScopeLines = append(inScopeLines, items...)

			if phaseEnd := iterator.PhaseEnd(); phaseEnd != nil && !phaseEnd.Before(asOf) {
				break
			}
		}

		return collectResult{
			Lines:                              inScopeLines,
			SubscriptionMaxGenerationTimeLimit: maxGenerationTimeLimit,
		}, nil
	})
}

func (b Builder) correctPeriodStartForUpcomingLines(ctx context.Context, subscriptionID string, inScopeLines []SubscriptionItemWithPeriods, persisted persistedstate.State) ([]SubscriptionItemWithPeriods, error) {
	for idx, line := range inScopeLines {
		if line.PeriodIndex == 0 {
			continue
		}

		if existingCurrentLine, ok := persisted.ByUniqueID[line.UniqueID]; ok {
			syncIgnore, err := b.lineOrHierarchyHasAnnotation(existingCurrentLine, billing.AnnotationSubscriptionSyncIgnore)
			if err != nil {
				return nil, fmt.Errorf("checking if line has subscription sync ignore annotation: %w", err)
			}

			if syncIgnore {
				continue
			}
		}

		previousPeriodUniqueID := strings.Join([]string{
			subscriptionID,
			line.PhaseKey,
			line.Spec.ItemKey,
			fmt.Sprintf("v[%d]", line.ItemVersion),
			fmt.Sprintf("period[%d]", line.PeriodIndex-1),
		}, "/")

		existingPreviousLine, ok := persisted.ByUniqueID[previousPeriodUniqueID]
		if !ok {
			continue
		}

		existingPreviousLineSyncIgnoreAnnotation, err := b.lineOrHierarchyHasAnnotation(existingPreviousLine, billing.AnnotationSubscriptionSyncIgnore)
		if err != nil {
			return nil, fmt.Errorf("checking if previous line has subscription sync ignore annotation: %w", err)
		}

		if !existingPreviousLineSyncIgnoreAnnotation {
			continue
		}

		existingPreviousLineSyncForceContinuousLinesAnnotation, err := b.lineOrHierarchyHasAnnotation(existingPreviousLine, billing.AnnotationSubscriptionSyncForceContinuousLines)
		if err != nil {
			return nil, fmt.Errorf("checking if previous line has subscription sync force continuous lines annotation: %w", err)
		}

		if !existingPreviousLineSyncForceContinuousLinesAnnotation {
			continue
		}

		previousServicePeriod := existingPreviousLine.ServicePeriod()
		continuousStart := previousServicePeriod.To.Truncate(streaming.MinimumWindowSizeDuration)
		if line.ServicePeriod.Start.Equal(continuousStart) {
			continue
		}

		if !line.ServicePeriod.Start.Equal(line.FullServicePeriod.Start) {
			return nil, fmt.Errorf("line[%s] service period and full service period start does not match", line.UniqueID)
		}

		inScopeLines[idx].ServicePeriod.Start = continuousStart
		inScopeLines[idx].FullServicePeriod.Start = continuousStart

		if line.FullServicePeriod.Start.Equal(line.BillingPeriod.Start) {
			inScopeLines[idx].BillingPeriod.Start = continuousStart
		}
	}

	return inScopeLines, nil
}

func (b Builder) lineOrHierarchyHasAnnotation(lineOrHierarchy billing.LineOrHierarchy, annotation string) (bool, error) {
	switch lineOrHierarchy.Type() {
	case billing.LineOrHierarchyTypeLine:
		previousLine, err := lineOrHierarchy.AsGenericLine()
		if err != nil {
			return false, fmt.Errorf("getting previous line: %w", err)
		}

		return b.lineHasAnnotation(previousLine.GetManagedBy(), previousLine.GetAnnotations(), annotation), nil
	case billing.LineOrHierarchyTypeHierarchy:
		hierarchy, err := lineOrHierarchy.AsHierarchy()
		if err != nil {
			return false, fmt.Errorf("getting previous hierarchy: %w", err)
		}

		return b.hierarchyHasAnnotation(hierarchy, annotation)
	default:
		return false, nil
	}
}

func (b Builder) lineHasAnnotation(managedBy billing.InvoiceLineManagedBy, annotations models.Annotations, annotation string) bool {
	if managedBy != billing.SubscriptionManagedLine {
		return false
	}

	return annotations.GetBool(annotation)
}

func (b Builder) hierarchyHasAnnotation(hierarchy *billing.SplitLineHierarchy, annotation string) (bool, error) {
	servicePeriod := hierarchy.Group.ServicePeriod
	for _, child := range hierarchy.Lines {
		if child.Line.GetServicePeriod().To.Equal(servicePeriod.End) && child.Line.GetDeletedAt() == nil {
			return b.lineHasAnnotation(child.Line.GetManagedBy(), child.Line.GetAnnotations(), annotation), nil
		}
	}

	return false, nil
}

// TODO: make a member of the SubscriptionItemWithPeriods type (for now it's kept here for easier review)
func lineFromSubscriptionRateCard(subs subscription.SubscriptionView, item SubscriptionItemWithPeriods, currency currencyx.Calculator) (*billing.GatheringLine, error) {
	line := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace:   subs.Subscription.Namespace,
				Name:        item.Spec.RateCard.AsMeta().Name,
				Description: item.Spec.RateCard.AsMeta().Description,
			}),
			ManagedBy:              billing.SubscriptionManagedLine,
			Currency:               subs.Spec.Currency,
			ChildUniqueReferenceID: &item.UniqueID,
			TaxConfig:              item.Spec.RateCard.AsMeta().TaxConfig,
			ServicePeriod:          item.ServicePeriod.ToClosedPeriod(),
			InvoiceAt:              item.GetInvoiceAt(),
			RateCardDiscounts:      discountsToBillingDiscounts(item.Spec.RateCard.AsMeta().Discounts),
			Subscription: &billing.SubscriptionReference{
				SubscriptionID: subs.Subscription.ID,
				PhaseID:        item.PhaseID,
				ItemID:         item.SubscriptionItem.ID,
				BillingPeriod: timeutil.ClosedPeriod{
					From: item.BillingPeriod.Start,
					To:   item.BillingPeriod.End,
				},
			},
		},
	}

	if price := item.SubscriptionItem.RateCard.AsMeta().Price; price != nil && price.GetPaymentTerm() == productcatalog.InArrearsPaymentTerm {
		if item.FullServicePeriod.Duration() == time.Duration(0) {
			return nil, nil
		}
	}

	switch item.SubscriptionItem.RateCard.AsMeta().Price.Type() {
	case productcatalog.FlatPriceType:
		price, err := item.SubscriptionItem.RateCard.AsMeta().Price.AsFlat()
		if err != nil {
			return nil, fmt.Errorf("converting price to flat: %w", err)
		}

		perUnitAmount := currency.RoundToPrecision(price.Amount)
		if !item.ServicePeriod.IsEmpty() && shouldProrate(item, subs) {
			perUnitAmount = currency.RoundToPrecision(price.Amount.Mul(item.PeriodPercentage()))
		}

		if perUnitAmount.IsZero() {
			return nil, nil
		}

		line.Price = lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      perUnitAmount,
			PaymentTerm: price.PaymentTerm,
		}))
		line.FeatureKey = lo.FromPtr(item.SubscriptionItem.RateCard.AsMeta().FeatureKey)
	default:
		if item.SubscriptionItem.RateCard.AsMeta().Price == nil {
			return nil, fmt.Errorf("price must be defined for usage based price")
		}

		line.Price = lo.FromPtr(item.SubscriptionItem.RateCard.AsMeta().Price)
		line.FeatureKey = lo.FromPtr(item.SubscriptionItem.RateCard.AsMeta().FeatureKey)
	}

	return &line, nil
}

func discountsToBillingDiscounts(discounts productcatalog.Discounts) billing.Discounts {
	out := billing.Discounts{}

	if discounts.Usage != nil {
		out.Usage = &billing.UsageDiscount{UsageDiscount: *discounts.Usage}
	}

	if discounts.Percentage != nil {
		out.Percentage = &billing.PercentageDiscount{PercentageDiscount: *discounts.Percentage}
	}

	return out
}

func shouldProrate(item SubscriptionItemWithPeriods, subView subscription.SubscriptionView) bool {
	if !subView.Subscription.ProRatingConfig.Enabled {
		return false
	}

	if item.Spec.RateCard.AsMeta().Price.Type() != productcatalog.FlatPriceType {
		return false
	}

	if subView.Subscription.ActiveTo != nil && !subView.Subscription.ActiveTo.After(item.ServicePeriod.End) {
		return false
	}

	switch subView.Subscription.ProRatingConfig.Mode {
	case productcatalog.ProRatingModeProratePrices:
		return true
	default:
		return false
	}
}
