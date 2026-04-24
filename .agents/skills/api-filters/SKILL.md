---
name: api-filters
description: Add, modify, or convert AIP-style query-parameter filters on v3 list endpoints. Use when adding filterable fields to a list API, wiring filter parsing into a handler, converting API filters into pkg/filter predicates, or debugging filter parsing/validation behavior.
user-invocable: false
argument-hint: "[resource or list endpoint to add filters to]"
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

# v3 API Filter Parsing

You are helping the user add or modify AIP-style query-parameter filters on an OpenMeter v3 list endpoint.

OpenMeter follows the **Kong AIP filter spec** (NOT Google AIP-160 expression syntax). Filters use the deepObject query-parameter encoding `?filter[field][op]=value`. The implementation is split across three layers:

- `api/v3/filters/` — API-layer filter types, `Parse` entry point, and `FromAPI*` converters
- `pkg/filter/` — internal predicate model with `Validate()`, `Select(field)`, `ApplyToQuery(...)` helpers
- domain service input structs — hold the already-converted `*filter.*` predicates

## Relationship to other skills

Filtering straddles two layers that the repo skill set keeps separate:

- **TypeSpec / OAS side** — `Common.*FieldFilter` types, `Shared.ResourceFilters`, `deepObject` exposure, label dot-notation. See `../api/rules/aip-160-filtering.md` (the canonical Kong AIP-160 rule for OpenMeter). Use the `/api` skill when you also need to scaffold or modify the TypeSpec operation itself.
- **Go implementation side** — what this skill covers: `api/v3/filters.Parse`, the API-layer filter structs, `FromAPI*` helpers, service input wiring, adapter `filter.ApplyToQuery`, gotchas.

If you are adding a brand-new filterable endpoint, invoke `/api` first to wire up the TypeSpec + handler shell, then come back here for the conversion + adapter code. If you are only adding/modifying filters on an existing endpoint, this skill is enough on its own.

## Context

- **API-layer package:** `api/v3/filters/` — API-shaped filter structs and `FromAPI*` converters
- **Internal predicate model:** `pkg/filter/` — implements the `Filter` interface (`Validate`, `Select`, `IsEmpty`, …); used by Ent query builders
- **Reference implementation in use:** `api/v3/handlers/customers/list.go` (handler) + `openmeter/customer/adapter/customer.go` (adapter) + `openmeter/customer/customer.go` (service input struct)
- **Kong AIP spec for filtering:** `../api/rules/aip-160-filtering.md`

## Architecture: three-layer conversion

```
TypeSpec Common.*FieldFilter
        │  (make gen-api)
        ▼
api.Filter* (generated OAS types)        ─┐
        │                                  │ handler decode layer
api/v3/filters.Filter*  (API-layer types)  │  calls filters.FromAPIFilter*(...)
        │                                  │
pkg/filter.Filter*      (predicate model) ─┘  stored on service input struct
        │
        ▼                                    adapter layer
filter.ApplyToQuery(query, input.Field, dbField)
```

Rules:

- The **handler** converts `params.Filter.X` (API-shaped) → `*filter.X` (predicate) using `filters.FromAPIFilter*`.
- The **service input struct** holds `*filter.FilterString`, `*filter.FilterTime`, `*filter.FilterULID`, etc. — NOT the API-layer types.
- The **adapter** calls `filter.ApplyToQuery(query, input.Field, dbField)` to attach the predicate to the Ent query.
- `filters.Parse` is called by the **generated** deepObject binding layer in `api/v3/api.gen.go`, not by handlers. Handlers receive `params.Filter` already populated.

## Filter Grammar (Kong AIP)

**Encoding:** deepObject query parameters. Two-level brackets identify field and operator:

```
filter[field]=value              # shorthand → eq
filter[field][eq]=value          # exact match
filter[field][neq]=value         # not equal (also returns NULLs)
filter[field][contains]=value    # substring match (case-insensitive on strings)
filter[field][oeq]=a,b,c         # one-of-equal (comma-separated, max 50 items)
filter[field][ocontains]=a,b     # one-of-contains
filter[field][gt]=value          # greater than
filter[field][gte]=value         # greater than or equal
filter[field][lt]=value          # less than
filter[field][lte]=value         # less than or equal
filter[field]                    # bare key → exists=true (presence check)
filter[field][exists]            # explicit existence check
filter[field][nexists]           # absence check (only for additionalProperties maps like labels)
filter[labels.key_1][eq]=val     # dot-notation: only the FIRST dot is a delimiter
```

