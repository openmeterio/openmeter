# cloud

<!-- archie:ai-start -->

> Cloud-variant spec entry point that re-exports the entire OpenMeter API under the `OpenMeterCloud` namespace with cloud-specific auth schemes (BearerAuth + cookie + portal token) and Svix/cloud server URLs. Contains no new type definitions — only namespace interface extensions.

## Patterns

**Interface extension via `extends`** — Every interface in cloud/main.tsp extends an OpenMeter interface with a new `@route` and `@tag` override, e.g. `interface MetersEndpoints extends OpenMeter.MetersEndpoints {}`. Never copy-paste operations — always extend. (`@route("/api/v1/meters") @tag("Meters") interface MetersEndpoints extends OpenMeter.MetersEndpoints {}`)
**Separate OpenMeterCloud namespace blocks** — Related interface groups are collected in separate `namespace OpenMeterCloud { ... }` blocks for readability. Multiple blocks with the same namespace name are valid TypeSpec. (`namespace OpenMeterCloud { @route("/api/v1/apps") interface AppsEndpoints extends OpenMeter.AppsEndpoints {} }`)
**Auth overrides at service level** — auth.tsp defines CloudTokenAuth, CloudCookieAuth, CloudPortalTokenAuth. The `@useAuth(CloudTokenAuth | CloudCookieAuth)` decorator on the `@service` replaces the default OpenMeter self-hosted auth for the cloud spec. (`@service(#{ title: "OpenMeter Cloud API" }) @useAuth(CloudTokenAuth | CloudCookieAuth) namespace OpenMeterCloud;`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `auth.tsp` | Defines the three cloud auth schemes as models. No routes or operations here. | CloudPortalTokenAuth is a BearerAuth; the portal meters endpoint separately applies `@useAuth(CloudPortalTokenAuth)` to scope portal-only access. |
| `main.tsp` | The cloud OpenAPI spec entry point. Imports `..` (the main legacy spec), then auth.tsp, then declares OpenMeterCloud namespace with all interface extensions. All cloud-facing endpoint routing lives here. | When adding a new endpoint group to the base OpenMeter namespace, add a corresponding `extends` interface here so it appears in the cloud OpenAPI output. |

## Anti-Patterns

- Defining new models or type aliases in cloud/ — all types belong in the base OpenMeter namespace.
- Duplicating operation definitions instead of using `extends`.
- Adding auth decorators to individual operations in cloud/main.tsp — service-level `@useAuth` covers all.

## Decisions

- **cloud/main.tsp extends rather than duplicates base OpenMeter interfaces** — Single source of truth for operation contracts; cloud only differs in auth, servers, and tag metadata.

<!-- archie:ai-end -->
