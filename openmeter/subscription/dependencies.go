package subscription

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/samber/lo"
)

func (c *connector) createRateCardEntitlement(ctx context.Context, rateCard RateCard) error {
	// Lets create the entitlements

	// FIXME: clean up this control flow

	entSpec, err := rateCard.GetEntitlementSpec()
	// If it's an unexpected error we return with the error
	if _, ok := lo.ErrorsAs[*DoesntHaveResourceError](err); !ok && err != nil {
		return fmt.Errorf("failed to get entitlement spec: %w", err)
	} else if err == nil {
		// If theres no error, we create the entitlements
		_, err := c.entitlementConnector.CreateEntitlement(ctx, entSpec)
		if entitlementExistsError, ok := lo.ErrorsAs[*entitlement.AlreadyExistsError](err); ok {
			// TODO: there might be a cleaner upsert than using the override flow, probably a custom method is needed
			_, err = c.entitlementConnector.OverrideEntitlement(ctx, entitlementExistsError.SubjectKey, entitlementExistsError.EntitlementID, entSpec)
			if err != nil {
				return fmt.Errorf("failed to override entitlement: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("failed to create entitlement: %w", err)
		}
	}
	return nil
}

// Close the entitlements for a rate card returning the closed entitlements.
func (c *connector) closeRateCardEntitlement(ctx context.Context, rateCard SubscriptionRateCard, at time.Time) (Entitlement, error) {
	var def Entitlement
	ent, err := c.subscriptionEntitlementRepo.GetByRateCard(ctx, rateCard.ID)
	if err != nil {
		return def, fmt.Errorf("failed to get entitlements of rate card: %w", err)
	}
	err = c.entitlementConnector.DeleteEntitlement(ctx, ent.Entitlement.Namespace, ent.Entitlement.ID, at)
	// TODO: the returned entitlements should have deletedAt set...
	return ent, err
}

// In some cases we have to migrate entitlement usage from the previous one to the new one.
type entitlementUsageMigratingRateCard struct {
	RateCard
	measureUsageFrom *time.Time
}

var _ RateCard = entitlementUsageMigratingRateCard{}

func (e entitlementUsageMigratingRateCard) GetEntitlementSpec(args ...[]any) (entitlement.CreateEntitlementInputs, error) {
	spec, err := e.RateCard.GetEntitlementSpec()
	if err != nil {
		return spec, err
	}

	if spec.EntitlementType != entitlement.EntitlementTypeMetered {
		return spec, err
	}

	m := entitlement.MeasureUsageFromInput{}

	if e.measureUsageFrom == nil {
		return spec, nil
	}

	err = m.FromTime(*e.measureUsageFrom)
	if err != nil {
		return spec, err
	}

	spec.MeasureUsageFrom = &m
	return spec, nil
}
