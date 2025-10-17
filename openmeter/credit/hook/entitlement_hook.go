package hook

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	EntitlementHook = models.ServiceHook[entitlement.Entitlement]
	NoopHook        = models.NoopServiceHook[entitlement.Entitlement]
)

type entitlementHook struct {
	NoopHook

	grantRepo grant.Repo
}

func NewEntitlementHook(
	grantRepo grant.Repo,
) EntitlementHook {
	return &entitlementHook{
		grantRepo: grantRepo,
	}
}

func (h *entitlementHook) PreDelete(ctx context.Context, ent *entitlement.Entitlement) error {
	if ent == nil {
		return nil
	}

	meteredEnt, err := meteredentitlement.ParseFromGenericEntitlement(ent)
	if err != nil {
		return nil
	}

	return h.grantRepo.DeleteOwnerGrants(ctx, models.NamespacedID{Namespace: meteredEnt.Namespace, ID: meteredEnt.ID})
}
