# AIP-3101 — API versioning

Reference: https://kong-aip.netlify.app/aip/3101/

## Version goes in the URL path, not headers

Kong-style APIs version exclusively via the URL path. Version numbers appear in logs and route configuration, which makes the API surface explicit to operators and consumers.

## Major versions only

Kong does not use minor versions like `/v1.1`. Any version-number change is treated as a breaking change. Backward-compatible additions (new fields, new optional parameters) are delivered within the same major version — use default values for new fields so older clients continue to work.

## Per-resource versioning, not global

Each resource is versioned individually. Mixing `/v1/users` and `/v2/runtime-groups` in the same API is allowed and encouraged — consumers can migrate resource-by-resource instead of being forced through a global cutover.

## URL structure

- Public APIs: `/v{major}/{resource}`
- Internal APIs: `/{service-prefix}/v{major}/{resource}`
- **Nested versions are forbidden** — you must not have both `v1` and `v2` in the same path, e.g., `/v1/foo/{id}/v2/bar/{id}` is invalid.

## OpenMeter specifics

OpenMeter v3 APIs live under `/api/v3/openmeter/<resource>`. The `api/v3/` prefix is fixed for the AIP package.
