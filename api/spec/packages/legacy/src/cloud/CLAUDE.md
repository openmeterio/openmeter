# cloud

<!-- archie:ai-start -->

> Cloud-variant spec entry point that re-exports the entire OpenMeter API under the OpenMeterCloud namespace with cloud-specific auth schemes (BearerAuth + cookie + portal token) and Svix/cloud server URLs. Contains no new type definitions — only namespace interface extensions.

## Patterns

**Interface extension via extends** — Every interface in cloud/main.tsp extends an OpenMeter interface with a new @route and @tag override. Never copy-paste operations — always extend. (`@route("/api/v1/meters") @tag("Meters") interface MetersEndpoints extends OpenMeter.MetersEndpoints {}`)
**Separate OpenMeterCloud namespace blocks** — Related interface groups are collected in separate namespace OpenMeterCloud { ... } blocks; multiple blocks with the same namespace name are valid TypeSpec. (`namespace OpenMeterCloud { @route("/api/v1/apps") interface AppsEndpoints extends OpenMeter.AppsEndpoints {} }`)
**Auth overrides at service level** — auth.tsp defines CloudTokenAuth/CloudCookieAuth/CloudPortalTokenAuth; the @useAuth on @service replaces the default self-hosted auth for the cloud spec. (`@service(#{ title: "OpenMeter Cloud API" }) @useAuth(CloudTokenAuth | CloudCookieAuth) namespace OpenMeterCloud;`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `auth.tsp` | Defines the three cloud auth schemes as models. No routes or operations. | CloudPortalTokenAuth is a BearerAuth; the portal meters endpoint separately applies @useAuth(CloudPortalTokenAuth) for portal-only scope. |
| `main.tsp` | Cloud OpenAPI entry point: imports '..' (the base legacy spec) and auth.tsp, then declares the OpenMeterCloud namespace with all extends interfaces and tag metadata. | When adding a new endpoint group to the base OpenMeter namespace, add a matching extends interface here or it won't appear in the cloud output. |

## Anti-Patterns

- Defining new models or type aliases in cloud/ — all types belong in the base OpenMeter namespace
- Duplicating operation definitions instead of using extends
- Adding auth decorators to individual operations in cloud/main.tsp — service-level @useAuth covers all

## Decisions

- **cloud/main.tsp extends rather than duplicates base OpenMeter interfaces** — Single source of truth for operation contracts; cloud only differs in auth, servers, and tag metadata.

<!-- archie:ai-end -->
