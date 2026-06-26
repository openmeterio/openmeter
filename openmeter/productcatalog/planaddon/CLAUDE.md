# planaddon

<!-- archie:ai-start -->

> Root domain package for the PlanAddon join entity (a plan↔addon assignment): defines the PlanAddon type embedding full plan.Plan and addon.Addon, the Repository/Service interfaces, lifecycle events, not-found errors, and input structs. DB in adapter/, HTTP in httpdriver/, business rules in service/.

## Patterns

**PlanAddon embeds full plan.Plan and addon.Addon** — The assignment carries fully-loaded Plan and Addon domain objects (not just IDs); Validate() validates NamespacedID, ManagedModel, Plan, Addon, and AsProductCatalogPlanAddon() in turn. (`type PlanAddon struct { models.NamespacedID; models.ManagedModel; productcatalog.PlanAddonMeta; Plan plan.Plan; Addon addon.Addon }`)
**AsProductCatalogPlanAddon converter** — Maps to productcatalog.PlanAddon by calling Plan.AsProductCatalogPlan() and Addon.AsProductCatalogAddon() so shared catalog validation can run. (`return productcatalog.PlanAddon{PlanAddonMeta: a.PlanAddonMeta, Plan: a.Plan.AsProductCatalogPlan(), Addon: a.Addon.AsProductCatalogAddon()}`)
**ID-or-(plan,addon) dual identification on inputs** — Get/Update/Delete inputs accept either an assignment ID or a (PlanID/IDOrKey, AddonID/IDOrKey) pair; Validate requires both halves of the pair when ID is empty. (`if i.ID == "" { if i.PlanID == "" { errs = append(errs, errors.New("plan id must be provided if assignment id is not provided")) } ... }`)
**Repository embeds entutils.TxCreator; Service mirrors it** — Both interfaces expose List/Create/Delete/Get/Update; no extra lifecycle ops (publish/archive don't exist for assignments). (`type Repository interface { entutils.TxCreator; ListPlanAddons(...); CreatePlanAddon(...); ... }`)
**Equaler with pointer-aware MaxQuantity comparison** — UpdatePlanAddonInput.Equal compares pointer fields (MaxQuantity, FromPlanPhase) carefully, treating nil/non-nil mismatches as not-equal. (`if i.MaxQuantity == nil && p.MaxQuantity != nil { return false }`)
**Events use EntityAddon resource path** — Create/Update/Delete events compose Source/Subject via metadata.EntityAddon (not a dedicated planaddon entity) and pull session.GetSessionUserID(ctx). (`metadata.ComposeResourcePath(e.PlanAddon.Namespace, metadata.EntityAddon, e.PlanAddon.ID)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `planaddon.go` | PlanAddon type, Validate, AsProductCatalogPlanAddon | Validate cascades into embedded Plan and Addon — both must be fully loaded or validation fails |
| `service.go` | Service interface + Input structs (Create/Update/Get/Delete/List) | ListPlanAddonsInput.Validate returns nil (no validation); Update/Get/Delete enforce ID-or-pair rule |
| `repository.go` | Repository persistence contract | Embeds entutils.TxCreator; only CRUD, no lifecycle methods |
| `event.go` | PlanAddonCreate/Update/Delete events (v1) | Delete event requires DeletedAt set; events use EntityAddon, not a planaddon entity |
| `errors.go` | Typed NotFoundError keyed by ns/id/plan.idOrKey/addon.idOrKey | Adapter must wrap ent not-found into this type |

## Anti-Patterns

- Storing only Plan/Addon IDs on PlanAddon instead of the full embedded plan.Plan/addon.Addon the type expects
- Adding DB or HTTP code in this root package rather than adapter/ or httpdriver/
- Comparing MaxQuantity/FromPlanPhase without nil-vs-non-nil pointer checks in Equal
- Requiring an assignment ID when callers may legitimately identify by (plan, addon) pair
- Returning a raw ent not-found instead of planaddon.NewNotFoundError

## Decisions

- **PlanAddon embeds the full Plan and Addon aggregates** — Compatibility validation (currency, phases, rate cards) needs both complete sides; the adapter eager-loads them so the service can validate the assignment as a whole.
- **Inputs support ID-or-(plan,addon) identification** — API callers reference assignments either by their own ID or by the natural (plan key, addon key) pair; both must resolve to the same row.

## Example: Validating an assignment by cascading into both sides

```
func (a PlanAddon) Validate() error {
	var errs []error
	if err := a.NamespacedID.Validate(); err != nil { errs = append(errs, err) }
	if err := a.ManagedModel.Validate(); err != nil { errs = append(errs, err) }
	if err := a.Plan.Validate(); err != nil { errs = append(errs, err) }
	if err := a.Addon.Validate(); err != nil { errs = append(errs, err) }
	if err := a.AsProductCatalogPlanAddon().Validate(); err != nil { errs = append(errs, err) }
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
```

<!-- archie:ai-end -->
