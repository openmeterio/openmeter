# info

<!-- archie:ai-start -->

> TypeSpec definitions for static v1 lookup endpoints: the supported-currencies list and async-operation progress tracking. A small self-contained read-only sub-package with no mutable resources and no pagination.

## Patterns

**Flat model for static lookup entities** — Currency and Progress are plain models with no Resource spread (no id/timestamps) because they are not persisted lifecycle entities. listCurrencies returns a flat Currency[] array, never PaginatedResponse. (`@get @operationId("listCurrencies") listCurrencies(): Currency[] | CommonErrors;`)
**routes.tsp holds interfaces, model file holds the model** — CurrenciesEndpoints lives in routes.tsp; the Currency model lives in currencies.tsp; ProgressEndpoints and Progress co-locate in progress.tsp. main.tsp only imports the three files. (`// currencies.tsp: model Currency only; routes.tsp: interface CurrenciesEndpoints`)
**Progress id is a plain string, not ULID** — getProgress takes id: string (not ULID) to match the progressmanager domain identifiers. (`@get @route("/{id}") getProgress(id: string): Progress | NotFoundError | CommonErrors;`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `routes.tsp` | CurrenciesEndpoints interface at /api/v1/info/currencies; single GET returning Currency[]. | Returns a plain array, not PaginatedResponse — do not add pagination here without a spec change. |
| `progress.tsp` | ProgressEndpoints interface (GET by id) and the Progress model (success/failed/total/updatedAt). | id is a plain string, not ULID — matches the progressmanager domain. |
| `currencies.tsp` | Currency model with code (CurrencyCode), name, symbol, subunits (uint32). | subunits is uint32; zero-subunit currencies (JPY) are valid. |
| `main.tsp` | Import manifest only — imports currencies.tsp, progress.tsp, routes.tsp. | New .tsp files must be added here to be compiled. |

## Anti-Patterns

- Adding a paginated response to listCurrencies — it returns all currencies as a flat array.
- Adding mutating (POST/PUT/DELETE) operations — this folder is read-only lookup data.
- Typing the Progress id as ULID instead of string.

## Decisions

- **Currency and Progress are flat models with no Resource/timestamp spread.** — They are static lookup/transient data, not persisted lifecycle entities, so id and timestamps are unnecessary.

## Example: Static lookup list endpoint returning a flat array (not paginated)

```
@route("/api/v1/info/currencies")
@tag("Lookup Information")
interface CurrenciesEndpoints {
  @get @operationId("listCurrencies") listCurrencies(): Currency[] | CommonErrors;
}
```

<!-- archie:ai-end -->
