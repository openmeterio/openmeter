package meteredentitlement

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type entitlementGrantOwner struct {
	featureRepo     feature.FeatureRepo
	entitlementRepo entitlement.EntitlementRepo
	usageResetRepo  UsageResetRepo
	meterService    meter.Service
	logger          *slog.Logger
}

func NewEntitlementGrantOwnerAdapter(
	featureRepo feature.FeatureRepo,
	entitlementRepo entitlement.EntitlementRepo,
	usageResetRepo UsageResetRepo,
	meterService meter.Service,
	logger *slog.Logger,
) grant.OwnerConnector {
	return &entitlementGrantOwner{
		featureRepo:     featureRepo,
		entitlementRepo: entitlementRepo,
		usageResetRepo:  usageResetRepo,
		meterService:    meterService,
		logger:          logger,
	}
}

func (e *entitlementGrantOwner) GetMeter(ctx context.Context, owner models.NamespacedID) (*grant.OwnerMeter, error) {
	// get feature of entitlement
	entitlement, err := e.entitlementRepo.GetEntitlement(ctx, owner)
	if err != nil {
		e.logger.Debug(fmt.Sprintf("failed to get entitlement for owner %s in namespace %s: %s", owner.ID, owner.Namespace, err))
		return nil, &grant.OwnerNotFoundError{
			Owner:          owner,
			AttemptedOwner: "entitlement",
		}
	}
	feature, err := getRepoMaybeInTx(ctx, e.featureRepo, e.featureRepo).GetByIdOrKey(ctx, owner.Namespace, entitlement.FeatureID, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature of entitlement: %w", err)
	}

	if feature.MeterSlug == nil {
		return nil, fmt.Errorf("feature does not have a meter")
	}

	// meterrepo is not transactional
	meter, err := e.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
		Namespace: owner.Namespace,
		IDOrSlug:  *feature.MeterSlug,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get meter: %w", err)
	}

	queryParams := streaming.QueryParams{
		FilterSubject: []string{entitlement.SubjectKey},
	}

	if feature.MeterGroupByFilters != nil {
		queryParams.FilterGroupBy = map[string][]string{}
		for k, v := range feature.MeterGroupByFilters {
			queryParams.FilterGroupBy[k] = []string{v}
		}
	}

	return &grant.OwnerMeter{
		Meter:         meter,
		DefaultParams: queryParams,
	}, nil
}

func (e *entitlementGrantOwner) GetStartOfMeasurement(ctx context.Context, owner models.NamespacedID) (time.Time, error) {
	owningEntitlement, err := e.entitlementRepo.GetEntitlement(ctx, owner)
	if err != nil {
		if _, ok := lo.ErrorsAs[*entitlement.NotFoundError](err); ok {
			return time.Time{}, &grant.OwnerNotFoundError{
				Owner:          owner,
				AttemptedOwner: "entitlement",
			}
		}

		return time.Time{}, fmt.Errorf("failed to get entitlement: %w", err)
	}

	metered, err := ParseFromGenericEntitlement(owningEntitlement)
	if err != nil {
		return time.Time{}, err
	}

	return metered.MeasureUsageFrom, nil
}

// The current usage period start time is either the current period start time, or if this is the first period then the start of measurement
func (e *entitlementGrantOwner) GetUsagePeriodStartAt(ctx context.Context, owner models.NamespacedID, at time.Time) (time.Time, error) {
	ent, err := e.entitlementRepo.GetEntitlement(ctx, owner)
	if err != nil {
		return time.Time{}, err
	}

	lastReset, err := e.usageResetRepo.GetLastAt(ctx, owner, at)
	if err != nil {
		// If it's a not found error thats ok, it means there are no manual resets yet. Otherwise we return an error
		if _, ok := lo.ErrorsAs[*UsageResetNotFoundError](err); !ok {
			return time.Time{}, err
		}
	}

	cp, ok := ent.CalculateCurrentUsagePeriodAt(lastReset.Anchor, at)
	if !ok {
		return time.Time{}, fmt.Errorf("failed to calculate current usage period")
	}

	return cp.From, nil
}

func (e *entitlementGrantOwner) GetOwnerSubjectKey(ctx context.Context, owner models.NamespacedID) (string, error) {
	entitlement, err := e.entitlementRepo.GetEntitlement(ctx, owner)
	if err != nil {
		return "", &grant.OwnerNotFoundError{
			Owner:          owner,
			AttemptedOwner: "entitlement",
		}
	}
	return entitlement.SubjectKey, nil
}

func (e *entitlementGrantOwner) GetResetBehavior(ctx context.Context, owner models.NamespacedID) (grant.ResetBehavior, error) {
	ent, err := e.entitlementRepo.GetEntitlement(ctx, owner)
	if err != nil {
		return grant.ResetBehavior{}, err
	}

	mEnt, err := ParseFromGenericEntitlement(ent)
	if err != nil {
		return grant.ResetBehavior{}, err
	}

	return grant.ResetBehavior{
		PreserveOverage: mEnt.PreserveOverageAtReset,
	}, nil
}