Operator constants live in `api/v3/filters/parse.go` as `OpEq`, `OpNeq`, `OpGt`, `OpGte`, `OpLt`, `OpLte`, `OpContains`, `OpOeq`, `OpOcontains`, `OpExists`, `OpNexists`.

### API-layer filter types (`api/v3/filters/filter.go`)

| Go type             | Fields                                                                         |
| ------------------- | ------------------------------------------------------------------------------ |
| `FilterBoolean`     | `Eq`                                                                           |
| `FilterNumeric`     | `Eq`, `Neq`, `Oeq`, `Gt`, `Gte`, `Lt`, `Lte`                                   |
| `FilterDateTime`    | `Eq`, `Gt`, `Gte`, `Lt`, `Lte` (all `*time.Time`; no `Neq`/`Oeq`)              |
| `FilterString`      | `Eq`, `Neq`, `Gt`, `Gte`, `Lt`, `Lte`, `Contains`, `Oeq`, `Ocontains`, `Exists` |
| `FilterULID`        | `Eq`, `Neq`, `Contains`, `Oeq`, `Ocontains`, `Exists` (no range ops)           |
| `FilterStringExact` | `Eq`, `Neq`, `Oeq` (no `Exists`, no `Contains`)                                |
| `FilterLabel`       | `Eq`, `Neq`, `Contains`, `Oeq`, `Ocontains` (label map value predicates)       |
| `FilterLabels`      | type alias: `map[string]FilterLabel`                                           |

The `Exists` field is serialized under the JSON key `$exists` so it does not collide with a literal `exists` operator in flattened encodings.

**Important:** the API-layer types do NOT have `Validate()` methods. Validation (mutual exclusivity, complexity bounds, format checks) happens on the internal `pkg/filter.*` predicates — typically from the service input struct's own `Validate()`, calling `f.Validate()` on each non-nil filter.

### `pkg/filter` predicates

| Predicate                | Produced by converter         | Notes                            |
| ------------------------ | ----------------------------- | -------------------------------- |
| `*filter.FilterString`   | `FromAPIFilterString`         | Also used by `FromAPIFilterLabel`, `FromAPIFilterStringExact` |
| `*filter.FilterULID`     | `FromAPIFilterULID`           | Embeds `FilterString`            |
| `*filter.FilterFloat`    | `FromAPIFilterNumeric`        | (note: not `FilterNumeric`)      |
| `*filter.FilterTime`     | `FromAPIFilterDateTime`       | RFC-3339 already parsed to `time.Time` by `Parse` |
| `*filter.FilterBoolean`  | `FromAPIFilterBoolean`        |                                  |
| `map[string]filter.FilterString` | `FromAPIFilterLabels` | Label map flatten                |

The `Filter` interface (`pkg/filter/filter.go:19`) exposes `Validate()`, `ValidateWithComplexity(maxDepth int)`, `Select(field string) func(*sql.Selector)`, `SelectWhereExpr(...)`, and `IsEmpty()`.

### Multi-filter semantics

- Multiple `filter[...]` parameters with **different fields** combine with **AND**.
- A single field with `oeq` / `ocontains` combines its values with **OR** (`IN (...)` or `OR ILIKE ...`).
- A single field with multiple operators (e.g. both `gte` and `lte`) is wrapped by the converter into `And{...}` of single-operator `pkg/filter` nodes.
- The bare-key existence shortcut maps to `IS NOT NULL`; `nexists` only works on schemaless maps (`labels`, `metadata`).

### Validation is done by `pkg/filter`

Mutual-exclusivity and format rules (e.g. "multiple operators on one node", ULID format, complexity depth) are enforced by `*filter.FilterX.Validate()` — not by the API-layer types. A typical service input `Validate()` looks like:

```go
if i.Key != nil {
    if err := i.Key.Validate(); err != nil {
        errs = append(errs, models.NewGenericValidationError(fmt.Errorf("invalid key filter: %w", err)))
    }
}
```

### Hard limits (security, `api/v3/filters/parse.go:16-19`)

