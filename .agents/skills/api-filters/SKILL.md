---
name: api-filters
description: Add, modify, or convert AIP-style query-parameter filters on v3 list endpoints. Use when adding filterable fields to a list API, wiring filter parsing into a handler, converting filters into pkg/filter or Ent predicates, or debugging filter parsing/validation behavior.
user-invocable: false
argument-hint: "[resource or list endpoint to add filters to]"
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

# v3 API Filter Parsing

You are helping the user add or modify AIP-style query-parameter filters on an OpenMeter v3 list endpoint.

OpenMeter follows the **Kong AIP filter spec** (NOT Google AIP-160 expression syntax). Filters use the deepObject query-parameter encoding `?filter[field][op]=value`. The implementation lives in `api/v3/filters/` and converts into the lower-level `pkg/filter` predicate model that Ent query builders consume.

## Relationship to other skills

Filtering straddles two layers that the repo skill set keeps separate:

- **TypeSpec / OAS side** — `Common.*FieldFilter` types, `Shared.ResourceFilters`, `deepObject` exposure, label dot-notation. See `../api/rules/aip-160-filtering.md` (the canonical Kong AIP-160 rule for OpenMeter). Use the `/api` skill when you also need to scaffold or modify the TypeSpec operation itself.
- **Go implementation side** — what this skill covers: `api/v3/filters.Parse`, the typed filter structs, `Convert*` helpers, Ent predicate wiring, validation rules, gotchas.

If you are adding a brand-new filterable endpoint, invoke `/api` first to wire up the TypeSpec + handler shell, then come back here (or run both in sequence) for the filter decoder and adapter code. If you are only adding/modifying filters on an existing endpoint, this skill is enough on its own.

## Context

- **Public package:** `api/v3/filters/` — typed filter structs, the `Parse` entry point, and converters
- **Internal predicate model:** `pkg/filter/` — used by both v3 and v1 (via `openmeter/apiconverter/filter.go`) and applied directly to Ent queries via `.Select(fieldName)`
- **Reference implementation in use:** `openmeter/customer/adapter/customer.go` (customer list filters)
- **Kong AIP spec for filtering:** `../api/rules/aip-160-filtering.md` (the rule file this skill depends on for TypeSpec-side guidance)

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

### Operator-by-type matrix

The Go types accept a **superset** of what each matching OAS `Common.*FieldFilter` type advertises. Operators marked with † are Go-only extensions — the OAS does not document them, so **do not rely on them as public contract**. See `../api/rules/aip-160-filtering.md` for the canonical per-OAS-type operator set.

| Go type             | Accepted operators                                                                 |
| ------------------- | ---------------------------------------------------------------------------------- |
| `FilterString`      | eq, neq, contains, oeq, ocontains, gt†, gte†, lt†, lte†, exists†, nexists†         |
| `FilterStringExact` | eq, neq, oeq                                                                       |
| `StringFilter`      | eq, neq, contains (internal convenience — not a Common type)                       |
| `FilterNumeric`     | eq, gt, gte, lt, lte, neq†, oeq†                                                   |
| `FilterDateTime`    | eq, gt, gte, lt, lte (RFC-3339 values, parsed at convert-time)                     |
| `FilterBoolean`     | eq (`true`/`false` literals — wraps the bare OAS scalar)                           |

### Multi-filter semantics

- Multiple `filter[...]` parameters with **different fields** combine with **AND**.
- A single field with `oeq` / `ocontains` combines its values with **OR** (`IN (...)` or `OR ILIKE ...`).
- A single field with both `gte` and `lte` is a **range** (combined as **AND** at convert-time).
- The bare-key existence shortcut maps to `IS NOT NULL`; `nexists` only works on schemaless maps (`labels`, `metadata`).

### Mutual exclusivity rules (enforced by `Validate`)

- `eq` / `neq` are mutually exclusive
- `eq` / `contains` are mutually exclusive
- `gt` and `gte` are mutually exclusive (pick one lower bound)
- `lt` and `lte` are mutually exclusive (pick one upper bound)
- Only one of `eq` and any range operator may be set
- Range queries: each bound is optional — an open-ended range with only a lower bound (`gt` or `gte`) or only an upper bound (`lt` or `lte`) is valid. When both bounds are present, the converter splits them into `And{lower, upper}` single-operator nodes. Within each direction you may set at most one: `gt` or `gte` (not both), `lt` or `lte` (not both).

