# hook

<!-- archie:ai-start -->

> Provides lifecycle hook implementations that bridge the credit/grant domain into the entitlement service hook registry. Currently contains a single PreDelete hook that cleans up all grants owned by a metered entitlement before the entitlement is deleted.

## Patterns

**Embed NoopHook for unimplemented lifecycle methods** — Every hook struct must embed models.NoopServiceHook[entitlement.Entitlement] so only the relevant lifecycle methods need to be overridden. This satisfies the full models.ServiceHook interface without boilerplate. (`type entitlementHook struct { NoopHook; grantRepo grant.Repo }`)
**Type-alias EntitlementHook and NoopHook at package boundary** — Package exposes typed aliases (EntitlementHook, NoopHook) that parameterise models.ServiceHook and models.NoopServiceHook with entitlement.Entitlement, keeping callers decoupled from the generic type parameter. (`type EntitlementHook = models.ServiceHook[entitlement.Entitlement]`)
**Guard on nil and type before acting** — PreDelete (and any future hook method) must nil-check the entity and type-assert to the specific sub-type via the domain helper before executing domain logic. Unknown sub-types must return nil (silent no-op), never an error. (`if ent == nil { return nil }; meteredEnt, err := meteredentitlement.ParseFromGenericEntitlement(ent); if err != nil { return nil }`)
**Constructor returns the interface type, not the concrete struct** — NewEntitlementHook returns EntitlementHook (interface alias), not *entitlementHook. Constructors in this package must always return the interface so callers cannot depend on the concrete type. (`func NewEntitlementHook(grantRepo grant.Repo) EntitlementHook { return &entitlementHook{...} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entitlement_hook.go` | Sole file in the package; defines the EntitlementHook type alias, NoopHook alias, the concrete entitlementHook struct, and its constructor. Wired into entitlement.Service via RegisterHooks at startup. | Adding hook methods that call grantRepo without nil-checking the entity or without type-asserting to meteredentitlement first will panic or corrupt non-metered entitlement deletes. |

## Anti-Patterns

- Returning an error for an unrecognised entitlement sub-type — unknown sub-types must silently return nil
- Adding business logic that reaches outside credit/grant (e.g. calling balance or engine packages directly) — this hook owns only grant cleanup
- Implementing PostCreate/PostUpdate without also embedding NoopHook — always embed NoopHook to satisfy the full interface
- Registering this hook against a non-metered service hook registry — the hook is parameterised on entitlement.Entitlement and must only be registered via entitlement.Service.RegisterHooks

## Decisions

- **Use models.NoopServiceHook embedding instead of a full interface implementation** — The hook only needs PreDelete; embedding NoopHook provides all other methods as no-ops and prevents future interface additions from breaking the build.
- **Type assertions to meteredentitlement rather than a switch on entitlement type field** — ParseFromGenericEntitlement encapsulates the sub-type detection logic in the metered package, keeping the hook free of entitlement kind constants and resilient to new entitlement types.

## Example: Adding a new lifecycle method (e.g. PostCreate) to clean up on creation failure

```
import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (h *entitlementHook) PostCreate(ctx context.Context, ent *entitlement.Entitlement) error {
	if ent == nil {
		return nil
	}
	meteredEnt, err := meteredentitlement.ParseFromGenericEntitlement(ent)
	if err != nil {
		return nil // not a metered entitlement — silent no-op
// ...
```

<!-- archie:ai-end -->