func (e *entitlementGrantOwner) GetResetTimelineInclusive(ctx context.Context, owner models.NamespacedID, period timeutil.Period) (timeutil.SimpleTimeline, error) {
	var def timeutil.SimpleTimeline

	// Let's fetch the owner entitlement
	ent, err := e.entitlementRepo.GetEntitlement(ctx, owner)
	if err != nil {
		return def, err
	}

	// 1. Let's find the last reset time before the period. It doesn't necessarily exist!
	lastReset, err := e.usageResetRepo.GetLastAt(ctx, owner, period.From)
	if err != nil {
		if _, ok := lo.ErrorsAs[*UsageResetNotFoundError](err); ok {
			// We build a synthetic last reset based on the entitlement's usage period
			// This will be the case if there have been no resets for this entitlement
			up := entitlement.UsagePeriod{
				Anchor:   *ent.OriginalUsagePeriodAnchor,
				Interval: ent.UsagePeriod.Interval,
			}

			currentUsagePeriod, err := up.GetCurrentPeriodAt(period.From)
			if err != nil {
				return def, err
			}

			lastReset = UsageResetTime{
				NamespacedModel: models.NamespacedModel{
					Namespace: owner.Namespace,
				},
				EntitlementID: owner.ID,
				ResetTime:     currentUsagePeriod.From,
				Anchor:        currentUsagePeriod.From,
			}
		} else {
			return def, err
		}
	}

	// 2. Now let's find all the resets between the period
	usageResets, err := e.usageResetRepo.GetBetween(ctx, owner, period)
	if err != nil {
		return def, err
	}

	// The timeline would look something like:
	// [lastReset = mReset0], [mReset1], ...[mResetN], [period.To]
	// (where mReset0 might be equal to period.From AND mResetN might be equal to period.To)
	// For these times, in any period between them there might be programmatic resets

	resets := []UsageResetTime{lastReset}
	// usageResets are sorted ASC by ResetTime
	for _, reset := range usageResets {
		// The period start could be a reset time so we need to dedupe it (would be both lastReset and usageResets[0])
		// usageResets are sorted ASC by ResetTime
		if l, _ := lo.Last(resets); reset.ResetTime.After(l.ResetTime) {
			resets = append(resets, reset)
		}
	}

	// Let's convert it to a timeline
	resetTimeline := timeutil.NewTimeline(lo.Map(resets, func(reset UsageResetTime, _ int) timeutil.Timed[UsageResetTime] {
		return timeutil.AsTimed(func(reset UsageResetTime) time.Time { return reset.ResetTime })(reset)
	}))

	resetTimes := make([]time.Time, 0)

	periods := resetTimeline.GetPeriods()

	// We need to go through all the manual reset times and check if programmatic resets occur in between
	for idx, p := range periods {
		resetTimes = append(resetTimes, p.From)

		// There is always one less period than values in the timeline
		val := resetTimeline.GetAt(idx)

		// Let's build the UsagePeriod for the period
		up := entitlement.UsagePeriod{
			Anchor:   val.GetValue().Anchor,
			Interval: ent.UsagePeriod.Interval,
		}

		inbetweenTimes, err := e.getProgrammaticResetTimesInPeriodExclusiveInclusive(p, up)
		if err != nil {
			return def, err
		}

		// inbetweenTimes might contain the period end, so we need to dedupe it
		if len(inbetweenTimes) > 0 && inbetweenTimes[len(inbetweenTimes)-1].Equal(p.To) {
			inbetweenTimes = inbetweenTimes[:len(inbetweenTimes)-1]
		}

		resetTimes = append(resetTimes, inbetweenTimes...)
	}

	// Now, let's add the last manual reset time as well (as the cycle above only added period starts)
	lastPeriod := periods[len(periods)-1]

	// Let's make sure we're not adding it twice
	if !lastPeriod.From.Equal(lastPeriod.To) {
		resetTimes = append(resetTimes, lastPeriod.To)
	}

	lastResetTime, ok := lo.Last(resetTimes)
	if !ok {
		// Cannot happen as we always add a first value
		return def, fmt.Errorf("no last reset found")
	}

	// Now we need to check the final period, which is [lastResetTime, period.To]
	finalPeriod := timeutil.Period{
		From: lastResetTime,
		To:   period.To,
	}

	// Let's build the usage period for the final period
	finalReset, _ := lo.Last(resets)

	finalUsagePeriod := entitlement.UsagePeriod{
		Anchor:   finalReset.Anchor,
		Interval: ent.UsagePeriod.Interval,
	}

	lastPeriodTimes, err := e.getProgrammaticResetTimesInPeriodExclusiveInclusive(finalPeriod, finalUsagePeriod)
	if err != nil {
		return def, err
	}

	resetTimes = append(resetTimes, lastPeriodTimes...)

	// Let's return in UTC
	return timeutil.NewSimpleTimeline(lo.Map(resetTimes, func(t time.Time, _ int) time.Time { return t.UTC() })), nil
}

