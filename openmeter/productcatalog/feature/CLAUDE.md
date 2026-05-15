# feature

<!-- archie:ai-start -->

> Domain package for feature entities: defines FeatureConnector (service with validation and event publishing), FeatureRepo (repository interface), Feature struct, UnitCost tagged types, MeterGroupByFilters, Watermill events, and FeatureMeters resolution. Primary constraint: all business logic and validation must live in featureConnector, never in the adapter.

## Patterns

**FeatureConnector over FeatureRepo** — External callers always go through FeatureConnector (meter validation, key uniqueness, event publishing). Never call FeatureRepo directly from outside this package. (`feature.NewFeatureConnector(featureRepo, meterService, publisher)`)
**Input validation before repo delegation** — All connector methods validate inputs (meter aggregation check, ULID key guard, UnitCost.Validate(), UnitCost.ValidateWithMeter()) before delegating to featureRepo. (`if _, err := ulid.Parse(feature.Key); err == nil { return Feature{}, models.NewGenericValidationError(fmt.Errorf("Feature key cannot be a valid ULID")) }`)
**Event publishing after successful repo mutation** — CreateFeature, UpdateFeature, and ArchiveFeature each publish a typed event (FeatureCreateEvent, FeatureUpdateEvent, FeatureArchiveEvent) after the repo write succeeds. Events implement EventName() and EventMetadata() for Watermill. (`featureCreatedEvent := NewFeatureCreateEvent(ctx, &createdFeature); if err := c.publisher.Publish(ctx, featureCreatedEvent); err != nil { return createdFeature, fmt.Errorf("failed to publish: %w", err) }`)
**nullable.Nullable[UnitCost] for partial updates** — UpdateFeatureInputs.UnitCost uses oapi-codegen/nullable.Nullable to distinguish unset (validation error), null (clear), and value (update). Check IsSpecified/IsNull/Get accordingly. (`if !i.UnitCost.IsSpecified() { errs = append(errs, errors.New("unitCost is required")) }
if input.UnitCost.IsNull() { /* clear */ } else if v, _ := input.UnitCost.Get(); v != nil { /* update */ }`)
**FeatureMeterCollection dual-index resolution** — ResolveFeatureMeters returns ByKey (latest non-archived per key) and ByID (all requested IDs). Key lookup always resolves to the latest active feature; explicit IDs remain addressable even if archived. (`out.ByKey[featureKey] = out.ByID[latestFeat.ID]`)
**UnitCost mutual-exclusion validation** — UnitCost.Validate() enforces that for LLM type, each dimension (provider, model, token_type) has exactly one of property OR static value — never both. Adding a new dimension requires updating both Validate() and ValidateWithMeter(). (`if u.LLM.ProviderProperty != "" && u.LLM.Provider != "" { errs = append(errs, errors.New("provider and providerProperty are mutually exclusive")) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `connector.go` | featureConnector implements FeatureConnector with all domain validation logic and event publishing; also defines input types CreateFeatureInputs, UpdateFeatureInputs, ListFeaturesParams. | TODO comment says refactor to Service pattern — do not add more methods to FeatureConnector without considering that future refactor direction. |
| `repository.go` | Defines FeatureRepo interface (CRUD + entutils.TxCreator + entutils.TxUser[FeatureRepo]) and ArchiveFeatureInput. | FeatureRepo embeds TxCreator and TxUser — any new adapter must implement Tx(), WithTx(), Self(). |
| `unitcost.go` | UnitCost, ManualUnitCost, LLMUnitCost types; Validate() and ValidateWithMeter(); LLMTokenType constants. | New token types must be added to the validTypes map inside Validate(); new dimensions require updating both Validate and ValidateWithMeter. |
| `featuremeter.go` | ResolveFeatureMeters, FeatureMeterCollection, resolveFeatureMeters, getLastFeatures, latestEntity generic. | getLastFeatures prefers non-archived; if all are archived, returns the most recently archived — callers must handle archived features being resolved. |
| `event.go` | Watermill event structs for create/update/archive; each implements EventName() (subsystem + name + version) and EventMetadata(). | Event version is hardcoded as 'v1' — bump version as a new constant if the event payload changes to avoid breaking consumers. |
| `feature.go` | Feature struct, domain error types (FeatureNotFoundError, FeatureInvalidFiltersError, FeatureWithNameAlreadyExistsError, ForbiddenError), MeterGroupByFilters with Validate and conversion helpers. | FIXME note: use models.NewGenericNotFoundError instead of FeatureNotFoundError in new code. |

## Anti-Patterns

- Calling featureRepo directly from outside the feature package — always go through FeatureConnector.
- Publishing events before the repo write succeeds — always publish after successful persistence.
- Setting UnitCost without calling Validate() and ValidateWithMeter() for LLM type.
- Using context.Background() instead of the passed ctx in connector methods.
- Adding a new UnitCostType without updating the validTypes map in Validate() and adding cases in adapter MapFeatureEntity and driver parser.

## Decisions

- **FeatureConnector wraps FeatureRepo to keep validation and event publishing in the domain layer** — Adapters are pure persistence; validation (meter aggregation, ULID key guard, unit cost mutual exclusion) and side-effects (Watermill events) belong in the connector so they run regardless of which adapter is used.
- **Features support key-based versioning via archive: same key can have multiple archived versions plus one active** — Plans and subscriptions reference features by key; archiving without replacement allows existing references to resolve to the most recent version via getLastFeatures.

## Example: Add a new UnitCostType 'fixed_fee' end-to-end

```
// unitcost.go: const UnitCostTypeFixedFee UnitCostType = "fixed_fee"
// Add FixedFeeUnitCost struct and UnitCost.FixedFee field
// Add case in UnitCost.Validate()
// adapter/feature.go: add case in CreateFeature query builder and MapFeatureEntity
// driver/parser.go: add case in domainUnitCostToAPI and apiUnitCostToDomain
```

<!-- archie:ai-end -->
