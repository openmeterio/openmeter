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

## Notification Event contracts

The producing domain owns the payload, and for rule-produced events the Rule configuration, of each Notification Event type. They are ordinary models of this TypeSpec program, declared next to the domain's other models and marked with `@eventType` and `@jsonSchema`. The v3 Event API references them, so the Go types are generated into `api/v3/models` with the rest of the API. `make update-openapi` also generates their JSON Schemas, Event type values and definitions into `openmeter/notification/v4/definitions`.

- A released contract is immutable. Change it by adding the next version (`v2.<name>`, `...V2Payload`) next to it, never by editing it in place. Nested types follow the same rule: a breaking change needs a new type. A test fails when a generated definition no longer matches [`definitions.sum`](../../openmeter/notification/v4/definitions/definitions.sum); rewrite the file with `make update-notification-definitions-sum` only for a definition that was never released.
- A payload may reuse API models. A change to such a model changes every contract that uses it, and `definitions.sum` catches that.
- [`packages/typespec-notification-events`](packages/typespec-notification-events) enforces the naming and `@id` conventions at compile time.
