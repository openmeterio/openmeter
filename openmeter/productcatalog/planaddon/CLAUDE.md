# planaddon

<!-- archie:ai-start -->

> Domain package for plan-to-addon assignment (PlanAddon): defines the PlanAddon aggregate (linking plan.Plan and addon.Addon with quantity/phase constraints), typed errors, domain events, and the Service + Repository interfaces.

## Patterns

**PlanAddon embeds productcatalog.PlanAddonMeta + full plan.Plan + addon.Addon** — PlanAddon = NamespacedID + ManagedModel + PlanAddonMeta + Plan plan.Plan + Addon addon.Addon. AsProductCatalogPlanAddon() converts to the base productcatalog type. Always ensure Plan and Addon are fully populated before constructing. (`type PlanAddon struct { models.NamespacedID; models.ManagedModel; productcatalog.PlanAddonMeta; Plan plan.Plan; Addon addon.Addon }`)
**NotFoundError with PlanIDOrKey + AddonIDOrKey params** — NotFoundErrorParams supports namespace, ID, PlanIDOrKey, and AddonIDOrKey fields. Use IsNotFound() for errors.As detection. Error message format: 'plan add-on assignment not found [namespace=... plan.idOrKey=... addon.idOrKey=...]'. (`return nil, planaddon.NewNotFoundError(planaddon.NotFoundErrorParams{Namespace: ns, PlanIDOrKey: planKey, AddonIDOrKey: addonKey})`)
**Domain events carry PlanAddon pointer + UserID from session** — Three events: PlanAddonCreateEvent, PlanAddonUpdateEvent, PlanAddonDeleteEvent. Delete event Validate() asserts PlanAddon.DeletedAt != nil — soft-delete must be applied before constructing the event. (`event := planaddon.NewPlanAddonCreateEvent(ctx, created); s.publisher.Publish(ctx, event)`)
**UpdatePlanAddonInput.Equal(PlanAddon) for idempotency detection** — UpdatePlanAddonInput implements models.Equaler[PlanAddon]. Service can check if the patch produces any change before calling the adapter, enabling idempotent update handling. (`if input.Equal(*existing) { return existing, nil } // skip adapter call`)
**GetPlanAddonInput dual-lookup: assignment ID or plan+addon pair** — If ID is empty, PlanIDOrKey and AddonIDOrKey must both be provided. Same dual-lookup pattern as plan.GetPlanInput. Adapters must handle both lookup paths. (`if i.ID == "" { if i.PlanIDOrKey == "" || i.AddonIDOrKey == "" { return err } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service interface (ListPlanAddons, CreatePlanAddon, DeletePlanAddon, GetPlanAddon, UpdatePlanAddon) and all input types. | CreatePlanAddonInput requires PlanID and AddonID non-empty; Get/Update/Delete support both ID-only and plan+addon-pair lookup. |
| `planaddon.go` | PlanAddon aggregate with Validate() and AsProductCatalogPlanAddon(). | Validate() calls both Plan.Validate() and Addon.Validate(); ensure both are fully populated before constructing. |
| `errors.go` | NotFoundError and IsNotFound; wraps models.NewGenericNotFoundError. | Error message format includes 'plan add-on assignment not found' — differs from plan/addon error messages. |
| `event.go` | Three lifecycle events; subsystem is 'planaddon'. | PlanAddonDeleteEvent.Validate() requires PlanAddon.DeletedAt != nil — soft-delete must precede event construction. |
| `repository.go` | Repository extends entutils.TxCreator; 5-method CRUD interface. | TxCreator is required for TransactingRepo in adapter layer. |
| `assert.go` | Test helpers AssertPlanAddonCreateInputEqual, AssertPlanAddonUpdateInputEqual, AssertPlanAddonEqual in the planaddon package (not testutils). | These are in the main package so they are importable in adapter/service tests without a separate testutils import. |

## Anti-Patterns

- Checking plan/addon status (draft, active) inside the adapter — status validation belongs in the service layer.
- Returning raw adapter-level errors without wrapping in planaddon.NewNotFoundError.
- Calling adapter methods outside transaction.Run for write paths — risks partial writes if event publish fails.
- Hard-deleting PlanAddon rows instead of soft-deleting via SetDeletedAt.
- Skipping eager-load of Plan and Addon edges after create/update — PlanAddon.Plan and PlanAddon.Addon must always be fully populated.

## Decisions

- **PlanAddon carries full plan.Plan and addon.Addon embedded rather than just IDs** — Callers (subscription sync, billing) need the full plan and addon data without additional lookups; eager-loading at the adapter layer avoids N+1 queries.
- **Both ID-based and plan+addon pair-based lookups supported in Get/Update/Delete inputs** — HTTP handlers and internal code may have either the assignment ID or only the plan/addon identifiers; supporting both avoids forcing a separate Get before Update.

## Example: Service write operation with transaction.Run, cross-entity status validation, and event publish

```
import (
    "github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
    "github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) CreatePlanAddon(ctx context.Context, params planaddon.CreatePlanAddonInput) (*planaddon.PlanAddon, error) {
    if err := params.Validate(); err != nil { return nil, err }
    // cross-entity checks: plan must be active, addon must be active
    return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*planaddon.PlanAddon, error) {
        created, err := s.adapter.CreatePlanAddon(ctx, params)
        if err != nil { return nil, err }
        event := planaddon.NewPlanAddonCreateEvent(ctx, created)
        if err := s.publisher.Publish(ctx, event); err != nil {
            return nil, fmt.Errorf("publish planaddon created: %w", err)
        }
// ...
```

<!-- archie:ai-end -->
