# feature

<!-- archie:ai-start -->

> Domain package for feature entities: defines FeatureConnector (service), FeatureRepo (repository interface), Feature struct, input/output types, UnitCost types, MeterGroupByFilters, Watermill events, and FeatureMeters resolution logic.

## Patterns

**FeatureConnector over FeatureRepo** — Business logic (meter validation, key uniqueness check, event publishing) lives in featureConnector which wraps FeatureRepo. Never call FeatureRepo directly from outside this package; always go through FeatureConnector. (`feature.NewFeatureConnector(featureRepo, meterService, publisher)`)
**Input validation before repo call** — All connector methods validate inputs (e.g. meter aggregation check, key-is-ULID guard, UnitCost.Validate(), UnitCost.ValidateWithMeter()) before delegating to featureRepo. (`if _, err := ulid.Parse(feature.Key); err == nil { return Feature{}, models.NewGenericValidationError(...) }`)
**Event publishing after successful mutation** — CreateFeature, UpdateFeature, ArchiveFeature each publish a typed event (FeatureCreateEvent, FeatureUpdateEvent, FeatureArchiveEvent) after the repo write. Events implement EventName() and EventMetadata() for Watermill. (`c.publisher.Publish(ctx, NewFeatureCreateEvent(ctx, &createdFeature))`)
**nullable.Nullable[UnitCost] for partial updates** — UpdateFeatureInputs.UnitCost uses oapi-codegen/nullable.Nullable to distinguish unset (validation error), null (clear), and value (update). Check IsSpecified/IsNull/Get accordingly. (`if input.UnitCost.IsNull() { /* clear */ } else if input.UnitCost.IsSpecified() { v, _ := input.UnitCost.Get() }`)
**FeatureMeters dual-index resolution** — ResolveFeatureMeters returns FeatureMeterCollection with ByKey (latest non-archived per key) and ByID (all requested IDs). Key lookup resolves to the latest active feature; explicit IDs remain addressable even if archived. (`out.ByKey[featureKey] = out.ByID[latestFeat.ID]`)
**UnitCost mutual-exclusion validation** — UnitCost.Validate() enforces that for LLM type, each dimension (provider, model, token_type) has exactly one of property OR static value. Adding a new dimension requires adding it to both Validate() and ValidateWithMeter(). (`if u.LLM.ProviderProperty != "" && u.LLM.Provider != "" { errs = append(errs, errors.New("mutually exclusive")) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `connector.go` | featureConnector implements FeatureConnector; contains all domain validation logic and event publishing. Also defines input types CreateFeatureInputs, UpdateFeatureInputs, ListFeaturesParams. | TODO comment says refactor to Service pattern — do not add more methods to FeatureConnector without following that future refactor direction. |
| `repository.go` | Defines FeatureRepo interface (CRUD + entutils.TxCreator + entutils.TxUser[FeatureRepo]) and ArchiveFeatureInput. | FeatureRepo embeds TxCreator and TxUser — any new adapter must implement Tx(), WithTx(), Self(). |
| `unitcost.go` | UnitCost, ManualUnitCost, LLMUnitCost types; Validate() and ValidateWithMeter(); LLMTokenType constants. | New token types must be added to the validTypes map inside Validate(). |
| `featuremeter.go` | ResolveFeatureMeters, FeatureMeterCollection, resolveFeatureMeters, getLastFeatures, latestEntity generic. | getLastFeatures prefers non-archived; if all archived, returns most recently archived — callers must handle archived features being resolved. |
| `event.go` | Watermill event structs for create/update/archive; each implements EventName() (subsystem + name + version) and EventMetadata(). | Event version is hardcoded as v1 — bump version as a new constant if the event payload changes. |
| `feature.go` | Feature struct, domain error types (FeatureNotFoundError, FeatureInvalidFiltersError, FeatureWithNameAlreadyExistsError, ForbiddenError), MeterGroupByFilters with Validate and conversion helpers. | FIXME note: use models.NewGenericNotFoundError instead of FeatureNotFoundError in new code. |

## Anti-Patterns

- Calling featureRepo directly from outside the feature package — always go through FeatureConnector.
- Setting UnitCost without calling Validate() and ValidateWithMeter() for LLM type.
- Publishing events before the repo write succeeds — always publish after.
- Using context.Background() instead of the passed ctx in connector methods.
- Adding a new UnitCostType without updating the validTypes map in Validate() and adding cases in adapter MapFeatureEntity.

## Decisions

- **FeatureConnector wraps FeatureRepo to keep validation and event publishing in the domain layer, not the adapter** — Adapters are pure persistence; validation (meter aggregation, ULID key guard, unit cost) and side-effects (events) belong in the connector so they run regardless of which adapter is used.
- **Features support key-based versioning via archive: same key can have multiple archived versions plus one active** — Plans and subscriptions reference features by key; archiving without replacement allows existing references to resolve to the most-recent version via getLastFeatures.

## Example: Add a new UnitCostType 'fixed_fee' with its own sub-struct

```
// unitcost.go — add type constant
const UnitCostTypeFixedFee UnitCostType = "fixed_fee"

// Add FixedFeeUnitCost struct and populate UnitCost.FixedFee field
// Add case in UnitCost.Validate()
// Add case in adapter/feature.go CreateFeature query builder and MapFeatureEntity
// Add case in driver/parser.go domainUnitCostToAPI and apiUnitCostToDomain
```

<!-- archie:ai-end -->
