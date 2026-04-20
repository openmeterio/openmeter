package targetstate

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type State struct {
	Items                  []StateItem
	MaxGenerationTimeLimit time.Time
}

type BuildInput struct {
	AsOf              time.Time
	CustomerDeletedAt *time.Time
	SubscriptionView  subscription.SubscriptionView
	Persisted         persistedstate.State
}

func (i BuildInput) Validate() error {
	var errs []error

	if i.AsOf.IsZero() {
		errs = append(errs, errors.New("asOf is required"))
	}

	if err := i.SubscriptionView.Validate(true); err != nil {
		errs = append(errs, err)
	}

	if err := i.Persisted.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

type Builder struct {
	logger *slog.Logger
	tracer trace.Tracer
}

func NewBuilder(logger *slog.Logger, tracer trace.Tracer) Builder {
	return Builder{logger: logger, tracer: tracer}
}

func (b Builder) Build(ctx context.Context, input BuildInput) (State, error) {
	span := tracex.Start[State](ctx, b.tracer, "billing.worker.subscription.sync.targetstate.Build")

	return span.Wrap(func(ctx context.Context) (State, error) {
		if err := input.Validate(); err != nil {
			return State{}, fmt.Errorf("validating input: %w", err)
		}

		subs := input.SubscriptionView
		// If the customer is deleted, we need to cap the subscription view at the customer deleted at
		// or invoicing will not allow creating the lines.
		// Note: this only happens if there is a db inconsistency between the subscription and the customer lifecycle.
		if input.CustomerDeletedAt != nil && (input.SubscriptionView.Subscription.ActiveTo == nil || input.SubscriptionView.Subscription.ActiveTo.After(*input.CustomerDeletedAt)) {
			subsActiveTo := "(nil)"
			if subs.Subscription.ActiveTo != nil {
				subsActiveTo = subs.Subscription.ActiveTo.Format(time.RFC3339)
			}

			b.logger.WarnContext(ctx, "customer deleted at is before subscription active to, capping subscription view at customer deleted at",
				"subscription_id", subs.Subscription.ID,
				"customer_deleted_at", *input.CustomerDeletedAt,
				"subscription_active_to", subsActiveTo,
			)

			subs = withActiveTo(subs, *input.CustomerDeletedAt)
		}

		slices.SortFunc(subs.Phases, func(i, j subscription.SubscriptionPhaseView) int {
			return timeutil.Compare(i.SubscriptionPhase.ActiveFrom, j.SubscriptionPhase.ActiveFrom)
		})

		upcomingLinesResult, err := b.collectUpcomingLines(ctx, subs, input.AsOf)
		if err != nil {
			return State{}, fmt.Errorf("collecting upcoming lines: %w", err)
		}

		inScopeLines, err := b.correctPeriodStartForUpcomingLines(ctx, subs.Subscription.ID, upcomingLinesResult.Lines, input.Persisted)
		if err != nil {
			return State{}, fmt.Errorf("correcting period start for upcoming lines: %w", err)
		}

		currencyCalculator, err := subs.Subscription.Currency.Calculator()
		if err != nil {
			return State{}, fmt.Errorf("getting currency calculator: %w", err)
		}

		return State{
			Items: lo.Map(inScopeLines, func(item SubscriptionItemWithPeriods, _ int) StateItem {
				return StateItem{
					SubscriptionItemWithPeriods: item,
					CurrencyCalculator:          currencyCalculator,
					Subscription:                subs.Subscription,
				}
			}),
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

func withActiveTo(subs subscription.SubscriptionView, endAt time.Time) subscription.SubscriptionView {
	subs.Subscription.ActiveTo = &endAt
	subs.Spec.ActiveTo = &endAt
	return subs
}

func (b Builder) correctPeriodStartForUpcomingLines(ctx context.Context, subscriptionID string, inScopeLines []SubscriptionItemWithPeriods, persisted persistedstate.State) ([]SubscriptionItemWithPeriods, error) {
	for idx, line := range inScopeLines {
		if line.PeriodIndex == 0 {
			continue
		}

		if existingCurrentLine, ok := persisted.ByUniqueID[line.UniqueID]; ok {
			isSubscriptionManaged := existingCurrentLine.IsSubscriptionManaged()
			syncIgnore := existingCurrentLine.HasLastLineAnnotation(billing.AnnotationSubscriptionSyncIgnore)

			if isSubscriptionManaged && syncIgnore {
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

		existingPreviousLineIsSubscriptionManaged := existingPreviousLine.IsSubscriptionManaged()
		existingPreviousLineSyncIgnoreAnnotation := existingPreviousLine.HasLastLineAnnotation(billing.AnnotationSubscriptionSyncIgnore)

		if !existingPreviousLineIsSubscriptionManaged || !existingPreviousLineSyncIgnoreAnnotation {
			continue
		}

		existingPreviousLineSyncForceContinuousLinesAnnotation := existingPreviousLine.HasLastLineAnnotation(billing.AnnotationSubscriptionSyncForceContinuousLines)

		if !existingPreviousLineSyncForceContinuousLinesAnnotation {
			continue
		}

		previousServicePeriod := existingPreviousLine.ServicePeriod()
		// The iterator output is already normalized to meter resolution, but this
		// continuity correction reuses a boundary from persisted state. Historical
		// rows can carry sub-second precision that the meter engine cannot query, so
		// we must normalize the carried-over boundary before writing it back into the
		// target state or sync will keep proposing no-op timestamp repairs.
		// TODO: Add a migration to normalize existing billing timestamps to the precision
		// supported by meter queries.
		continuousStart := previousServicePeriod.To.Truncate(streaming.MinimumWindowSizeDuration)
		if line.ServicePeriod.From.Equal(continuousStart) {
			continue
		}

		if !line.ServicePeriod.From.Equal(line.FullServicePeriod.From) {
			return nil, fmt.Errorf("line[%s] service period and full service period start does not match", line.UniqueID)
		}

		inScopeLines[idx].ServicePeriod.From = continuousStart
		inScopeLines[idx].FullServicePeriod.From = continuousStart

		if line.FullServicePeriod.From.Equal(line.BillingPeriod.From) {
			inScopeLines[idx].BillingPeriod.From = continuousStart
		}
	}

	return inScopeLines, nil
}
