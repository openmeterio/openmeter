# hook

<!-- archie:ai-start -->

> Provides lifecycle hook implementations bridging the credit/grant domain into the entitlement service hook registry. Currently owns a single PreDelete hook that deletes all grants for a metered entitlement before the entitlement itself is removed.

## Patterns

**Embed NoopHook for unimplemented lifecycle methods** — Every hook struct must embed models.NoopServiceHook[entitlement.Entitlement] so only the relevant lifecycle methods need overriding. This satisfies the full models.ServiceHook interface without boilerplate. (`type entitlementHook struct { NoopHook; grantRepo grant.Repo }`)
**Type-alias hook interfaces at package boundary** — Export typed aliases (EntitlementHook, NoopHook) that parameterise the generic models types with entitlement.Entitlement, keeping callers decoupled from the type parameter. (`type EntitlementHook = models.ServiceHook[entitlement.Entitlement]`)
**Nil-check then type-assert before acting** — Every hook method must nil-check the entity first, then type-assert to the specific sub-type via the domain helper. Unknown sub-types return nil (silent no-op), never an error. (`if ent == nil { return nil }; meteredEnt, err := meteredentitlement.ParseFromGenericEntitlement(ent); if err != nil { return nil }`)
**Constructor returns the interface, not the concrete struct** — NewEntitlementHook must return EntitlementHook (the interface alias), not *entitlementHook, so callers cannot depend on the concrete type. (`func NewEntitlementHook(grantRepo grant.Repo) EntitlementHook { return &entitlementHook{grantRepo: grantRepo} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entitlement_hook.go` | Sole file in the package. Defines EntitlementHook and NoopHook type aliases, the concrete entitlementHook struct, its constructor, and the PreDelete implementation. Wired into entitlement.Service via RegisterHooks in app/common. | Adding hook methods that call grantRepo without nil-checking the entity or without type-asserting to meteredentitlement first — this will panic or corrupt non-metered entitlement deletes. |

## Anti-Patterns

- Returning an error for an unrecognised entitlement sub-type — unknown sub-types must silently return nil
- Adding business logic that reaches outside credit/grant (e.g. calling balance or engine packages) — this hook owns only grant cleanup
- Implementing PostCreate/PostUpdate without embedding NoopHook — always embed to satisfy the full interface
- Registering this hook against a non-entitlement service hook registry — it is parameterised on entitlement.Entitlement only

## Decisions

- **Embed models.NoopServiceHook instead of implementing the full interface** — The hook only needs PreDelete; embedding NoopHook provides all other methods as no-ops and prevents future interface additions from breaking the build.
- **Type-assert via meteredentitlement.ParseFromGenericEntitlement rather than switching on an entitlement kind constant** — ParseFromGenericEntitlement encapsulates sub-type detection in the metered package, keeping the hook free of kind constants and resilient to new entitlement types.

## Example: Adding a new lifecycle method (e.g. PostCreate) that targets metered entitlements only

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
