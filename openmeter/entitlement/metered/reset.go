package meteredentitlement

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	eventmodels "github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (e *connector) ResetEntitlementUsage(ctx context.Context, entitlementID models.NamespacedID, params ResetEntitlementUsageParams) (*EntitlementBalance, error) {
	ctx, span := e.tracer.Start(ctx, "ResetEntitlementUsage")
	defer span.End()

	return transaction.Run(ctx, e.grantRepo, func(ctx context.Context) (*EntitlementBalance, error) {
		owner := models.NamespacedID{
			Namespace: entitlementID.Namespace,
			ID:        entitlementID.ID,
		}

		ent, err := e.entitlementRepo.GetEntitlement(ctx, entitlementID)
		if err != nil {
			return nil, fmt.Errorf("failed to get entitlement: %w", err)
		}

		mEnt, err := ParseFromGenericEntitlement(ent)
		if err != nil {
			return nil, fmt.Errorf("failed to parse entitlement: %w", err)
		}

		if err := e.hooks.PreUpdate(ctx, mEnt); err != nil {
			return nil, err
		}

		balanceAfterReset, err := e.balanceConnector.ResetUsageForOwner(ctx, owner, credit.ResetUsageForOwnerParams{
			At:              params.At,
			RetainAnchor:    params.RetainAnchor,
			PreserveOverage: defaultx.WithDefault(params.PreserveOverage, mEnt.PreserveOverageAtReset),
		})
		if err != nil {
			if _, ok := lo.ErrorsAs[*grant.OwnerNotFoundError](err); ok {
				return nil, &entitlement.NotFoundError{EntitlementID: entitlementID}
			}
			return nil, err
		}

		event := EntitlementResetEventV3{
			EntitlementID: entitlementID.ID,
			Namespace: eventmodels.NamespaceID{
				ID: entitlementID.Namespace,
			},
			CustomerID:       ent.CustomerID,
			ResetAt:          params.At,
			RetainAnchor:     params.RetainAnchor,
			ResetRequestedAt: time.Now(),
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
	entitlements, err := c.entitlementRepo.ListActiveEntitlementsWithExpiredUsagePeriod(ctx, entitlement.ListExpiredEntitlementsParams{
		Namespaces:    []string{namespace},
		Highwatermark: highwatermark,
	})
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
