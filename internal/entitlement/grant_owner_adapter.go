package entitlement

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/internal/streaming"
)

type entitlementGrantOwner struct {
	fdb    productcatalog.FeatureDBConnector
	edb    EntitlementDBConnector
	logger *slog.Logger
}

func NewEntitlementGrantOwnerAdapter(
	fdb productcatalog.FeatureDBConnector,
	edb EntitlementDBConnector,
	logger *slog.Logger,
) credit.OwnerConnector {
	return &entitlementGrantOwner{
		fdb:    fdb,
		edb:    edb,
		logger: logger,
	}
}

func (e *entitlementGrantOwner) GetOwnerQueryParams(ctx context.Context, owner credit.NamespacedGrantOwner) (meterSlug string, defaultParams *streaming.QueryParams, err error) {
	// get feature of entitlement
	entitlement, err := e.edb.GetEntitlement(ctx, NamespacedEntitlementID{
		Namespace: owner.Namespace,
		ID:        EntitlementID(owner.ID),
	})
	if err != nil {
		e.logger.Debug("failed to get entitlement for owner %s in namespace %s: %w", string(owner.ID), owner.Namespace, err)
		return "", nil, credit.OwnerNotFoundError{
			Owner:          owner,
			AttemptedOwner: "entitlement",
		}
	}
	feature, err := e.fdb.GetByID(ctx, productcatalog.NamespacedFeatureID{
		Namespace: owner.Namespace,
		ID:        productcatalog.FeatureID(entitlement.FeatureID),
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
	entitlement, err := e.edb.GetEntitlement(ctx, NamespacedEntitlementID{
		Namespace: owner.Namespace,
		ID:        EntitlementID(owner.ID),
	})
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

	// old date
	return time.Parse(time.RFC3339, "2021-03-01T00:00:00Z")
}

func (e *entitlementGrantOwner) GetPeriodStartTimesBetween(ctx context.Context, owner credit.NamespacedGrantOwner, from, to time.Time) ([]time.Time, error) {
	panic("implement me")
}

func (e *entitlementGrantOwner) EndCurrentUsagePeriod(ctx context.Context, owner credit.NamespacedGrantOwner, at time.Time) error {
	panic("implement me")
}
