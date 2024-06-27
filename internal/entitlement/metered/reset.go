package meteredentitlement

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/pkg/models"
)

// This is just a wrapper around credot.BalanceConnector.ResetUsageForOwner
func (e *connector) ResetEntitlementUsage(ctx context.Context, entitlementID models.NamespacedID, params ResetEntitlementUsageParams) (*EntitlementBalance, error) {
	owner := credit.NamespacedGrantOwner{
		Namespace: entitlementID.Namespace,
		ID:        credit.GrantOwner(entitlementID.ID),
	}

	balanceAfterReset, err := e.balanceConnector.ResetUsageForOwner(ctx, owner, credit.ResetUsageForOwnerParams{
		At:           params.At,
		RetainAnchor: params.RetainAnchor,
	})
	if err != nil {
		if _, ok := err.(*credit.OwnerNotFoundError); ok {
			return nil, &entitlement.NotFoundError{EntitlementID: entitlementID}
		}
		return nil, err
	}

	return &EntitlementBalance{
		EntitlementID: entitlementID.ID,
		Balance:       balanceAfterReset.Balance(),
		UsageInPeriod: 0.0, // you cannot have usage right after a reset
		Overage:       balanceAfterReset.Overage,
		StartOfPeriod: params.At,
	}, nil
}

func (c *connector) ResetEntitlementsWithExpiredUsagePeriod(ctx context.Context, namespace string, highwatermark time.Time) ([]models.NamespacedID, error) {
	entitlements, err := c.entitlementRepo.ListEntitlementsWithExpiredUsagePeriod(ctx, namespace, highwatermark)
	if err != nil {
		return nil, fmt.Errorf("failed to list entitlements with due reset: %w", err)
	}

	result := make([]models.NamespacedID, 0, len(entitlements))

	var finalError error
	for _, ent := range entitlements {
		namespacedID := models.NamespacedID{Namespace: namespace, ID: ent.ID}

		_, err := c.ResetEntitlementUsage(ctx,
			namespacedID,
			ResetEntitlementUsageParams{
				At:           ent.CurrentUsagePeriod.To,
				RetainAnchor: true,
			})
		if err != nil {
			finalError = errors.Join(finalError, fmt.Errorf("failed to reset entitlement usage ns=%s id=%s: %w", namespace, ent.ID, err))
		}

		result = append(result, namespacedID)
	}
	return result, finalError
}
