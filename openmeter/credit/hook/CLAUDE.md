# hook

<!-- archie:ai-start -->

> Lifecycle hook implementations bridging the credit/grant domain into the entitlement service hook registry. Currently owns a single PreDelete hook that deletes all grants for a metered entitlement before the entitlement is removed.

## Patterns

**Embed NoopHook for unimplemented lifecycle methods** — Every hook struct embeds models.NoopServiceHook[entitlement.Entitlement] (aliased NoopHook) so only the relevant lifecycle methods need overriding, satisfying the full interface without boilerplate. (`type entitlementHook struct { NoopHook; grantRepo grant.Repo }`)
**Type-alias hook interfaces at package boundary** — Export typed aliases (EntitlementHook, NoopHook) parameterising the generic models types with entitlement.Entitlement, decoupling callers from the type parameter. (`type EntitlementHook = models.ServiceHook[entitlement.Entitlement]`)
**Nil-check then type-assert before acting** — Every hook method nil-checks the entity, then type-asserts via meteredentitlement.ParseFromGenericEntitlement; unknown sub-types return nil (silent no-op), never an error. (`if ent == nil { return nil }; meteredEnt, err := meteredentitlement.ParseFromGenericEntitlement(ent); if err != nil { return nil }`)
**Constructor returns the interface, not the concrete struct** — NewEntitlementHook returns EntitlementHook (the interface alias), not *entitlementHook, so callers cannot depend on the concrete type. (`func NewEntitlementHook(grantRepo grant.Repo) EntitlementHook { return &entitlementHook{grantRepo: grantRepo} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entitlement_hook.go` | Sole file: EntitlementHook/NoopHook aliases, the concrete entitlementHook struct, its constructor, and PreDelete. Wired into entitlement.Service via RegisterHooks in app/common. | Adding hook methods that call grantRepo without nil-checking the entity or type-asserting to meteredentitlement first will panic or corrupt non-metered entitlement deletes. |

## Anti-Patterns

- Returning an error for an unrecognised entitlement sub-type — unknown sub-types must silently return nil
- Adding logic reaching outside credit/grant (calling balance or engine packages) — this hook owns only grant cleanup
- Implementing PostCreate/PostUpdate without embedding NoopHook
- Registering this hook against a non-entitlement service hook registry — it is parameterised on entitlement.Entitlement only

## Decisions

- **Embed models.NoopServiceHook instead of implementing the full interface** — The hook only needs PreDelete; embedding NoopHook provides all other methods as no-ops and keeps future interface additions from breaking the build.
- **Type-assert via meteredentitlement.ParseFromGenericEntitlement rather than switching on a kind constant** — ParseFromGenericEntitlement encapsulates sub-type detection in the metered package, keeping the hook free of kind constants and resilient to new entitlement types.

## Example: Add a new lifecycle method targeting metered entitlements only

```
func (h *entitlementHook) PostCreate(ctx context.Context, ent *entitlement.Entitlement) error {
	if ent == nil {
		return nil
	}
	meteredEnt, err := meteredentitlement.ParseFromGenericEntitlement(ent)
	if err != nil {
		return nil // not a metered entitlement — silent no-op
	}
	_ = meteredEnt
	return nil
}
```

<!-- archie:ai-end -->
