# features

<!-- archie:ai-start -->

> TypeSpec definitions for the Features domain: feature models (meter-backed with optional unit costs), cost query types, and CRUD+query operations compiled to v3 OpenAPI and Go/JS/Python SDKs.

## Patterns

**Namespace isolation** — All types and interfaces are declared inside `namespace Features;`. Every .tsp file opens with this namespace declaration to keep generated names scoped. (`namespace Features;
@friendlyName("Feature") model Feature { ... }`)
**Discriminated union for polymorphic types** — Use `@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })` on unions (e.g. FeatureUnitCost) so the generated OpenAPI uses a `type` field as discriminator without extra envelope wrapping. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union FeatureUnitCost { manual: FeatureManualUnitCost, llm: FeatureLlmUnitCost }`)
**Visibility annotations control lifecycle access** — Use `@visibility(Lifecycle.Read, Lifecycle.Create)` etc. on fields to control which HTTP operations expose or accept a field. Read-only computed fields use `@visibility(Lifecycle.Read)` only. (`@visibility(Lifecycle.Read) pricing?: FeatureLlmUnitCostPricing;`)
**Extension markers for private/unstable/internal endpoints** — Operations that are not yet public must carry all three extensions: `@extension(Shared.PrivateExtension, true)`, `@extension(Shared.UnstableExtension, true)`, `@extension(Shared.InternalExtension, true)`. (`@extension(Shared.PrivateExtension, true)
@extension(Shared.UnstableExtension, true)
@extension(Shared.InternalExtension, true)
@get list(...)`)
**Shared generic response wrappers** — Operations return `Shared.PagePaginatedResponse<T>`, `Shared.CreateResponse<T>`, `Shared.UpdateResponse<T>`, `Shared.DeleteResponse` — never inline response shapes. (`list(...): Shared.PagePaginatedResponse<Feature> | Common.ErrorResponses;`)
**index.tsp as the re-export barrel** — Each sub-folder has an index.tsp that imports all sibling .tsp files and re-opens the namespace so callers only need `import "./features/index.tsp"`. (`// index.tsp
import "./cost.tsp";
import "./unitcost.tsp";
import "./operations.tsp";
import "./feature.tsp";
namespace Features;`)
**Suppress linter rules with inline justification** — When a field name pattern legitimately violates the `@openmeter/api-spec-aip` linter (e.g. nullable fields, repeated-prefix grouping), suppress with `#suppress` and a quoted reason string. (`#suppress "@openmeter/api-spec-aip/no-nullable" "cost is nullable"
cost: Shared.Numeric | null;`)

## Key Files

| File             | Role                                                                                                                                | Watch For                                                                                                                                                                |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------- |
| `operations.tsp` | Declares the FeatureOperations and FeatureCostOperations interfaces with all HTTP verbs, route params, and response types.          | Must import @typespec/http and use `using TypeSpec.Http;` — omitting this breaks @query/@get/@post decorators. deepObject+explode style required for filter params.      |
| `unitcost.tsp`   | Defines FeatureUnitCost discriminated union (manual/llm), enums, and resolved pricing model.                                        | Mutually exclusive fields (e.g. provider vs provider_property) are enforced by docs only — no TypeSpec constraint; add #suppress if linter flags repeated-prefix naming. |
| `feature.tsp`    | Core Feature model using Shared.ResourceWithKey spread and optional meter + unit_cost fields.                                       | FeatureUpdateRequest only includes updatable fields (unit_cost); do not add read-only fields to it.                                                                      |
| `cost.tsp`       | FeatureCostQueryRow and FeatureCostQueryResult models for the cost query endpoint; reuses Meters.MeterQueryRequest as request body. | cost field is nullable (Shared.Numeric                                                                                                                                   | null) — requires #suppress on no-nullable rule. |

## Anti-Patterns

- Declaring HTTP routes directly in model files — routes belong in operations.tsp only
- Defining response envelope shapes inline instead of using Shared.CreateResponse<T>/PagePaginatedResponse<T>
- Adding @visibility(Lifecycle.Create) to computed/server-set fields (e.g. id, created_at, status)
- Omitting all three extension markers on internal-only operations
- Editing generated api/v3/api.gen.go instead of regenerating after TypeSpec changes

## Decisions

- **LLM unit cost uses a flat model with mutually exclusive static/property fields rather than a nested union** — Keeps the API surface flat and JSON-friendly; TypeSpec discriminated unions can't enforce mutual exclusion so the constraint is documented and validated at the Go service layer.
- **FeatureCostQueryRow.cost is nullable with explicit #suppress** — Cost can be unresolvable when LLM pricing data is unavailable; null is the correct semantic rather than omitting the field, even though it violates the no-nullable linter rule.

## Example: Adding a new operation to FeatureOperations in operations.tsp

```
// operations.tsp
import "@typespec/http";
import "@typespec/openapi";
import "./feature.tsp";
using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Features;

interface FeatureOperations {
  @patch
  @route("/{featureId}")
  @operationId("update-feature")
  @summary("Update a feature by id.")
  @extension(Shared.PrivateExtension, true)
// ...
```

<!-- archie:ai-end -->
