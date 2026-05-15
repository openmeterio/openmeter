# llmcost

<!-- archie:ai-start -->

> TypeSpec definitions for the LLM cost pricing domain: global synced price models, per-namespace override CRUD, and filter/pagination operations compiled to v3 OpenAPI. Two separate interfaces split global read-only prices from namespace-scoped override management.

## Patterns

**Split interfaces by resource boundary** — Separate LLMCostPricesOperations (global read-only prices) from LLMCostOverridesOperations (namespace-scoped CRUD) into distinct interfaces so Go generates separate handler groups. (`interface LLMCostPricesOperations { list_prices(...); get_price(...); }
interface LLMCostOverridesOperations { list_overrides(...); create_override(...); delete_override(...); }`)
**deepObject filter params with explode** — All filter parameters use @query(#{ style: "deepObject", explode: true }) so filter[field][op]=value query strings deserialize correctly. (`@query(#{ style: "deepObject", explode: true })
filter?: ListPricesParamsFilter,`)
**Read-only visibility for all Price response fields** — All Price model fields use @visibility(Lifecycle.Read) since prices are system-managed. OverrideCreate is a separate flat input model for writes — never mix read and write fields on the same model. (`@visibility(Lifecycle.Read)
source: PriceSource;`)
**Enum for controlled string values** — Use enum (not union) for closed string sets with known members like PriceSource. Use union only when open-ended or when needing @summary per variant. (`enum PriceSource { Manual: "manual", System: "system" }`)
**Suppress repeated-prefix linter with justification** — Fields like model_id/model_name, cache_read_per_token/cache_write_per_token, and effective_from/effective_to trigger the repeated-prefix-grouping linter; suppress with a specific reason string. (`#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "model_id and model_name should not be grouped"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `operations.tsp` | Declares LLMCostPricesOperations and LLMCostOverridesOperations interfaces with all HTTP endpoints and ListPricesParamsFilter filter model. | Must import @typespec/http and use `using TypeSpec.Http;`. Sort parameter is documented on list_prices but not list_overrides — keep consistent when extending. All operations require UnstableExtension and InternalExtension markers. |
| `prices.tsp` | Defines Price, ModelPricing, Provider, Model, OverrideCreate models and PriceSource enum. | OverrideCreate uses model_id (plain string) not a ULID — override creation is upsert semantics (unique per provider+model+currency), not idempotent by ID. effective_from is required on OverrideCreate. |
| `index.tsp` | Barrel: imports prices.tsp then operations.tsp in that order (models before operations). | Order matters — operations.tsp imports types from prices.tsp; the index must maintain this import order. |

## Anti-Patterns

- Merging global price operations and override operations into a single interface
- Using @visibility(Lifecycle.Create/Update) on Price model fields — prices are read-only responses; OverrideCreate is the separate write input model
- Omitting effective_from on OverrideCreate — it is required to establish when the override takes effect
- Using a ULID for OverrideCreate identification — overrides are upserted by provider+model+currency composite key, not by ID

## Decisions

- **OverrideCreate is a flat input model separate from the Price response model** — The write input (provider string, model_id, pricing) is structurally different from the full Price response which includes system-managed fields (id, source, created_at). Sharing one model would require complex visibility gymnastics.
- **Two interfaces (Prices vs Overrides) rather than one combined interface** — Global prices are read-only and system-synced; per-namespace overrides are user-managed CRUD. Keeping them separate makes handler responsibility boundaries clear and allows independent authorization policies.

## Example: Add a new filter field to ListPricesParamsFilter

```
// operations.tsp
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "model_id and model_name should not be grouped"
@friendlyName("ListLLMCostPricesParamsFilter")
model ListPricesParamsFilter {
  provider?: Common.StringFieldFilter;
  model_id?: Common.StringFieldFilter;
  model_name?: Common.StringFieldFilter;
  currency?: Common.StringFieldFilter;
  source?: Common.StringFieldFilter;

  // New: filter by effective date
  effective_from?: Common.DateTimeFieldFilter;
}
```

<!-- archie:ai-end -->
