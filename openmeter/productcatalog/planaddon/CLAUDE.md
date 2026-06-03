# planaddon

<!-- archie:ai-start -->

> Domain package for plan-to-addon assignment (PlanAddon): defines the PlanAddon aggregate linking a full plan.Plan and addon.Addon with quantity/phase constraints, typed errors, domain events, and the Service + Repository interfaces; children split into adapter (Ent), httpdriver (v1 HTTP), and service (cross-entity validation + events).

## Patterns

**PlanAddon embeds full plan.Plan + addon.Addon** — PlanAddon = NamespacedID + ManagedModel + PlanAddonMeta + Plan plan.Plan + Addon addon.Addon; AsProductCatalogPlanAddon() converts to the base type. Always populate Plan and Addon fully before constructing. (`type PlanAddon struct { models.NamespacedID; models.ManagedModel; productcatalog.PlanAddonMeta; Plan plan.Plan; Addon addon.Addon }`)
**Dual-lookup inputs (ID or plan+addon pair)** — Get/Update/Delete inputs support either an assignment ID or a PlanIDOrKey + AddonIDOrKey pair; if ID is empty both pair fields must be provided. Adapters must handle both paths. (`if i.ID == "" { if i.PlanIDOrKey == "" || i.AddonIDOrKey == "" { return err } }`)
**Typed NotFoundError with pair params** — NewNotFoundError accepts namespace, ID, PlanIDOrKey, AddonIDOrKey; use IsNotFound() for errors.As detection. Message format differs from plan/addon ('plan add-on assignment not found ...'). (`return nil, planaddon.NewNotFoundError(planaddon.NotFoundErrorParams{Namespace: ns, PlanIDOrKey: planKey, AddonIDOrKey: addonKey})`)
**Equaler-based idempotent updates** — UpdatePlanAddonInput implements models.Equaler[PlanAddon] so the service can skip the adapter call when the patch produces no change. (`if input.Equal(*existing) { return existing, nil }`)
**Events carry PlanAddon pointer + UserID, published in tx** — Three events (Create/Update/Delete) carry the PlanAddon and UserID from session; DeleteEvent.Validate() asserts DeletedAt != nil. Publish inside transaction.Run after every mutation. (`event := planaddon.NewPlanAddonCreateEvent(ctx, created); s.publisher.Publish(ctx, event)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service interface (List/Create/Delete/Get/Update) and input types. | CreatePlanAddonInput requires non-empty PlanID and AddonID; Get/Update/Delete support both ID-only and plan+addon-pair lookup. |
| `planaddon.go` | PlanAddon aggregate with Validate() and AsProductCatalogPlanAddon(). | Validate() calls both Plan.Validate() and Addon.Validate() — ensure both are fully populated before constructing. |
| `errors.go` | NotFoundError and IsNotFound wrapping models.NewGenericNotFoundError. | Message format includes 'plan add-on assignment not found' — differs from plan/addon error messages. |
| `event.go` | Three lifecycle events; subsystem is 'planaddon'. | PlanAddonDeleteEvent.Validate() requires PlanAddon.DeletedAt != nil — soft-delete must precede event construction. |
| `repository.go` | Repository extends entutils.TxCreator; 5-method CRUD interface. | TxCreator is required for TransactingRepo in the adapter layer. |
| `assert.go` | Test helpers (AssertPlanAddon*Equal) in the main package, not testutils. | Kept in the main package so adapter/service tests can import them without a separate testutils import. |

## Anti-Patterns

- Checking plan/addon status (draft, active) inside the adapter — status validation belongs in the service layer.
- Returning raw adapter-level errors without wrapping in planaddon.NewNotFoundError.
- Calling adapter methods outside transaction.Run for writes — risks partial writes if event publish fails.
- Hard-deleting PlanAddon rows instead of soft-deleting via SetDeletedAt.
- Skipping eager-load of Plan and Addon edges after create/update — both must always be fully populated.

## Decisions

- **PlanAddon embeds full plan.Plan and addon.Addon rather than just IDs.** — Callers (subscription sync, billing) need the full plan and addon data without extra lookups; eager-loading at the adapter avoids N+1 queries.
- **Both ID-based and plan+addon-pair lookups supported in Get/Update/Delete.** — HTTP handlers and internal code may have either the assignment ID or only the plan/addon identifiers; supporting both avoids forcing a Get before Update.

## Example: Service write with cross-entity validation and event publish in a transaction

```
import (
    "github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
    "github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) CreatePlanAddon(ctx context.Context, params planaddon.CreatePlanAddonInput) (*planaddon.PlanAddon, error) {
    if err := params.Validate(); err != nil { return nil, err }
    return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*planaddon.PlanAddon, error) {
        created, err := s.adapter.CreatePlanAddon(ctx, params)
        if err != nil { return nil, err }
        if err := s.publisher.Publish(ctx, planaddon.NewPlanAddonCreateEvent(ctx, created)); err != nil {
            return nil, fmt.Errorf("publish planaddon created: %w", err)
        }
        return created, nil
    })
// ...
```

<!-- archie:ai-end -->
