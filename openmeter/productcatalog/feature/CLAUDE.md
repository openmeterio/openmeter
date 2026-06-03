# feature

<!-- archie:ai-start -->

> Domain package for feature entities: defines FeatureConnector (service with validation + event publishing), FeatureRepo (repository interface), Feature struct, UnitCost tagged types, MeterGroupByFilters, Watermill events, and feature-meter resolution. Primary constraint: all business logic and validation lives in featureConnector, never in the adapter.

## Patterns

**FeatureConnector over FeatureRepo** — External callers always go through FeatureConnector (meter validation, key uniqueness, event publishing); never call FeatureRepo directly from outside this package. (`feature.NewFeatureConnector(featureRepo, meterService, publisher)`)
**Input validation before repo delegation** — All connector methods validate inputs (meter aggregation, ULID key guard, UnitCost.Validate/ValidateWithMeter) before delegating to featureRepo. (`if _, err := ulid.Parse(feature.Key); err == nil { return Feature{}, models.NewGenericValidationError(fmt.Errorf("Feature key cannot be a valid ULID")) }`)
**Event publishing after successful repo mutation** — Create/Update/ArchiveFeature each publish a typed Watermill event (implementing EventName/EventMetadata) after the repo write succeeds. (`ev := NewFeatureCreateEvent(ctx, &createdFeature); if err := c.publisher.Publish(ctx, ev); err != nil { return createdFeature, fmt.Errorf("failed to publish: %w", err) }`)
**nullable.Nullable[UnitCost] for partial updates** — UpdateFeatureInputs.UnitCost distinguishes unset (validation error), null (clear), and value (update) via IsSpecified/IsNull/Get. (`if input.UnitCost.IsNull() { /* clear */ } else if v, _ := input.UnitCost.Get(); v != nil { /* update */ }`)
**FeatureMeterCollection dual-index resolution** — ResolveFeatureMeters returns ByKey (latest non-archived per key) and ByID (all requested IDs); key lookup resolves to the latest active feature. (`out.ByKey[featureKey] = out.ByID[latestFeat.ID]`)
**UnitCost mutual-exclusion validation** — UnitCost.Validate() enforces that for LLM type each dimension has exactly one of property OR static value; new dimensions require updating both Validate and ValidateWithMeter. (`if u.LLM.ProviderProperty != "" && u.LLM.Provider != "" { errs = append(errs, errors.New("provider and providerProperty are mutually exclusive")) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `connector.go` | featureConnector implements FeatureConnector with all domain validation + event publishing; defines CreateFeatureInputs, UpdateFeatureInputs, ListFeaturesParams. | TODO: refactor to Service pattern — avoid adding more methods without considering that direction. |
| `repository.go` | FeatureRepo interface (CRUD + entutils.TxCreator + entutils.TxUser[FeatureRepo]) and ArchiveFeatureInput. | FeatureRepo embeds TxCreator/TxUser — any new adapter must implement Tx(), WithTx(), Self(). |
| `unitcost.go` | UnitCost, ManualUnitCost, LLMUnitCost; Validate()/ValidateWithMeter(); LLMTokenType constants. | New token types must go in the validTypes map inside Validate(); new dimensions need both Validate and ValidateWithMeter. |
| `featuremeter.go` | ResolveFeatureMeters, FeatureMeterCollection, getLastFeatures, latestEntity generic. | getLastFeatures prefers non-archived but returns the most recently archived if all are archived — callers must handle archived resolution. |
| `event.go` | Watermill create/update/archive event structs implementing EventName() and EventMetadata(). | Event version is hardcoded 'v1' — bump a new constant if payload changes to avoid breaking consumers. |
| `feature.go` | Feature struct, domain errors (FeatureNotFoundError, FeatureInvalidFiltersError, FeatureWithNameAlreadyExistsError, ForbiddenError), MeterGroupByFilters + conversions. | FIXME: prefer models.NewGenericNotFoundError over FeatureNotFoundError in new code. |

## Anti-Patterns

- Calling featureRepo directly from outside the package — go through FeatureConnector.
- Publishing events before the repo write succeeds.
- Setting UnitCost without Validate() and ValidateWithMeter() for LLM type.
- Using context.Background() instead of the passed ctx in connector methods.
- Adding a new UnitCostType without updating the validTypes map and the adapter/driver cases.

## Decisions

- **FeatureConnector wraps FeatureRepo to keep validation and events in the domain layer** — Adapters are pure persistence; validation and Watermill side-effects belong in the connector so they run regardless of adapter.
- **Features support key-based versioning via archive (one active + many archived per key)** — Plans/subscriptions reference features by key; archiving without replacement lets existing references resolve to the most recent version via getLastFeatures.

## Example: Add a new UnitCostType 'fixed_fee' end-to-end

```
// unitcost.go: const UnitCostTypeFixedFee UnitCostType = "fixed_fee"
// add FixedFeeUnitCost struct + UnitCost.FixedFee field + case in UnitCost.Validate()
// adapter/feature.go: add case in CreateFeature query builder and MapFeatureEntity
// driver/parser.go: add case in domainUnitCostToAPI and apiUnitCostToDomain
```

<!-- archie:ai-end -->
