# plan

<!-- archie:ai-start -->

> Root domain package for product-catalog Plans: defines Plan/Phase/RateCard/Addon value types, the Repository and Service interfaces, lifecycle events, validators, not-found errors, and custom Plan JSON (de)serialization. DB lives in adapter/, HTTP in httpdriver/, business rules in service/.

## Patterns

**ValidateWith + ValidatorFunc composition** — Plan and Phase implement models.CustomValidator and run Validate() via ValidateWith(ValidatePlanMeta(), ValidatePlanPhases()) — each rule is a models.ValidatorFunc[Plan]. (`func (p Plan) Validate() error { return p.ValidateWith(ValidatePlanMeta(), ValidatePlanPhases()) }`)
**Managed wrappers expose AsProductCatalog* + ManagedFields** — Phase embeds PhaseManagedFields (with PlanID) + productcatalog.Phase; RateCard embeds RateCardManagedFields (with PhaseID); both implement ManagedFields() and AsProductCatalogPhase()/converters. (`func (p Phase) AsProductCatalogPhase() productcatalog.Phase { return p.Phase }`)
**Custom Plan JSON with raw-message rate cards** — serializer.go defines planAlias/phaseAlias/rateCardAlias and marshals each rate card's Price and EntitlementTemplate as json.RawMessage to control nested shape, avoiding recursive MarshalJSON. (`rateCard := rateCardAlias{Type: rc.Type(), Key: rc.Key(), Price: priceJSON, EntitlementTemplate: entitlementTemplateJSON}`)
**Repository embeds entutils.TxCreator; Service adds lifecycle** — Repository is CRUD; Service mirrors it plus PublishPlan/ArchivePlan/NextPlan. OrderBy is a string enum with Values()+Validate(). (`type Service interface { ...CreatePlan; PublishPlan; ArchivePlan; NextPlan }`)
**Field-prefixed phase validation errors** — ValidatePlanPhases wraps each phase error with the phase Key so callers see which phase failed. (`errs = append(errs, fmt.Errorf("invalid plan phase %q: %s", phase.Key, err))`)
**Typed NotFoundError + lifecycle events** — NewNotFoundError builds messages from namespace/id/key/version; events (PlanCreate/Update/Delete/Publish/Archive) use metadata.EntityPlan and session.GetSessionUserID(ctx). (`metadata.ComposeResourcePath(e.Plan.Namespace, metadata.EntityPlan, e.Plan.ID)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `plan.go` | Plan type, ValidateWith, AsProductCatalogPlan, ValidatePlanMeta/Phases | Addons slice is only populated when expanded; phase validation is per-phase, error-prefixed |
| `phase.go` | Phase + PhaseManagedFields + ManagedPhase interface | Managed phase Validate requires PlanID set; Equal handles both Phase and productcatalog.Phase variants |
| `ratecard.go` | Managed RateCard with PhaseID + custom JSON | UnmarshalJSON dispatches on RateCardSerde.Type; missing type yields 'invalid RateCard type' |
| `serializer.go` | Plan MarshalJSON/UnmarshalJSON via aliases + raw messages | Don't call json.Marshal(p) on Plan inside its own MarshalJSON — use planAlias to avoid recursion |
| `service.go` | Service interface + Input structs/Validate | OrderBy enum must stay in sync with Values(); effective-date inputs use 30s timeJitter vs clock.Now() |
| `repository.go` | Repository persistence contract (entutils.TxCreator) | No lifecycle methods here; keep them on Service |
| `event.go` | Plan lifecycle events | Delete event requires Plan.DeletedAt; all events are v1 |
| `assert.go` | Test equality helpers (AssertPlanEqual, AssertPlanPhasesEqual) | Generic over productcatalog.Phase | Phase; key-based map comparison, not slice-order |

## Anti-Patterns

- Implementing Plan.MarshalJSON by calling json.Marshal on Plan directly (infinite recursion) instead of using planAlias
- Adding DB or HTTP code in this root package rather than adapter/ or httpdriver/
- Bypassing ValidateWith/ValidatorFunc composition with ad-hoc inline checks in Plan/Phase Validate
- Constructing a managed Phase/RateCard without PlanID/PhaseID (Validate will reject as 'reference not set')
- Putting PublishPlan/ArchivePlan/NextPlan on Repository instead of Service

## Decisions

- **Rate-card Price and EntitlementTemplate serialize as json.RawMessage** — These are polymorphic (multiple price/template kinds); raw-message passthrough preserves the discriminated shape without the parent serializer needing to know every variant.
- **Managed wrappers (Phase/RateCard) separate identity from catalog value type** — productcatalog.* holds reusable plan-shape logic; this package adds DB-managed PlanID/PhaseID/timestamps and equality without polluting the shared types.

## Example: Composed validation via ValidatorFunc

```
func (p Plan) Validate() error {
	return p.ValidateWith(ValidatePlanMeta(), ValidatePlanPhases())
}

func ValidatePlanPhases() models.ValidatorFunc[Plan] {
	return func(p Plan) error {
		var errs []error
		for _, phase := range p.Phases {
			if err := phase.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("invalid plan phase %q: %s", phase.Key, err))
			}
		}
		return models.NewNillableGenericValidationError(errors.Join(errs...))
	}
}
```

<!-- archie:ai-end -->
