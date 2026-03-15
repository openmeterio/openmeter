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

- **OpenMeter** (`openmeter.tsp`): OpenMeter v3 API (events, meters, customers, subscriptions, billing, etc.).
- **Konnect** (`konnect.tsp`): Konnect metering & billing API, same surface with Konnect-specific auth and servers.

See [`packages/aip/README.md`](packages/aip/README.md) for patterns and guidelines.
