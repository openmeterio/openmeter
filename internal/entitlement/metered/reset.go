package meteredentitlement

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/grant"
	"github.com/openmeterio/openmeter/internal/entitlement"
	eventmodels "github.com/openmeterio/openmeter/internal/event/models"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (e *connector) ResetEntitlementUsage(ctx context.Context, entitlementID models.NamespacedID, params ResetEntitlementUsageParams) (*EntitlementBalance, error) {
	return entutils.StartAndRunTx(ctx, e.entitlementRepo, func(ctx context.Context, tx *entutils.TxDriver) (*EntitlementBalance, error) {
		txCtx := entutils.NewTxContext(ctx, tx)

		owner := grant.NamespacedOwner{
			Namespace: entitlementID.Namespace,
			ID:        grant.Owner(entitlementID.ID),
		}

		ent, err := e.entitlementRepo.WithTx(txCtx, tx).GetEntitlement(txCtx, entitlementID)
		if err != nil {
			return nil, fmt.Errorf("failed to get entitlement: %w", err)
		}

		_, err = ParseFromGenericEntitlement(ent)
		if err != nil {
			return nil, fmt.Errorf("failed to parse entitlement: %w", err)
		}

		balanceAfterReset, err := e.balanceConnector.ResetUsageForOwner(txCtx, owner, credit.ResetUsageForOwnerParams{
			At:              params.At,
			RetainAnchor:    params.RetainAnchor,
			PreserveOverage: params.PreserveOverage,
		})
		if err != nil {
			if _, ok := err.(*grant.OwnerNotFoundError); ok {
				return nil, &entitlement.NotFoundError{EntitlementID: entitlementID}
			}
			return nil, err
		}

		event := EntitlementResetEvent{
			EntitlementID: entitlementID.ID,
			Namespace: eventmodels.NamespaceID{
				ID: entitlementID.Namespace,
			},
			Subject: eventmodels.SubjectKeyAndID{
				Key: ent.SubjectKey,
			},
			ResetAt:      params.At,
			RetainAnchor: params.RetainAnchor,
		}

		if err := e.publisher.Publish(ctx, event); err != nil {
			return nil, err
		}

		return &EntitlementBalance{
			EntitlementID: entitlementID.ID,
			Balance:       balanceAfterReset.Balance(),
			UsageInPeriod: 0.0, // you cannot have usage right after a reset
			Overage:       balanceAfterReset.Overage,
			StartOfPeriod: params.At,
		}, nil
	})
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
