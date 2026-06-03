# features

<!-- archie:ai-start -->

> TypeSpec definitions for the v3 Features domain: meter-backed feature models with optional unit costs, cost-query types, and CRUD+query operations compiled to OpenAPI and Go/JS/Python SDKs. All types live under namespace Features.

## Patterns

**Namespace isolation** — Every .tsp file opens with namespace Features; so generated names stay scoped. (`namespace Features;
@friendlyName("Feature") model Feature { ... }`)
**Discriminated union for polymorphic types** — Unions use @discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) so the OpenAPI discriminator is a top-level type field with no envelope wrapping (FeatureUnitCost: manual | llm). (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) union FeatureUnitCost { manual: FeatureManualUnitCost, llm: FeatureLlmUnitCost }`)
**Visibility annotations control lifecycle access** — Fields use @visibility(Lifecycle.Read, Lifecycle.Create, ...) to control HTTP exposure. Computed read-only fields (resolved pricing) use @visibility(Lifecycle.Read) only. (`@visibility(Lifecycle.Read) pricing?: FeatureLlmUnitCostPricing;`)
**Extension markers for unstable/internal endpoints** — Non-public operations carry extension markers. All current feature operations carry @extension(Shared.UnstableExtension, true). (`@extension(Shared.UnstableExtension, true) @get list(...)`)
**Shared generic response wrappers** — Operations return Shared.PagePaginatedResponse<T>, Shared.CreateResponse<T>, Shared.UpdateResponse<T>, Shared.GetResponse<T>, Shared.DeleteResponse — never inline response shapes. (`list(...): Shared.PagePaginatedResponse<Feature> | Common.ErrorResponses;`)
**Suppress linter rules with inline justification** — When a name/shape legitimately violates @openmeter/api-spec-aip (nullable, repeated-prefix grouping, shared-model doc), suppress with #suppress and a quoted reason. (`#suppress "@openmeter/api-spec-aip/no-nullable" "cost is nullable" cost: Shared.Numeric | null;`)
**deepObject+explode for filter query params** — List filter params use @query(#{ style: "deepObject", explode: true }) with a dedicated *ParamsFilter model of Common.*FieldFilter fields. (`@query(#{ style: "deepObject", explode: true }) filter?: ListFeaturesParamsFilter,`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `operations.tsp` | FeatureOperations (list/create/get/update(PATCH)/delete) and FeatureCostOperations (POST /query) interfaces with route params, filters, and response types. | Must import @typespec/http and use `using TypeSpec.Http;`. queryCost reuses Meters.MeterQueryRequest as the body. |
| `unitcost.tsp` | FeatureUnitCost discriminated union (manual/llm), FeatureUnitCostType / FeatureLlmTokenType enums, and FeatureLlmUnitCostPricing. | Mutually exclusive fields (provider vs provider_property, model vs model_property, token_type vs token_type_property) are docs-only — no TypeSpec constraint; add #suppress for repeated-prefix-grouping where flagged. |
| `feature.tsp` | Feature model (spreads Shared.ResourceWithKey, optional meter FeatureMeterReference + unit_cost) and FeatureUpdateRequest. | FeatureUpdateRequest includes only updatable fields (unit_cost, nullable to clear); do not add read-only fields like id or created_at. |
| `cost.tsp` | FeatureCostQueryRow and FeatureCostQueryResult; reuses Meters.MeterQueryRequest as request body. | cost is Shared.Numeric | null and requires the no-nullable #suppress; the detail field explains why cost is null. |
| `index.tsp` | Barrel importing cost/unitcost/operations/feature and reopening namespace Features. | New .tsp files must be added here. |

## Anti-Patterns

- Declaring HTTP routes directly in model files — routes belong in operations.tsp only.
- Defining response envelope shapes inline instead of using Shared.CreateResponse<T>/PagePaginatedResponse<T>.
- Adding @visibility(Lifecycle.Create) to computed/server-set fields (id, created_at, status, resolved pricing).
- Omitting the appropriate extension markers on internal-only operations.
- Editing generated api/v3/api.gen.go instead of regenerating after TypeSpec changes.

## Decisions

- **LLM unit cost is a flat model with mutually exclusive static/property fields, not a nested union.** — Keeps the API surface flat and JSON-friendly; TypeSpec unions cannot enforce mutual exclusion, so the constraint is documented and validated in the Go service layer.
- **FeatureCostQueryRow.cost is nullable with explicit #suppress.** — Cost can be unresolvable when LLM pricing data is unavailable; null is the correct semantic rather than omitting the field, even though it violates no-nullable.

## Example: Adding an operation to FeatureOperations in operations.tsp

```
import "@typespec/http";
using TypeSpec.Http;
namespace Features;
interface FeatureOperations {
  @patch @route("/{featureId}") @operationId("update-feature")
  @extension(Shared.UnstableExtension, true)
  update(@path featureId: Shared.ULID, @body feature: FeatureUpdateRequest): Shared.UpdateResponse<Feature> | Common.ErrorResponses | Common.NotFound;
}
```

<!-- archie:ai-end -->
