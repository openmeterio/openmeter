package entitlement

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

type entitlementGrantOwner struct {
	fdb    productcatalog.FeatureDBConnector
	edb    EntitlementDBConnector
	urdb   UsageResetDBConnector
	logger *slog.Logger
}

func NewEntitlementGrantOwnerAdapter(
	fdb productcatalog.FeatureDBConnector,
	edb EntitlementDBConnector,
	urdb UsageResetDBConnector,
	logger *slog.Logger,
) credit.OwnerConnector {
	return &entitlementGrantOwner{
		fdb:    fdb,
		edb:    edb,
		urdb:   urdb,
		logger: logger,
	}
}

func (e *entitlementGrantOwner) GetOwnerQueryParams(ctx context.Context, owner credit.NamespacedGrantOwner) (meterSlug string, defaultParams *streaming.QueryParams, err error) {
	// get feature of entitlement
	entitlement, err := e.edb.GetEntitlement(ctx, e.ownerToEntitlement(owner))
	if err != nil {
		e.logger.Debug("failed to get entitlement for owner %s in namespace %s: %w", string(owner.ID), owner.Namespace, err)
		return "", nil, credit.OwnerNotFoundError{
			Owner:          owner,
			AttemptedOwner: "entitlement",
		}
	}
	feature, err := e.fdb.GetByID(ctx, models.NamespacedID{
		Namespace: owner.Namespace,
		ID:        entitlement.FeatureID,
	})
	if err != nil {
		return "", nil, fmt.Errorf("failed to get feature of entitlement: %w", err)
	}

	queryParams := &streaming.QueryParams{}

	if feature.MeterGroupByFilters != nil {
		queryParams.FilterGroupBy = map[string][]string{}
		for k, v := range *feature.MeterGroupByFilters {
			queryParams.FilterGroupBy[k] = []string{v}
		}
	}

	return feature.MeterSlug, queryParams, nil
}

func (e *entitlementGrantOwner) GetStartOfMeasurement(ctx context.Context, owner credit.NamespacedGrantOwner) (time.Time, error) {
	entitlement, err := e.edb.GetEntitlement(ctx, e.ownerToEntitlement(owner))
	if err != nil {
		return time.Time{}, credit.OwnerNotFoundError{
			Owner:          owner,
			AttemptedOwner: "entitlement",
		}
	}

	return entitlement.MeasureUsageFrom, nil
}

func (e *entitlementGrantOwner) GetCurrentUsagePeriodStartAt(ctx context.Context, owner credit.NamespacedGrantOwner, at time.Time) (time.Time, error) {
	// TODO: implement this!

	// if this is the first period then return start of measurement, otherwise calculate based on anchor
	// to know if this is the first period check if usage has been reset

	lastUsageReset, err := e.urdb.GetLastAt(ctx, e.ownerToEntitlement(owner), at)
	if _, ok := err.(UsageResetNotFoundError); ok {
		return e.GetStartOfMeasurement(ctx, owner)
	}
	if err != nil {
		return time.Time{}, err
	}

	return lastUsageReset.ResetTime, nil
}

func (e *entitlementGrantOwner) GetPeriodStartTimesBetween(ctx context.Context, owner credit.NamespacedGrantOwner, from, to time.Time) ([]time.Time, error) {
	times := []time.Time{}
	usageResets, err := e.urdb.GetBetween(ctx, e.ownerToEntitlement(owner), from, to)
	if err != nil {
		return nil, err
	}
	for _, reset := range usageResets {
		times = append(times, reset.ResetTime)
	}
	return times, nil
}

func (e *entitlementGrantOwner) EndCurrentUsagePeriod(ctx context.Context, owner credit.NamespacedGrantOwner, at time.Time) error {
	// Check if time is after current start time. If so then we can end the period
	currentStartAt, err := e.GetCurrentUsagePeriodStartAt(ctx, owner, at)
	if err != nil {
		return fmt.Errorf("failed to get current usage period start time: %w", err)
	}
	if at.Before(currentStartAt) || at.Equal(currentStartAt) {
		return fmt.Errorf("can only end usage period after current period start time")
	}

	// Save usage reset
	return e.urdb.Save(ctx, UsageResetTime{
		EntitlementID: e.ownerToEntitlement(owner).ID,
		ResetTime:     at,
	})
}

func (e *entitlementGrantOwner) ownerToEntitlement(owner credit.NamespacedGrantOwner) models.NamespacedID {
	return models.NamespacedID{
		Namespace: owner.Namespace,
		ID:        string(owner.ID),
	}
}