- **1024 bytes** per single value (`maxFilterValueLength`)
- **50 items** per comma-separated list (`maxCommaSeparatedItems`)
- **Repeated query params for the same key are rejected** (e.g., `?filter[f][eq]=a&filter[f][eq]=b`)
- **Unknown filter fields are rejected** before any other validation (`checkUnknownFilterKeys`)

## Workflow

Follow these steps in order. Use the `/api` skill alongside this one when you also need to touch TypeSpec.

### Step 1: Define the filterable fields in TypeSpec

In `api/spec/packages/aip/src/<domain>/operations.tsp`, define a named filter model for the list operation and expose it as `filter` with `style: "deepObject", explode: true`. Use the `Common.*FieldFilter` types from `common/parameters.tsp` — **do not hand-roll filter models**.

The canonical rule for *which* `Common.*FieldFilter` type to pick, the `Shared.ResourceFilters` spread, label dot-notation, and OAS documentation requirements is `../api/rules/aip-160-filtering.md`. That rule also includes the TypeSpec type ↔ Go `filters.Filter*` mapping. Read it once before picking types — this skill is not the source of truth for the TypeSpec side.

The events list endpoint (`api/spec/packages/aip/src/events/operations.tsp`) and the customer list endpoint are the canonical worked examples.

After editing TypeSpec, run `make gen-api` so the generated `params.Filter` struct in `api/v3/api.gen.go` picks up the new fields.

### Step 2: Store `pkg/filter` predicates on the service input struct

In your domain service input type, add fields typed as **pkg/filter predicates**, not API-layer types. Example from `openmeter/customer/customer.go:296`:

```go
type ListCustomersInput struct {
    Namespace string
    pagination.Page

    OrderBy string
    Order   sortx.Order

    Key          *filter.FilterString
    Name         *filter.FilterString
    PrimaryEmail *filter.FilterString
    // ...
}

func (i ListCustomersInput) Validate() error {
    var errs []error
    // ...
    if i.Key != nil {
        if err := i.Key.Validate(); err != nil {
            errs = append(errs, models.NewGenericValidationError(fmt.Errorf("invalid key filter: %w", err)))
        }
    }
    // ...
    return models.NewNillableGenericValidationError(errors.Join(errs...))
}
```

Pick the narrowest predicate: `filter.FilterString` for strings, `filter.FilterULID` for ULID columns, `filter.FilterFloat` for numbers, `filter.FilterTime` for timestamps, `filter.FilterBoolean` for bools.

### Step 3: Convert API filters in the HTTP handler

In the handler decoder (the first argument to `httptransport.NewHandlerWithArgs`), call the matching `filters.FromAPIFilter*` helper against the generated `params.Filter.<field>` and assign to the request. The canonical pattern is in `api/v3/handlers/customers/list.go`:

```go
import (
    "github.com/openmeterio/openmeter/api/v3/apierrors"
    "github.com/openmeterio/openmeter/api/v3/filters"
)

if params.Filter != nil {
    key, err := filters.FromAPIFilterString(params.Filter.Key)
    if err != nil {
        return ListCustomersRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
            {Field: "filter[key]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
        })
    }
    req.Key = key

    name, err := filters.FromAPIFilterString(params.Filter.Name)
    if err != nil {
        return ListCustomersRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
            {Field: "filter[name]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
        })
    }
    req.Name = name
}
```

Notes:

- Handlers do **not** call `filters.Parse` directly — the generated OAS binding layer does that and surfaces any parse/validation errors as `InvalidParamFormatError` before the handler runs.
- Every `FromAPIFilter*` returns `(*filter.X, error)`. The error channel is reserved for helpers that can fail (e.g. future format checks); today most helpers only return `(nil, nil)` on a nil input, but always handle the error for forward-compatibility.
- On error, wrap with `apierrors.NewBadRequestError(...)` using `Source: apierrors.InvalidParamSourceQuery` and `Field: "filter[<field>]"`.

### Step 4: Apply to the query in the adapter

Adapters use `filter.ApplyToQuery(query, input.Field, dbField)` — a generic helper that:

1. Returns the query unchanged when the predicate is nil.
2. Builds an Ent predicate via `pkg/filter.SelectPredicate[P](...)`.
3. Calls `q.Where(*p)` when the predicate is non-empty.

From `openmeter/customer/adapter/customer.go:52`:

