# AIP-129 — Labels

Reference: https://kong-aip.netlify.app/aip/129/

`Common.Labels` stores mutable user-managed metadata. `Common.PublicLabels` is for publicly visible labels. Labels are case-sensitive key/value pairs.

Note: `Shared.Resource` includes `labels` by default but **not** `public_labels` — add `public_labels` explicitly on resources that need it.

## Key constraints

- Maximum **63 characters**
- Must **start and end with an alphanumeric** character
- Permitted internal characters: alphanumerics, dashes (`-`), underscores (`_`), dots (`.`)
- Reserved prefixes (case-insensitive): cannot begin with `kong`, `konnect`, `insomnia`, `mesh`, `kic`, `kuma`, or `_`

## Value constraints

- Maximum **63 characters** (same as keys — **not** 255)
- Same character rules as keys
- **Values must not be empty**

## Resource-level limits

- Maximum **50 user-defined labels** per resource
- Both `labels` and `public_labels` return `{}` (empty object) when unset, never omitted — see `aip-3106-empty-fields.md`

## PATCH semantics

- `PATCH` with a `null` value **deletes** that label key
- `PATCH` with an absent key leaves that entry untouched
- `PATCH` with a new key adds the entry
- Attempting to delete a missing key is not an error

## Filtering

Labels support filtering via dot-notation — see `aip-160-filtering.md`.
