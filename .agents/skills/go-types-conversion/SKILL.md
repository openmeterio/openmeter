---
name: go-types-conversion
description: Naming convention for type-translation files and functions. Use when creating or editing files that convert between domain, API, and DB types.
user-invocable: true
allowed-tools: Read, Edit, Write, Grep, Glob
---

# Type Translation Naming

Apply to new and touched code. Do not rename legacy symbols unsolicited.

## File naming

| Path contains                                     | File name    | Purpose      |
| ------------------------------------------------- | ------------ | ------------ |
| `httpdriver/`, `httphandler/`, `api/v3/handlers/` | `convert.go` | API ↔ domain |
| `adapter/`, `repo/`                               | `mapping.go` | DB ↔ domain  |

Split large files by entity: `convert_plan.go`, `mapping_subscription.go`.

`mapper.go` is forbidden. Rename it to `convert.go` or `mapping.go` (based on layer) when the file is touched.

## Function naming

### Shape: `From<Qualifier><Thing>` / `To<Qualifier><Thing>`

The qualifier is `API` or `DB` — no other qualifiers (`Domain`, `Model`, package-name infixes).

The suffix `<Thing>` is the **non-domain type's unqualified name** — the API type or DB type, not the domain type. This keeps it stable: a matched pair (`FromAPI<Thing>` / `ToAPI<Thing>`) always refers to the same non-domain type, regardless of direction.

- `FromAPI<Thing>` — takes the API type `<Thing>` as input, returns the domain representation.
- `ToAPI<Thing>` — takes the domain type as input, returns the API type `<Thing>`.
- Same for `FromDB<Thing>` / `ToDB<Thing>`.

### Examples

```go
// API ↔ domain
FromAPIPlan(a api.Plan) (plan.Plan, error)
ToAPIPlan(p plan.Plan) api.Plan

// Suffix is the API type name, even when domain type differs
FromAPIPlanCreate(a api.PlanCreate) (plan.CreateInput, error)
ToAPIPlanCreate(p plan.CreateInput) api.PlanCreate

FromAPIProRatingConfig(a api.ProRatingConfig) (productcatalog.ProRatingConfig, error)
ToAPIProRatingConfig(p productcatalog.ProRatingConfig) *api.ProRatingConfig

// DB ↔ domain — suffix is the DB type name
FromDBSubscription(row *db.Subscription) (subscription.Subscription, error)
ToDBSubscription(s subscription.Subscription) *db.Subscription

FromDBChargeFlatFee(row *entdb.ChargeFlatFee) (flatfee.Charge, error)
ToDBChargeFlatFee(c flatfee.Charge) *entdb.ChargeFlatFee
```

### Additional rules

- **Exported** functions always include the type suffix (`FromAPIPlanCreate`, not bare `FromAPI`).
- **Unexported** helpers in a single-type file may drop the suffix (`fromDB`, `toAPI`).
- **Fallible** (parse/validate) → `(T, error)`. **Infallible** (projection) → `T`. Typically `FromAPI…` / `FromDB…` is fallible; the reverse is not.
- **Batch helpers** use the plural: `FromAPIPlans`, `ToDBSubscriptions`. Same suffix rule — the plural of the non-domain type name.

### Forbidden patterns

- `Map…`, `Convert…To…`, primary `As…`
- `<Source>To<Target>` shape (e.g. `APIToPlan`)
- Bare `FromAPI` / `ToDB` without a type suffix
- goverter or other codegen type mappers

## Decision tree

### Naming a function

1. **Pick the qualifier:** API/HTTP/wire on one side → `API`. DB/persistence on one side → `DB`.
2. **Pick the suffix:** the non-domain type's unqualified name (`Plan`, `PlanCreate`, `ChargeFlatFee`).
3. **Pick the return style:** fallible → `(T, error)`, infallible → `T`.
4. **Exported?** Must include the type suffix. Unexported single-type helper may drop it.

### Interacting with the user

- **File is `mapper.go`?** Flag it — should be `convert.go` or `mapping.go` based on layer. Offer to rename as part of the edit. Don't rename silently.
- **Adding new functions to a legacy file?** Use the new convention for new functions. Don't rename old ones unless asked.
- **Task is "clean up this file"?** Rename, update call sites, `Grep` for the old name to catch misses, keep the rename in its own commit.
- **`// Code generated` header?** Off-limits regardless.

## Suggestion phrasing

Lead with the specific rename and the reason. Keep it short.

> `MapChargeFlatFeeFromDB` — use `FromDBChargeFlatFee`. Want me to rename and update callers?

> Direction looks inverted — `FromAPI…` returns a domain type, so this should be `ToAPIPlan`. Drop the error return if it can't actually fail.

> This file is `mapper.go` — should be `convert.go` since it lives in `httphandler/`. Want me to rename it?
