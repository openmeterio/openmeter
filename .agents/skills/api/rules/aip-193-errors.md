# AIP-193 — Error responses

Reference: https://kong-aip.netlify.app/aip/193/

All errors follow RFC 7807 (`Content-Type: application/problem+json`) with the following mandatory fields:

| Field      | Meaning                                                                  |
| ---------- | ------------------------------------------------------------------------ |
| `type`     | URL uniquely identifying the error; dereferences to human-readable docs  |
| `status`   | HTTP status code as an integer                                           |
| `title`    | Short human summary; **stable across occurrences** (except localization) |
| `detail`   | Per-occurrence explanation; may embed request values in `[brackets]`     |
| `instance` | Correlation ID, format: `kong:trace:<correlation_id>`                    |

## `invalid_parameters` extension (400 responses)

Every `400 Bad Request` response must include an `invalid_parameters` array. Each entry has:

- `field` — dot-notation path for body fields (with array indices); parameter name for query/path/header
- `rule` — validation type code (`required`, `enum`, `min_length`, `is_string`, etc.)
- `reason` — human-readable failure explanation
- `source` — one of `path`, `body`, `header`, `query`

Some rules require additional fields; for example `enum` includes a `choices` array.

## Status codes defined in AIP-193

AIP-193 explicitly documents these codes: **400, 401, 403, 404, 409**.

## Access control rule

When a caller lacks access to a resource:

- Return `403 Forbidden` if the resource is owned by the caller's organization (they exist but can't touch it)
- Return `404 Not Found` otherwise — this prevents data-existence leakage across tenants

## See also

- `rules/openmeter-error-types.md` — OpenMeter `Common.*` types that wire AIP-193 onto operations, including OpenMeter-only status codes (410, 413, 422).
- `rules/inline-errors.md` — inline errors returned **inside** a 2xx response body (partial successes, pre-flight validation on draft resources). AIP-193 itself only covers transport-level error responses.