```go
query = filter.ApplyToQuery(query, input.Key, customerdb.FieldKey)
query = filter.ApplyToQuery(query, input.Name, customerdb.FieldName)
query = filter.ApplyToQuery(query, input.PrimaryEmail, customerdb.FieldPrimaryEmail)
```

Important behaviors baked into the converter + `ApplyToQuery` pipeline:

- **Range splitting:** multiple operators on the same field (e.g. `gte`+`lte`) are packed into `FilterX{And: &parts}` by the `FromAPIFilter*` helper.
- **`Oeq` → `In`:** comma-separated equals becomes a SQL `IN (...)` via `filter.FilterString{In: ...}`.
- **`Ocontains` → `Or` of `Contains`:** becomes an `OR ILIKE` chain via `FilterString{Or: ...}`.
- **`FilterLabels` is special:** convert with `FromAPIFilterLabels` and then apply each entry against the JSONB key in the adapter — `ApplyToQuery` does not handle map-shaped predicates on its own.
- **DateTime:** values are already parsed to `time.Time` by `Parse`. `FromAPIFilterDateTime` cannot fail on format anymore, but still returns `error` for the interface.

### Step 5: Tests

Write tests at three layers:

1. **Parser tests** (`api/v3/filters/parse_test.go`): the parse layer is already covered for the generic operator surface; only add cases when introducing a new filter type or operator.
2. **Converter tests** (`api/v3/filters/convert_test.go`): only when adding a new `FromAPIFilter*` helper.
3. **Handler/adapter integration tests:** the important layer for new endpoints — assert that representative `?filter[...]=` query strings produce the expected results. Cover at minimum:
   - shorthand `filter[name]=foo`
   - explicit `filter[name][eq]=foo`
   - `filter[name][contains]=...`
   - `filter[name][oeq]=a,b,c`
   - a range query `filter[created_at][gte]=...&filter[created_at][lte]=...`
   - an unknown field returns 400
   - a mutually exclusive combo returns 400
   - dot-notation against `labels` if applicable

Use `httptest.NewRequest` and assert on the response body or the captured service input.

## Common Patterns and Gotchas

### Picking the right filter type

| You want…                                              | API type            | Predicate                |
| ------------------------------------------------------ | ------------------- | ------------------------ |
| Equality + contains + ranges on a string column        | `FilterString`      | `*filter.FilterString`   |
| ULID column (eq/neq/contains/oeq/ocontains/exists)     | `FilterULID`        | `*filter.FilterULID`     |
| Equality + neq + IN-list on an enum-like string column | `FilterStringExact` | `*filter.FilterString`   |
| Numeric column with ranges                             | `FilterNumeric`     | `*filter.FilterFloat`    |
| Timestamp column with ranges                           | `FilterDateTime`    | `*filter.FilterTime`     |
| Boolean flag column                                    | `FilterBoolean`     | `*filter.FilterBoolean`  |
| Single label map key                                   | `FilterLabel`       | `*filter.FilterString`   |
| Full labels map                                        | `FilterLabels`      | `map[string]filter.FilterString` |

### Dot notation and label maps

- `filter[labels.env][eq]=prod` is supported by treating the **first** `.` as the delimiter; the remainder is the map key. `.` is itself a legal label-key character, so anything after the first dot is the key verbatim.
- Allow-listing must explicitly opt the `labels` field into dot-filtering (the struct field must be typed as `FilterLabels`); otherwise dot-notation against a regular field is rejected.
- `nexists` is **only** valid on additionalProperties maps (`labels`, `metadata`) — do not document it for normal columns.

### Datetime values

`FilterDateTime` holds `*time.Time` and `Parse` rejects malformed RFC-3339 strings at parse time (via `ErrInvalidDateTime`). The converter cannot produce format errors; its `error` return is a forward-compat hook.

### Repeated parameters

`?filter[f][eq]=a&filter[f][eq]=b` is **rejected** — this is intentional. To express OR semantics use `oeq`. Do not try to "fix" the parser to merge repeated keys.

### Range queries

Multiple range operators on the same field (e.g. `gte`+`lte`) are packed into `And{gte, lte}` by the converter. Validation of pathological combinations (e.g. both `gt` and `gte`) lives in `pkg/filter.FilterX.Validate()`; the service input's own `Validate()` surfaces those errors.

### Case sensitivity

