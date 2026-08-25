package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	db_entitlement "github.com/openmeterio/openmeter/openmeter/ent/db/entitlement"
	"github.com/openmeterio/openmeter/pkg/models"
)

// ensureEntitlementOwnerExists prevents credit state from being attached to an
// entitlement in another namespace. Credit's owner model is generic, while its
// persisted owner foreign key currently targets entitlements.
func ensureEntitlementOwnerExists(ctx context.Context, dbClient *db.Client, owner models.NamespacedID) error {
	exists, err := dbClient.Entitlement.Query().
		Where(
			db_entitlement.ID(owner.ID),
			db_entitlement.Namespace(owner.Namespace),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("failed to resolve credit owner: %w", err)
	}

	if !exists {
		return grant.NewOwnerNotFoundError(owner, "entitlement")
	}

	return nil
}
