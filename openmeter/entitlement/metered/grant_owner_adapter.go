package meteredentitlement

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type entitlementGrantOwner struct {
	featureRepo     feature.FeatureRepo
	entitlementRepo entitlement.EntitlementRepo
	usageResetRepo  UsageResetRepo
	meterRepo       meter.Repository
	logger          *slog.Logger
}

func NewEntitlementGrantOwnerAdapter(
	featureRepo feature.FeatureRepo,
	entitlementRepo entitlement.EntitlementRepo,
	usageResetRepo UsageResetRepo,
	meterRepo meter.Repository,
	logger *slog.Logger,
) grant.OwnerConnector {
	return &entitlementGrantOwner{
		featureRepo:     featureRepo,
		entitlementRepo: entitlementRepo,
		usageResetRepo:  usageResetRepo,
		meterRepo:       meterRepo,
		logger:          logger,
	}
}

func (e *entitlementGrantOwner) GetMeter(ctx context.Context, owner grant.NamespacedOwner) (*grant.OwnerMeter, error) {
	// get feature of entitlement
	entitlement, err := e.entitlementRepo.GetEntitlement(ctx, owner.NamespacedID())
	if err != nil {
		e.logger.Debug(fmt.Sprintf("failed to get entitlement for owner %s in namespace %s: %s", string(owner.ID), owner.Namespace, err))
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
	meter, err := e.meterRepo.GetMeterByIDOrSlug(ctx, feature.Namespace, *feature.MeterSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to get meter: %w", err)
	}

	queryParams := &streaming.QueryParams{
		Aggregation: meter.Aggregation,
	}

	if feature.MeterGroupByFilters != nil {
		queryParams.FilterGroupBy = map[string][]string{}
		for k, v := range feature.MeterGroupByFilters {
			queryParams.FilterGroupBy[k] = []string{v}
		}
	}

	return &grant.OwnerMeter{
		MeterSlug:     meter.Slug,
		DefaultParams: queryParams,
		WindowSize:    meter.WindowSize,
		SubjectKey:    entitlement.SubjectKey,
	}, nil
}

func (e *entitlementGrantOwner) GetStartOfMeasurement(ctx context.Context, owner grant.NamespacedOwner) (time.Time, error) {
	entitlement, err := e.entitlementRepo.GetEntitlement(ctx, owner.NamespacedID())
	if err != nil {
		return time.Time{}, &grant.OwnerNotFoundError{
			Owner:          owner,
			AttemptedOwner: "entitlement",
		}
	}

	metered, err := ParseFromGenericEntitlement(entitlement)
	if err != nil {
		return time.Time{}, err
	}

	return metered.MeasureUsageFrom, nil
}

func (e *entitlementGrantOwner) GetUsagePeriodStartAt(ctx context.Context, owner grant.NamespacedOwner, at time.Time) (time.Time, error) {
	// If this is the first period then return start of measurement, otherwise calculate based on anchor.
	// To know if this is the first period check if usage has been reset.

	lastUsageReset, err := e.usageResetRepo.GetLastAt(ctx, owner.NamespacedID(), at)
	if _, ok := err.(*UsageResetNotFoundError); ok {
		return e.GetStartOfMeasurement(ctx, owner)
	}
	if err != nil {
		return time.Time{}, err
	}

	return lastUsageReset.ResetTime, nil
}

func (e *entitlementGrantOwner) GetOwnerSubjectKey(ctx context.Context, owner grant.NamespacedOwner) (string, error) {
	entitlement, err := e.entitlementRepo.GetEntitlement(ctx, owner.NamespacedID())
	if err != nil {
		return "", &grant.OwnerNotFoundError{
			Owner:          owner,
			AttemptedOwner: "entitlement",
		}
	}
	return entitlement.SubjectKey, nil
}

func (e *entitlementGrantOwner) GetPeriodStartTimesBetween(ctx context.Context, owner grant.NamespacedOwner, from, to time.Time) ([]time.Time, error) {
	times := []time.Time{}
	usageResets, err := e.usageResetRepo.GetBetween(ctx, owner.NamespacedID(), from, to)
	if err != nil {
		return nil, err
	}
	for _, reset := range usageResets {
		times = append(times, reset.ResetTime)
	}
	return times, nil
}

func (e *entitlementGrantOwner) EndCurrentUsagePeriod(ctx context.Context, owner grant.NamespacedOwner, params grant.EndCurrentUsagePeriodParams) error {
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
		if !params.At.After(currentStartAt) {
			return nil, &models.GenericUserError{Message: "can only end usage period after current period start time"}
		}

		if err := e.updateEntitlementUsagePeriod(txCtx, owner, params); err != nil {
			return nil, fmt.Errorf("failed to update entitlement usage period: %w", err)
		}

		// Save usage reset
		return nil, e.usageResetRepo.Save(txCtx, UsageResetTime{
			NamespacedModel: models.NamespacedModel{
				Namespace: owner.Namespace,
			},
			EntitlementID: owner.NamespacedID().ID,
			ResetTime:     params.At,
		})
	})
	return err
}

func (e *entitlementGrantOwner) updateEntitlementUsagePeriod(ctx context.Context, owner grant.NamespacedOwner, params grant.EndCurrentUsagePeriodParams) error {
	entitlementEntity, err := e.entitlementRepo.GetEntitlement(ctx, owner.NamespacedID())
	if err != nil {
		return err
	}

	if entitlementEntity.UsagePeriod == nil {
		return fmt.Errorf("entitlement=%s, namespace=%s does not have a usage period set, cannot guess interval", owner.ID, owner.Namespace)
	}

	usagePeriod := entitlementEntity.UsagePeriod

	var newAnchor *time.Time

	if !params.RetainAnchor {
		usagePeriod.Anchor = params.At
		newAnchor = &params.At
	}

	newCurrentUsagePeriod, err := usagePeriod.GetCurrentPeriodAt(params.At)
	if err != nil {
		return fmt.Errorf("failed to get next reset: %w", err)
	}

	return e.entitlementRepo.UpdateEntitlementUsagePeriod(
		ctx,
		owner.NamespacedID(),
		entitlement.UpdateEntitlementUsagePeriodParams{
			NewAnchor:          newAnchor,
			CurrentUsagePeriod: newCurrentUsagePeriod,
		})
}

func (e *entitlementGrantOwner) LockOwnerForTx(ctx context.Context, owner grant.NamespacedOwner) error {
	// If we're not in a transaction this method has to fail
	tx, err := entutils.GetDriverFromContext(ctx)
	if err != nil {
		return fmt.Errorf("lock owner for tx must be called in a transaction: %w", err)
	}
	return e.entitlementRepo.LockEntitlementForTx(ctx, tx, owner.NamespacedID())
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
