# info

<!-- archie:ai-start -->

> TypeSpec definitions for static lookup endpoints: supported currencies list and async operation progress tracking. Small self-contained sub-package; no mutable resources, no pagination on currencies.

## Patterns

**Flat model for static lookup entities** — Currency and Progress have no Resource spread (no id/timestamps) because they are not persisted domain entities with lifecycle. Currency returns a plain array, not a paginated response. (`interface CurrenciesEndpoints { @get listCurrencies(): Currency[] | CommonErrors; }`)
**routes.tsp holds interface definitions, currencies.tsp holds models** — Model types (Currency, Progress) live in their own files; routing interfaces live in routes.tsp or the model file. main.tsp imports all and re-exports. (`// routes.tsp imports ../rest.tsp and defines CurrenciesEndpoints
// currencies.tsp defines the Currency model only`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `routes.tsp` | CurrenciesEndpoints interface at /api/v1/info/currencies. Single list endpoint returning Currency[]. | Returns a plain array, not PaginatedResponse — do not add pagination here without a spec change. |
| `progress.tsp` | ProgressEndpoints interface and Progress model. GET by string id, not ULID. | Progress id is a plain `string`, not `ULID` — matches the progressmanager domain. |
| `currencies.tsp` | Currency model with code, name, symbol, subunits. No routes here. | subunits is uint32; zero-subunit currencies (JPY) are valid. |

## Anti-Patterns

- Adding paginated responses to the currencies list endpoint — it returns all currencies as a flat array.
- Adding mutable (POST/PUT/DELETE) operations — this folder is read-only lookup data.

<!-- archie:ai-end -->
