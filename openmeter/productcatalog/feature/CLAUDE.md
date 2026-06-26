# feature

<!-- archie:ai-start -->

> Core feature domain: the Feature aggregate plus FeatureConnector (business logic) and FeatureRepo (persistence contract). Highest-fanin sub-package — billing, entitlement, subscription, and notification all depend on it.

## Patterns

**Connector validates, repo persists** — featureConnector wraps FeatureRepo + meterService + eventbus.Publisher. All validation (meter aggregation, unit cost, key-not-ULID, key uniqueness) happens in the connector before calling featureRepo. (`func NewFeatureConnector(featureRepo FeatureRepo, meterService meterpkg.Service, publisher eventbus.Publisher) FeatureConnector`)
**Publish domain event after every mutation** — Create/Update/Archive publish NewFeatureCreateEvent/UpdateEvent/ArchiveEvent via publisher.Publish; event names live in event.go (feature.created/updated/archived, v1). (`c.publisher.Publish(ctx, NewFeatureCreateEvent(ctx, &createdFeature))`)
**Validate() collects errors via NewNillableGenericValidationError** — Input structs (UpdateFeatureInputs, ListFeaturesParams, ArchiveFeatureInput, FeatureOrderBy) gather into []error and return models.NewNillableGenericValidationError(errors.Join(...)). (`return models.NewNillableGenericValidationError(errors.Join(errs...))`)
**Typed domain errors** — feature.go defines FeatureNotFoundError, FeatureWithNameAlreadyExistsError, FeatureInvalidMeterAggregationError, FeatureInvalidFiltersError, ForbiddenError — return these (HTTP layer maps them). (`return Feature{}, &FeatureWithNameAlreadyExistsError{Name: feature.Key, ID: found.ID}`)
**Latest-by-key feature resolution** — ResolveFeatureMeters/getLastFeatures resolve a key to the latest (preferring unarchived, else most-recently-archived) feature while keeping explicit IDs addressable; uses the lastEntityAccessor[Feature] abstraction. (`func getLastFeatures(features []Feature) map[string]Feature { return getLastEntity(features, featureAccessor{}) }`)
**UnitCost is a typed sum (manual | llm) with two-stage validation** — UnitCost.Validate() enforces type/field exclusivity; ValidateWithMeter(meter) additionally checks LLM property names exist in meter.GroupBy. LLM cost requires an associated meter. (`if err := feature.UnitCost.ValidateWithMeter(*resolvedMeter); err != nil { ... }`)
**MeterGroupByFilters legacy<->filter conversions** — ConvertMapStringToMeterGroupByFilters / ConvertMeterGroupByFiltersToMapString bridge the legacy map[string]string and filter.FilterString forms; the latter returns nil unless all filters are pure Eq. (`result[k] = filter.FilterString{Eq: &v}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `connector.go` | FeatureConnector interface + featureConnector business logic and input/param structs. | TODO marks this for migration to the service pattern; CreateFeature normalizes MeterID via the meter service and rejects ULID-shaped keys; validMeterAggregations whitelist gates feature creation. |
| `feature.go` | Feature aggregate, Validate(), typed errors, MeterGroupByFilters. | MeterSlug is deprecated (v1 only) — prefer MeterID; Feature.Validate requires CreatedAt/UpdatedAt non-zero. |
| `repository.go` | FeatureRepo interface incl. entutils.TxCreator/TxUser. | Adding a repo method requires implementing it in productcatalog/adapter and keeping it tx-aware. |
| `unitcost.go` | UnitCost/ManualUnitCost/LLMUnitCost types, Validate, ValidateWithMeter, LLMTokenType enum. | Each LLM dimension is exactly-one-of property-or-static-value (mutually exclusive); token_type must be in the valid set (input/output/cache_read/cache_write/reasoning/request/response). |
| `featuremeter.go` | FeatureMeter/FeatureMeters collection + ResolveFeatureMeters (latest-by-key resolution joined with meters). | Resolution lists with IncludeArchived=true then picks the latest; requireMeter=true errors if a feature has no meter. |
| `event.go` | feature.created/updated/archived events with EventName/EventMetadata/Validate. | ArchiveEvent.Validate requires ArchivedAt set; EventMetadata.Time differs per event (CreatedAt/UpdatedAt/ArchivedAt). |

## Anti-Patterns

- Mutating features through FeatureRepo directly, bypassing the connector's validation and event publishing
- Returning generic errors where a typed feature.*Error exists (breaks HTTP status mapping)
- Allowing a ULID-shaped feature Key (explicitly rejected to keep ID/key slots distinguishable)
- Configuring LLM UnitCost without an associated meter or with both property and static value set
- Treating MeterSlug as authoritative instead of MeterID

## Decisions

- **Features are archived (ArchivedAt), never versioned or hard-deleted** — Features are referenced by ID with no publish-new-version action, so active plan/subscription references brick changes either way; archiving keeps history addressable by ID while key resolves to latest.
- **Key resolution prefers unarchived then most-recently-archived** — Lets callers reference a feature by stable key while old versions stay reachable by explicit ID (see featuremeter_test cases).
- **FeatureConnector kept as a connector, not yet a service** — Explicit TODO to refactor to the standard service pattern; new code should not entrench the connector shape further.

## Example: Validating and creating a feature with event publication

```
// in featureConnector.CreateFeature
if _, err := ulid.Parse(feature.Key); err == nil {
  return Feature{}, models.NewGenericValidationError(fmt.Errorf("Feature key cannot be a valid ULID"))
}
createdFeature, err := c.featureRepo.CreateFeature(ctx, feature)
if err != nil { return Feature{}, err }
if err := c.publisher.Publish(ctx, NewFeatureCreateEvent(ctx, &createdFeature)); err != nil {
  return createdFeature, fmt.Errorf("failed to publish feature created event: %w", err)
}
```

<!-- archie:ai-end -->