### Hard limits (security)

- **256 bytes** per single value (`maxFilterValueLength`, `parse.go:23`)
- **50 items** per comma-separated list (`maxCommaSeparatedItems`, `parse.go:18`)
- **Repeated query params for the same key are rejected** (e.g., `?filter[f][eq]=a&filter[f][eq]=b`)
- **Unknown filter fields are rejected** before any other validation

## Filter Types in `api/v3/filters/filter.go`

All filter types follow the same shape — fields are pointers/slices so absence vs presence is unambiguous:

```go
type FilterString struct {
    Eq        *string
    Neq       *string
    Gt        *string
    Gte       *string
    Lt        *string
    Lte       *string
    Contains  *string
    Oeq       []string
    Ocontains []string
    Exists    *bool
}
// Plus: FilterStringExact, StringFilter, FilterNumeric,
//       FilterDateTime, FilterBoolean
```

Every type implements:

- `IsEmpty() bool` — true when no operators are set
- `Validate(ctx context.Context, field string) error` — enforces mutual exclusivity, range rules, and value parsability

## Workflow

Follow these steps in order. Use the `/api` skill alongside this one when you also need to touch TypeSpec.

### Step 1: Define the filterable fields in TypeSpec

In `api/spec/packages/aip/src/<domain>/operations.tsp`, define a named filter model for the list operation and expose it as `filter` with `style: "deepObject", explode: true`. Use the `Common.*FieldFilter` types from `common/parameters.tsp` — **do not hand-roll filter models**.

The canonical rule for _which_ `Common.*FieldFilter` type to pick, the `Shared.ResourceFilters` spread, label dot-notation, and OAS documentation requirements is `../api/rules/aip-160-filtering.md`. That rule also includes the TypeSpec type ↔ Go `filters.Filter*` mapping used in Step 2 below. Read it once before picking types — this skill is not the source of truth for the TypeSpec side.

The customer list endpoint is the canonical worked example for how these types connect to a real list operation.

### Step 2: Define the input struct on the service