- `contains` / `ocontains` are **case-insensitive** (ILIKE under the hood).
- `eq` / `neq` are **case-sensitive** by default. If a column should match case-insensitively, document that in TypeSpec and either lowercase the value before storing it or use a different operator. Per the AIP spec, fields that are case-sensitive must be explicitly stated as such in the OAS.

### Quantifiers (`any` / `all`) on list fields

The Kong AIP spec allows `?filter[tags][eq][any]=urgent` and `[all]` quantifiers on **list-typed** fields. **The current OpenMeter implementation does NOT support quantifiers.** If a request comes in for a list field, raise this with the user before attempting to add it — this is a parser-level extension, not a per-endpoint change.

### Things the parser intentionally does NOT support

- AIP-160 expression syntax (free-form `name = "x" AND age > 5`)
- Logical NOT, parenthesized sub-expressions
- Function calls (`startsWith(...)`, etc.)
- Mixed AND/OR at the API layer beyond what `oeq` / `ocontains` / converter-built `And` chains provide
- Quantifiers on list fields (see above)

If a customer asks for any of these, treat it as a feature request, not a bug fix.

## Error Handling

- **`filters.Parse` errors** (from the generated binding) surface as `InvalidParamFormatError{ParamName: "filter"}` and are translated to 400 by the API error encoder.
- **`FromAPIFilter*` errors** (today mostly unreachable, but the surface exists) should be wrapped with `apierrors.NewBadRequestError` using `apierrors.InvalidParamSourceQuery` and `Field: "filter[<field>]"`.
- **`pkg/filter.Validate()` errors** (from the service input's own `Validate()`) surface as `models.GenericValidationError` and are translated by the handler's error encoder. The caller does not need special casing.

Representative error messages from `Parse`:

- `unknown filter field(s): foo, bar` — client used a field not declared on the input struct
- `unsupported operator` — client used an operator outside the supported set
- `filter[count][eq]: invalid number "abc"` — type coercion failed
- `filter[field]: only one filter can be set` / `gt and gte are mutually exclusive` — validation rejected the combination (raised in `pkg/filter.Validate`)
- `filter parameter "...": value too long (max 1024 bytes)` / `too many comma-separated items (max 50)` — security caps tripped
- `filter parameter "...": repeated query parameter not allowed (got 2 values)` — duplicate keys

## Reference Files

- `api/v3/filters/filter.go` — API-layer filter structs (no methods; plain data shapes)
- `api/v3/filters/parse.go` — `Parse` entry point, operator constants, per-type parsers, security caps (lines 16–19)
- `api/v3/filters/convert.go` — `FromAPIFilter*` helpers (String, ULID, Label, Labels, StringExact, Numeric, DateTime, Boolean)
- `api/v3/filters/parse_test.go`, `api/v3/filters/convert_test.go` — canonical examples of supported syntax
- `pkg/filter/filter.go` — `Filter` interface, predicate types, `Validate`, `Select`, `ApplyToQuery` (line 743)
- `api/v3/handlers/customers/list.go` — reference handler using `FromAPIFilterString`
- `openmeter/customer/customer.go:296` — reference service input struct typed with `*filter.FilterString` fields and a `Validate()` method
- `openmeter/customer/adapter/customer.go:52` — reference adapter using `filter.ApplyToQuery`
- `../api/rules/aip-160-filtering.md` — TypeSpec-side rule: `Common.*FieldFilter` ↔ Go `filters.Filter*` mapping, `Shared.ResourceFilters`, label dot-notation

## Important Reminders

- Service input structs hold `*filter.*` predicates, not `*filters.*` API types. Conversion is the handler's job.
- Use `filter.ApplyToQuery(query, input.Field, dbField)` in adapters, not `.Select(...)` by hand — the helper handles nil-skip and predicate construction.
- Every API filter field **must** appear on the generated `params.Filter` struct (from TypeSpec) for handlers to see it. Run `make gen-api` after editing TypeSpec.
- Validation belongs on the `pkg/filter.*` predicate (called from the service input's `Validate()`), not on the API-layer types.
- Do not invent new operators or quantifiers without first updating `api/v3/filters/parse.go`, `pkg/filter.FilterX.Validate`, and the matching `FromAPIFilter*` — the parser is the contract.
- When in doubt about an operator's behavior, read `parse_test.go` and `convert_test.go` — they are the executable spec.