// Returns all programmatic reset times in the period (start inclusive end exclusive)
func (e *entitlementGrantOwner) getProgrammaticResetTimesInPeriodExclusiveInclusive(period timeutil.Period, up entitlement.UsagePeriod) ([]time.Time, error) {
	rts := []time.Time{}

	upr := up.AsRecurrence()

	currentUP, err := up.GetCurrentPeriodAt(period.From)
	if err != nil {
		return nil, err
	}

	// Now let's check if there are any programmatic resets during the period (start and end exclusive!)

	// Let's find the last reset time when the period starts
	resetTime := currentUP.From

	// We might silence an error here at upr.Next but in practice this should never happen
	for rt := resetTime; err == nil && !rt.After(period.To); rt, err = upr.Next(rt) {
		if period.ContainsInclusive(rt) && period.From.Compare(rt) != 0 {
			rts = append(rts, rt)
		}
	}

	return rts, nil
}

func (e *entitlementGrantOwner) EndCurrentUsagePeriod(ctx context.Context, owner models.NamespacedID, params grant.EndCurrentUsagePeriodParams) error {
	// If we're not in a transaction this method should fail
	_, err := transaction.GetDriverFromContext(ctx)
	if err != nil {
		return fmt.Errorf("end current usage period must be called in a transaction: %w", err)
	}

	_, err = transaction.Run(ctx, e.featureRepo, func(txCtx context.Context) (*interface{}, error) {
		// Check if time is after current start time. If so then we can end the period
		currentStartAt, err := e.GetUsagePeriodStartAt(txCtx, owner, params.At)
		if err != nil {
			return nil, fmt.Errorf("failed to get current usage period start time: %w", err)
		}
		if params.At.Before(currentStartAt) {
			return nil, models.NewGenericValidationError(fmt.Errorf("cannot end usage period before current period start time"))
		}

		if err := e.updateEntitlementUsagePeriod(txCtx, owner, params); err != nil {
			return nil, fmt.Errorf("failed to update entitlement usage period: %w", err)
		}

		// Now let's see what the anchor is after the update
		entitlementEntity, err := e.entitlementRepo.GetEntitlement(txCtx, owner)
		if err != nil {
			return nil, fmt.Errorf("failed to get entitlement: %w", err)
		}

		anchor := entitlementEntity.UsagePeriod.Anchor
		if !params.RetainAnchor {
			anchor = params.At
		}

		// Save usage reset
		return nil, e.usageResetRepo.Save(txCtx, UsageResetTime{
			NamespacedModel: models.NamespacedModel{
				Namespace: owner.Namespace,
			},
			EntitlementID: owner.ID,
			ResetTime:     params.At,
			Anchor:        anchor,
		})
	})
	return err
}

func (e *entitlementGrantOwner) updateEntitlementUsagePeriod(ctx context.Context, owner models.NamespacedID, params grant.EndCurrentUsagePeriodParams) error {
	entitlementEntity, err := e.entitlementRepo.GetEntitlement(ctx, owner)
	if err != nil {
		return err
	}

	if entitlementEntity.UsagePeriod == nil {
		return fmt.Errorf("entitlement=%s, namespace=%s does not have a usage period set, cannot guess interval", owner.ID, owner.Namespace)
	}

	usagePeriod := entitlementEntity.UsagePeriod

	if !params.RetainAnchor {
		usagePeriod.Anchor = params.At
	}

	newCurrentUsagePeriod, err := usagePeriod.GetCurrentPeriodAt(params.At)
	if err != nil {
		return fmt.Errorf("failed to get next reset: %w", err)
	}

	return e.entitlementRepo.UpdateEntitlementUsagePeriod(
		ctx,
		owner,
		entitlement.UpdateEntitlementUsagePeriodParams{
			CurrentUsagePeriod: newCurrentUsagePeriod,
		})
}

func (e *entitlementGrantOwner) LockOwnerForTx(ctx context.Context, owner models.NamespacedID) error {
	// If we're not in a transaction this method has to fail
	tx, err := entutils.GetDriverFromContext(ctx)
	if err != nil {
		return fmt.Errorf("lock owner for tx must be called in a transaction: %w", err)
	}
	return e.entitlementRepo.LockEntitlementForTx(ctx, tx, owner)
}

// FIXME: this is a terrible hack to conditionally catch transactions
func getRepoMaybeInTx[T any](ctx context.Context, repo T, txUser entutils.TxUser[T]) T {
	if ctxTx, err := entutils.GetDriverFromContext(ctx); err == nil {
		// we're already in a tx
		return txUser.WithTx(ctx, ctxTx)
	} else {
		return repo
	}
}
