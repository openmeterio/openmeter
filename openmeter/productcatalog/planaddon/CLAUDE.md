# planaddon

<!-- archie:ai-start -->

> Domain package for plan-to-addon assignment (PlanAddon): defines the PlanAddon aggregate, typed errors, domain events, and the Service + Repository interfaces. PlanAddon links a plan.Plan and an addon.Addon with quantity constraints and phase anchoring.

## Patterns

**PlanAddon embeds productcatalog.PlanAddonMeta + plan.Plan + addon.Addon** — PlanAddon = NamespacedID + ManagedModel + PlanAddonMeta + Plan plan.Plan + Addon addon.Addon. AsProductCatalogPlanAddon() converts to the base productcatalog type. (`type PlanAddon struct { models.NamespacedID; models.ManagedModel; productcatalog.PlanAddonMeta; Plan plan.Plan; Addon addon.Addon }`)
**NotFoundError with PlanIDOrKey + AddonIDOrKey params** — NotFoundErrorParams supports namespace, ID, PlanIDOrKey, and AddonIDOrKey fields; IsNotFound() detects via errors.As. (`return nil, planaddon.NewNotFoundError(planaddon.NotFoundErrorParams{Namespace: ns, PlanIDOrKey: planKey, AddonIDOrKey: addonKey})`)
**Domain events carry PlanAddon pointer + UserID from session** — Three events: PlanAddonCreateEvent, PlanAddonUpdateEvent, PlanAddonDeleteEvent. Delete event Validate() asserts PlanAddon.DeletedAt != nil. (`event := planaddon.NewPlanAddonCreateEvent(ctx, created); s.publisher.Publish(ctx, eventbus.SystemTopic, event)`)
**UpdatePlanAddonInput.Equal(PlanAddon) for idempotency detection** — UpdatePlanAddonInput implements models.Equaler[PlanAddon]; service can check if the patch produces any change before calling adapter. (`if input.Equal(*existing) { return existing, nil } // skip adapter call`)
**GetPlanAddonInput supports lookup by assignment ID or plan+addon pair** — If ID is empty, PlanIDOrKey and AddonIDOrKey must both be provided. Same dual-lookup pattern as plan.GetPlanInput. (`if i.ID == '' { if i.PlanIDOrKey == '' || i.AddonIDOrKey == '' { return err } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service interface (ListPlanAddons, CreatePlanAddon, DeletePlanAddon, GetPlanAddon, UpdatePlanAddon) and all input types. | CreatePlanAddonInput requires PlanID and AddonID non-empty; ID-only lookup and plan+addon-pair lookup are both valid in Get/Update/Delete inputs. |
| `planaddon.go` | PlanAddon aggregate with Validate() and AsProductCatalogPlanAddon(). | Validate() calls both Plan.Validate() and Addon.Validate(); ensure Plan and Addon are fully populated before constructing. |
| `errors.go` | NotFoundError and IsNotFound; wraps models.NewGenericNotFoundError. | Error message format is 'plan add-on assignment not found [namespace=... plan.idOrKey=... addon.idOrKey=...]'. |
| `event.go` | Three lifecycle events; subsystem is 'planaddon'. | PlanAddonDeleteEvent.Validate() requires PlanAddon.DeletedAt != nil — soft-delete must precede event construction. |
| `repository.go` | Repository extends entutils.TxCreator; 5-method CRUD interface. | TxCreator is required for TransactingRepo in adapter layer. |
| `assert.go` | Test helpers AssertPlanAddonCreateInputEqual, AssertPlanAddonUpdateInputEqual, AssertPlanAddonEqual. | These are in the planaddon package (not a separate testutils sub-package) so they are importable in adapter/service tests. |

## Anti-Patterns

- Checking plan/addon status (draft, active) inside the adapter — status validation belongs in the service layer.
- Returning raw adapter-level errors without wrapping in planaddon.NewNotFoundError.
- Calling adapter methods outside transaction.Run for write paths — risks partial writes if event publish fails.
- Hard-deleting PlanAddon rows instead of soft-deleting via SetDeletedAt.
- Skipping eager-load of Plan and Addon edges after create/update — PlanAddon.Plan and PlanAddon.Addon must always be fully populated.

## Decisions

- **PlanAddon carries full plan.Plan and addon.Addon embedded rather than just IDs** — Callers (subscription sync, billing) need the full plan and addon data without additional lookups; eager-loading at the adapter layer avoids N+1 queries.
- **Both ID-based and plan+addon pair-based lookups supported in Get/Update/Delete inputs** — HTTP handlers and internal code may have either the assignment ID or only the plan/addon identifiers; supporting both avoids forcing a separate Get before Update.

<!-- archie:ai-end -->