In your service input type (e.g., `openmeter/customer/customer.go`'s `ListCustomersInput`), add typed filter fields. Use json tags — `Parse` reflects on them to learn the allowed field names:

```go
type ListCustomersInput struct {
    Namespace string

    Key          *filters.FilterString `json:"key"`
    Name         *filters.FilterString `json:"name"`
    PrimaryEmail *filters.FilterString `json:"primary_email"`
    CreatedAt    *filters.FilterDateTime `json:"created_at"`
    // ...
}
```

Pick the **narrowest filter type** that covers the field's documented operators. Prefer `FilterStringExact` or `StringFilter` over `FilterString` when range operators are not needed — narrower types fail loudly if a client sends an unsupported operator.

### Step 3: Parse query params in the HTTP decoder

In your handler's request decoder (the first argument to `httptransport.NewHandlerWithArgs`), call `filters.Parse` against the request URL values, passing a pointer to the filters sub-struct (or to your full input):

```go
import "github.com/openmeterio/openmeter/api/v3/filters"

func (h *handler) ListCustomers() ListCustomersHandler {
    return httptransport.NewHandlerWithArgs(
        func(ctx context.Context, r *http.Request, params api.ListCustomersParams) (ListCustomersRequest, error) {
            ns, err := h.resolveNamespace(ctx)
            if err != nil {
                return ListCustomersRequest{}, err
            }

            req := ListCustomersRequest{Namespace: ns}

            // Parse filters from raw query string — required because deepObject
            // params (filter[field][op]=...) are not represented in api.ListCustomersParams.
            if err := filters.Parse(r.URL.Query(), &req); err != nil {
                return req, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
                    {Field: "filter", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
                })
            }

            // pagination, sort, etc.
            return req, nil
        },
        // operation func, encoder, options...
    )
}
```

`filters.Parse` will:

1. Reflect over the target struct's `json` tags to build the known-field set
2. Reject any `filter[unknown_field]...` query parameter as an error
3. For each known field, dispatch to the type-specific parser (string / numeric / datetime / bool)
4. Run `Validate()` on each parsed filter (mutual exclusivity, range rules, type coercion)
5. Populate the target struct in place

### Step 4: Convert to `pkg/filter` and apply to Ent

Filter conversion lives in `api/v3/filters/convert.go`. Use the matching `Convert*` helper in your **adapter** (not the handler) to translate API-shaped filters into the internal `pkg/filter` predicate model, then call `.Select(fieldName)` to get an Ent predicate:

```go
import (
    "github.com/openmeterio/openmeter/api/v3/filters"
    customerdb "github.com/openmeterio/openmeter/openmeter/ent/db/customer"
)

func (a *adapter) ListCustomers(ctx context.Context, in customer.ListCustomersInput) (...) {
    q := a.db.Customer.Query().Where(customerdb.Namespace(in.Namespace))

    if in.Key != nil {
        if p := filters.ConvertFilterString(*in.Key).Select(customerdb.FieldKey); p != nil {
            q = q.Where(p)
        }
    }
    if in.Name != nil {
        if p := filters.ConvertFilterString(*in.Name).Select(customerdb.FieldName); p != nil {
            q = q.Where(p)
        }
    }
    // ...
}
```

Important behaviors `Convert*` handles for you:

- **Range splitting:** `gte`+`lte` on the same field becomes an `And` of two single-op `pkg/filter` nodes (the internal model only allows one operator per node).
- **`Oeq` → `In`:** comma-separated equals becomes a SQL `IN (...)` clause.
- **`Ocontains` → `Or` of `Contains`:** comma-separated contains becomes `OR ILIKE` chain.
- **DateTime parsing:** `ConvertFilterDateTime` parses RFC-3339 strings into `time.Time` and returns an error if parsing fails — propagate it up as a 400.
- **Empty collapse:** if `IsEmpty()` is true, the helper returns nil so you can skip the `.Where(...)` call entirely.

### Step 5: Tests

Write tests at three layers:

1. **Parser tests** (mirror `api/v3/filters/parse_test.go`): the parse layer is already covered for the generic operator surface; you only need to add cases when introducing a new filter type.
2. **Converter tests** (mirror `api/v3/filters/convert_test.go`): again, only when adding a new converter.
3. **Handler/adapter integration tests:** the important layer for new endpoints — assert that representative `?filter[...]=` query strings produce the expected SQL/Ent query results. Include at least:
   - shorthand `filter[name]=foo`
   - explicit `filter[name][eq]=foo`
   - `filter[name][contains]=...`
   - `filter[name][oeq]=a,b,c`
   - a range query `filter[created_at][gte]=...&filter[created_at][lte]=...`
   - an unknown field returns 400
   - a mutually exclusive combo returns 400
   - dot-notation against `labels` if applicable

Use `httptest.NewRequest` and assert on the response body or the captured service input — see existing customer list tests for the pattern.

## Common Patterns and Gotchas

### Picking the right filter type

| You want…                                              | Use                 |
| ------------------------------------------------------ | ------------------- |
| Equality + contains + ranges on a string column        | `FilterString`      |
| Equality + neq + IN-list on an enum-like string column | `FilterStringExact` |
| Tiny string filter (eq/neq/contains only)              | `StringFilter`      |
| Numeric column with ranges                             | `FilterNumeric`     |
| Timestamp column with ranges                           | `FilterDateTime`    |
| Boolean flag column                                    | `FilterBoolean`     |

### Dot notation and label maps

- `filter[labels.env][eq]=prod` is supported by treating the **first** `.` as the delimiter; the remainder is the map key. `.` is itself a legal label-key character, so anything after the first dot is the key verbatim.
- Allow-listing must explicitly opt the `labels` field into dot-filtering; otherwise dot-notation against a regular field is rejected.
- `nexists` is **only** valid on additionalProperties maps (`labels`, `metadata`) — do not document it for normal columns.

### Datetime values

`FilterDateTime` keeps strings raw at parse time and only validates RFC-3339 in `ConvertFilterDateTime`. This means:

- Malformed timestamps surface as **convert-time errors**, not parse-time errors.
- Always handle the `ConvertFilterDateTime` error and return a 400.

### Repeated parameters

`?filter[f][eq]=a&filter[f][eq]=b` is **rejected** — this is intentional. To express OR semantics use `oeq`. Do not try to "fix" the parser to merge repeated keys.

### Range queries

`gte`+`lte` together is the _only_ way to express a range. Do not document or support `gt`+`lte`, `gte`+`lt`, etc., until/unless the parser learns to allow those combinations — the current `Validate()` rejects mixed bound styles.

### Case sensitivity

- `contains` / `ocontains` are **case-insensitive** (ILIKE under the hood).
- `eq` / `neq` are **case-sensitive** by default. If a column should match case-insensitively, document that in TypeSpec and either lowercase the value before storing it or use a different operator. Per the AIP spec, fields that are case-sensitive must be explicitly stated as such in the OAS.

### Quantifiers (`any` / `all`) on list fields

The Kong AIP spec allows `?filter[tags][eq][any]=urgent` and `[all]` quantifiers on **list-typed** fields. **The current OpenMeter implementation does NOT support quantifiers.** If a request comes in for a list field, raise this with the user before attempting to add it — this is a parser-level extension, not a per-endpoint change.

### Things the parser intentionally does NOT support

- AIP-160 expression syntax (free-form `name = "x" AND age > 5`)
- Logical NOT, parenthesized sub-expressions
- Function calls (`startsWith(...)`, etc.)
- Mixed AND/OR at the API layer beyond what `oeq`/`ocontains` provide
- Quantifiers on list fields (see above)

If a customer asks for any of these, treat it as a feature request, not a bug fix.

## Error Handling

`filters.Parse` returns plain `error` values with structured messages (no `ValidationIssue` yet). Wrap them in `apierrors.NewBadRequestError` with `apierrors.InvalidParamSourceQuery` so the field surfaces in `invalid_parameters` per AIP. Example error messages you may see:

- `unknown filter field(s): foo, bar` — client used a field not declared on the input struct
- `unknown filter operator "like"` — client used an operator outside the supported set
- `filter[count][eq]: invalid number "abc": ...` — type coercion failed
- `filter[field]: only one filter can be set` / `gt and gte are mutually exclusive` — validation rejected the combination
- `filter parameter "...": value too long (max 256 bytes)` / `too many comma-separated items (max 50)` — security caps tripped
- `filter parameter "...": repeated query parameter not allowed (got 2 values)` — duplicate keys

## Reference Files

- `api/v3/filters/filter.go` — typed filter structs + `Validate` rules
- `api/v3/filters/parse.go` — `Parse` entry point, type-specific parsers, security caps (constants at lines 18 and 23)
- `api/v3/filters/convert.go` — `Convert*` helpers, range-splitting logic
- `api/v3/filters/parse_test.go`, `api/v3/filters/convert_test.go` — canonical examples of supported syntax
- `pkg/filter/filter.go` — internal predicate model used by Ent query builders
- `openmeter/apiconverter/filter.go` — v1 API → `pkg/filter` (goverter-generated)
- `openmeter/customer/adapter/customer.go` — current real-world consumer of v3 filters
- `../api/rules/aip-160-filtering.md` — TypeSpec-side rule: `Common.*FieldFilter` ↔ Go `filters.Filter*` mapping, `Shared.ResourceFilters`, label dot-notation

## Important Reminders

- All filterable fields **must** be declared on a Go struct with a `json` tag — `Parse` is reflection-driven; fields without json tags are invisible to it.
- All filterable fields **must** be documented in the TypeSpec OAS for the endpoint, including the supported operators per field.
- Use `Convert*` helpers in the **adapter**, not the handler — keep API-shape vs DB-predicate separation clean.
- Always return 400 (via `apierrors.NewBadRequestError`) for `Parse` errors and convert errors so they surface in `invalid_parameters`.
- Do not invent new operators or quantifiers without first updating `api/v3/filters/parse.go`, `Validate`, and the corresponding `Convert*` — the parser is the contract.
- When in doubt about an operator's behavior, read `parse_test.go` — it is the executable spec.
