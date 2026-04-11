# AIP-122 — Resource names & URL paths

Reference: https://kong-aip.netlify.app/aip/122/

## What AIP-122 actually covers

AIP-122 is a narrow rule about **resource names, URL paths, and field names** — nothing more:

- **URL paths** use `kebab-case` for multi-word resource names: `/v1/konnect-services/domestic-cats`
- Prefer lowercase resource names like `services`, `users`
- **Field names** (JSON property names) are lowercase and use `snake_case` for multi-word names: `primary_color`, `billing_profile_id`
- Uppercase, title case, and `camelCase` are exceptions and require justification

That's it. AIP-122 says nothing about TypeSpec model names, enum names, path parameter casing, or operation IDs.

## OpenMeter-local naming conventions (not from AIP-122)

These are locally enforced by linter rules in `api/spec/packages/aip/lib/rules/` but are **not** rules from AIP-122 itself. They are collected here for convenience.

| Element                      | Convention   | Example                                 | Source                                        |
| ---------------------------- | ------------ | --------------------------------------- | --------------------------------------------- |
| URL paths                    | `kebab-case` | `/api/v3/openmeter/llm-cost`            | AIP-122                                       |
| Field/property names         | `snake_case` | `created_at`, `billing_profile_id`      | AIP-122                                       |
| Enum wire values             | `snake_case` | `"unique_count"`                        | AIP-126 (see `aip-126-enums.md`)              |
| Operation IDs                | `kebab-case` | `update-meter`, `list-billing-profiles` | AIP-134 / AIP-135 (see `aip-134-135-crud.md`) |
| Model names (TypeSpec)       | `PascalCase` | `BillingProfile`                        | OpenMeter linter only                         |
| Enum type names (TypeSpec)   | `PascalCase` | `MeterAggregation`                      | OpenMeter linter only                         |
| Enum member names (TypeSpec) | `PascalCase` | `UniqueCount`                           | OpenMeter linter only                         |
| Path parameters (TypeSpec)   | `camelCase`  | `meterId`, `customerId`                 | OpenMeter linter only                         |

The TypeSpec-facing casing (`PascalCase` model names, `camelCase` path parameters) is an OpenMeter convention because TypeSpec model identifiers and path parameter identifiers are separate from the JSON wire format. The wire format still complies with AIP-122.

## Base resource models (not from AIP-122)

These are OpenMeter `Shared.Resource` conventions defined in `api/spec/packages/aip/src/shared/resource.tsp`, not AIP-122 rules. They use AIP-122-compliant snake_case field names on the wire.

- **`Shared.Resource`** — `id`, `name`, `description`, `labels`, `created_at`, `updated_at`, `deleted_at`
- **`Shared.ResourceWithKey`** — same as `Shared.Resource` plus `key: ResourceKey` (`Lifecycle.Read, Lifecycle.Create` only)
- **`Shared.ResourceImmutable`** — `Shared.Resource` with `updated_at` and `deleted_at` omitted, for resources that cannot be mutated after creation

`public_labels` is **not** part of `Shared.Resource`; add it explicitly on resources that need publicly visible labels (see `aip-129-labels.md`).

```tsp
model Meter {
  ...Shared.Resource;

  @visibility(Lifecycle.Read, Lifecycle.Create)
  event_type: string;
}
```
