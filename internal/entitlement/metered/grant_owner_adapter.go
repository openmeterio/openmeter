package meteredentitlement

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type entitlementGrantOwner struct {
	featureRepo     productcatalog.FeatureRepo
	entitlementRepo entitlement.EntitlementRepo
	usageResetRepo  UsageResetRepo
	meterRepo       meter.Repository
	logger          *slog.Logger
}

func NewEntitlementGrantOwnerAdapter(
	featureRepo productcatalog.FeatureRepo,
	entitlementRepo entitlement.EntitlementRepo,
	usageResetRepo UsageResetRepo,
	meterRepo meter.Repository,
	logger *slog.Logger,
) credit.OwnerConnector {
	return &entitlementGrantOwner{
		featureRepo:     featureRepo,
		entitlementRepo: entitlementRepo,
		usageResetRepo:  usageResetRepo,
		meterRepo:       meterRepo,
		logger:          logger,
	}
}

func (e *entitlementGrantOwner) GetMeter(ctx context.Context, owner credit.NamespacedGrantOwner) (*credit.Meter, error) {
	// get feature of entitlement
	entitlement, err := e.entitlementRepo.GetEntitlement(ctx, owner.NamespacedID())
	if err != nil {
		e.logger.Debug(fmt.Sprintf("failed to get entitlement for owner %s in namespace %s: %s", string(owner.ID), owner.Namespace, err))
		return nil, &credit.OwnerNotFoundError{
			Owner:          owner,
			AttemptedOwner: "entitlement",
		}
	}
	feature, err := e.featureRepo.GetByIdOrKey(ctx, owner.Namespace, entitlement.FeatureID, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature of entitlement: %w", err)
	}

	if feature.MeterSlug == nil {
		return nil, fmt.Errorf("feature does not have a meter")
	}

	meter, err := e.meterRepo.GetMeterByIDOrSlug(ctx, feature.Namespace, *feature.MeterSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to get meter: %w", err)
	}

	queryParams := &streaming.QueryParams{
		Aggregation: meter.Aggregation,
		WindowSize:  &meter.WindowSize,
	}

	if feature.MeterGroupByFilters != nil {
		queryParams.FilterGroupBy = map[string][]string{}
		for k, v := range *feature.MeterGroupByFilters {
			queryParams.FilterGroupBy[k] = []string{v}
		}
	}

	return &credit.Meter{
		MeterSlug:     meter.Slug,
		DefaultParams: queryParams,
		WindowSize:    meter.WindowSize,
	}, nil
}

func (e *entitlementGrantOwner) GetStartOfMeasurement(ctx context.Context, owner credit.NamespacedGrantOwner) (time.Time, error) {
	entitlement, err := e.entitlementRepo.GetEntitlement(ctx, owner.NamespacedID())
	if err != nil {
		return time.Time{}, &credit.OwnerNotFoundError{
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

func (e *entitlementGrantOwner) GetUsagePeriodStartAt(ctx context.Context, owner credit.NamespacedGrantOwner, at time.Time) (time.Time, error) {
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

func (e *entitlementGrantOwner) GetPeriodStartTimesBetween(ctx context.Context, owner credit.NamespacedGrantOwner, from, to time.Time) ([]time.Time, error) {
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

// FIXME: this is a terrible hack, write generic Atomicity stuff for connectors...
func (e *entitlementGrantOwner) EndCurrentUsagePeriodTx(ctx context.Context, tx *entutils.TxDriver, owner credit.NamespacedGrantOwner, params credit.EndCurrentUsagePeriodParams) error {
	_, err := entutils.RunInTransaction(ctx, tx, func(ctx context.Context, tx *entutils.TxDriver) (*interface{}, error) {
		// Check if time is after current start time. If so then we can end the period
		currentStartAt, err := e.GetUsagePeriodStartAt(ctx, owner, params.At)
		if err != nil {
			return nil, fmt.Errorf("failed to get current usage period start time: %w", err)
		}
		if !params.At.After(currentStartAt) {
			return nil, &models.GenericUserError{Message: "can only end usage period after current period start time"}
		}

		if err := e.updateEntitlementUsagePeriod(ctx, tx, owner, params); err != nil {
			return nil, fmt.Errorf("failed to update entitlement usage period: %w", err)
		}

		// Save usage reset
		return nil, e.usageResetRepo.WithTx(ctx, tx).Save(ctx, UsageResetTime{
			NamespacedModel: models.NamespacedModel{
				Namespace: owner.Namespace,
			},
			EntitlementID: owner.NamespacedID().ID,
			ResetTime:     params.At,
		})
	})
	return err
}

func (e *entitlementGrantOwner) updateEntitlementUsagePeriod(ctx context.Context, tx *entutils.TxDriver, owner credit.NamespacedGrantOwner, params credit.EndCurrentUsagePeriodParams) error {
	er := e.entitlementRepo.WithTx(ctx, tx)

	entitlementEntity, err := er.GetEntitlement(ctx, owner.NamespacedID())
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

	newCurrentUsagePeriod, err := usagePeriod.GetCurrentPeriod()
	if err != nil {
		return fmt.Errorf("failed to get next reset: %w", err)
	}

	return er.UpdateEntitlementUsagePeriod(
		ctx,
		owner.NamespacedID(),
		entitlement.UpdateEntitlementUsagePeriodParams{
			NewAnchor:          newAnchor,
			CurrentUsagePeriod: newCurrentUsagePeriod,
		})
}

// FIXME: this is a terrible hack using select for udpate...
func (e *entitlementGrantOwner) LockOwnerForTx(ctx context.Context, tx *entutils.TxDriver, owner credit.NamespacedGrantOwner) error {
	return e.entitlementRepo.WithTx(ctx, tx).LockEntitlementForTx(ctx, owner.NamespacedID())
}
