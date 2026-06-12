# hook

<!-- archie:ai-start -->

> Lifecycle hook bridging the entitlement service to credit-grant cleanup: when a metered entitlement is deleted, its owner grants are removed. Wired into the entitlement service via the ServiceHook mechanism and feature-gated by credits.enabled.

## Patterns

**ServiceHook via type alias + NoopHook embed** — Hook types are aliases over the generic models.ServiceHook[entitlement.Entitlement]; the concrete struct embeds models.NoopServiceHook so only the relevant lifecycle method (PreDelete) is overridden and all other hook callbacks default to no-ops. (`type EntitlementHook = models.ServiceHook[entitlement.Entitlement]; type entitlementHook struct { NoopHook; grantRepo grant.Repo }`)
**Constructor returns the alias interface, not the concrete type** — NewEntitlementHook returns EntitlementHook (the ServiceHook alias), keeping the concrete entitlementHook struct unexported. Dependencies are injected as constructor args, not pulled from globals. (`func NewEntitlementHook(grantRepo grant.Repo) EntitlementHook { return &entitlementHook{grantRepo: grantRepo} }`)
**Tolerant PreDelete: nil and non-metered entitlements are skipped** — PreDelete returns nil (success, no-op) when the entitlement is nil or cannot be parsed as a metered entitlement via meteredentitlement.ParseFromGenericEntitlement. Only metered entitlements own grants, so a parse failure is treated as 'nothing to clean up', not an error. (`meteredEnt, err := meteredentitlement.ParseFromGenericEntitlement(ent); if err != nil { return nil }`)
**Grant cleanup keyed by NamespacedID** — Owner-grant deletion is scoped by tenant: it calls grantRepo.DeleteOwnerGrants with models.NamespacedID{Namespace, ID} built from the metered entitlement, preserving multi-tenant isolation. (`return h.grantRepo.DeleteOwnerGrants(ctx, models.NamespacedID{Namespace: meteredEnt.Namespace, ID: meteredEnt.ID})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entitlement_hook.go` | Defines EntitlementHook/NoopHook aliases, the entitlementHook struct, NewEntitlementHook constructor, and the PreDelete implementation that deletes owner grants for a deleted metered entitlement. | PreDelete must stay defensive: keep the nil-entitlement and ParseFromGenericEntitlement error guards returning nil, otherwise deleting non-metered entitlements would fail. Do not import the entitlement service/wiring layer here — depend only on grant.Repo and the entitlement domain types to avoid import cycles. |

## Anti-Patterns

- Implementing the full ServiceHook surface manually instead of embedding NoopHook — drop NoopHook and you must implement every hook callback.
- Returning an error from PreDelete when the entitlement is not metered; non-metered entitlements have no grants and must pass through cleanly.
- Calling DeleteOwnerGrants without a namespace-scoped NamespacedID, which would cross tenant boundaries.
- Injecting concrete services or the registry/wiring layer into the hook instead of the narrow grant.Repo dependency.

## Decisions

- **Grant cleanup lives in a hook rather than inline in the entitlement service.** — Keeps the entitlement service unaware of credits and lets the whole credit stack stay feature-gated by credits.enabled at the wiring layer — when credits are off, the hook is simply not registered.
- **Non-metered entitlements are silently ignored via ParseFromGenericEntitlement.** — Only metered entitlements own credit grants, so the hook is a no-op for other entitlement types and must not block their deletion.

## Example: Delete owner grants when a metered entitlement is removed

```
import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (h *entitlementHook) PreDelete(ctx context.Context, ent *entitlement.Entitlement) error {
	if ent == nil {
		return nil
	}
	meteredEnt, err := meteredentitlement.ParseFromGenericEntitlement(ent)
	if err != nil {
		return nil
// ...
```

<!-- archie:ai-end -->
