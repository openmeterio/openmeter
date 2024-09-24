package subscription

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/samber/lo"
)

func (c *connector) createDependentsOfRateCard(ctx context.Context, rateCard RateCard) error {
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
