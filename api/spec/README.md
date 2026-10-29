# OpenMeter API specs

This workspace contains two TypeSpec packages that generate OpenAPI specs:

| Package                        | Description                                                   | Output                                                  |
| ------------------------------ | ------------------------------------------------------------- | ------------------------------------------------------- |
| **Legacy** (`packages/legacy`) | OpenMeter API (v1-v2) and OpenMeter Cloud API                 | `openapi.OpenMeter.yaml`, `openapi.OpenMeterCloud.yaml` |
| **AIP** (`packages/aip`)       | OpenMeter and Konnect metering & billing APIs (v3), AIP-style | `openapi.MeteringAndBilling.yaml` (OpenMeter + Konnect) |

From the repo root, run `make gen-api` (or `make -C api/spec generate`) to build both packages and copy/bundle artifacts into `api/`.

---

## Legacy API (`packages/legacy`)

Legacy specs follow OpenMeter’s existing TypeSpec conventions. See [`packages/legacy/README.md`](packages/legacy/README.md) for patterns and guidelines.

---

## AIP (`packages/aip`)

The AIP package defines v3 metering and billing APIs in line with [Kong’s AIP (API Improvement Proposals)](https://kong-aip.netlify.app/list/).

- **OpenMeter** (`openmeter.tsp`): OpenMeter v3 API.
- **Konnect** (`konnect.tsp`): Konnect metering & billing API, same surface with Konnect-specific auth and servers.

---

## Notification Event contracts (`packages/aip/src/<domain>/notification-events/`)

The producing domain owns the payload, and for rule-produced events the Rule configuration, of each Notification Event type. It declares them in a `<Domain>.NotificationEvents` namespace, mapped to a Go package in `packages/aip/src/notification-events.tsp`. `make update-openapi` generates the Go structs, JSON Schemas, event type values and event type definitions into `openmeter/<domain>/notificationevents`, the list of every definition into `openmeter/notification/v4/notificationcontracts`, and the events-only OpenAPI spec into `api/notificationevents/<domain>/`.

- A released contract is immutable. Change it by adding the next version (`v2.<name>`, `...V2Payload`) next to it, never by editing it in place. Nested types follow the same rule: a breaking change needs a new type. Reviewers enforce this; generation does not.
- Payloads are dedicated models. They may use `Shared` types but must not reuse API resource models, so that API changes cannot change a contract.
- [`packages/typespec-notification-events`](packages/typespec-notification-events) enforces the naming and `@id` conventions at compile time.
