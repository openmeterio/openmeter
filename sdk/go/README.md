# OpenMeter Go SDK (baseline)

A hand-written, idiomatic Go SDK for the OpenMeter v3 API.

> **Status: baseline / reference.** It implements the three meter endpoints as a
> worked example of the target SDK shape. This shape is intended to be
> reproduced by a TypeSpec emitter. It is not yet a complete SDK.

## Design principles

- **Standard-library public surface.** The `Client` exposes only `net/http`,
  `context`, typed request/response structs, and a typed `APIError`. No
  third-party type appears in the public API.
- **Swappable transport.** Requests run through an `*http.Client`. The default
  one retries with exponential backoff (via an internal dependency hidden behind
  the `*http.Client` seam). Inject your own with `WithHTTPClient` to own retry,
  timeout, proxy, TLS, and tracing behavior.
- **Resource-grouped operations.** Operations hang off resource sub-clients,
  e.g. `client.Meters.List(...)`.
- **Auth survives injection.** The bearer token is set during request
  construction, not via a transport wrapper, so it is applied even when you
  inject your own `*http.Client`.

## Install

```bash
go get github.com/openmeterio/openmeter/sdk/go
```

```go
import openmeter "github.com/openmeterio/openmeter/sdk/go"
```

## Usage

The base URL must include the API version prefix (`/api/v3`). For OpenMeter
Cloud that is `https://openmeter.cloud/api/v3`; for a local server it is
`http://127.0.0.1:8888/api/v3`.

```go
client, err := openmeter.New("https://openmeter.cloud/api/v3", openmeter.WithToken("om_..."))
if err != nil {
    log.Fatal(err)
}

ctx := context.Background()

// Get one meter by ID.
m, err := client.Meters.Get(ctx, "01ABC...")

// List meters with pagination, sort, and filter (query-string params).
page, err := client.Meters.List(ctx, openmeter.MeterListParams{
    Page:   &openmeter.PageParams{Size: openmeter.Int(20), Number: openmeter.Int(1)},
    Sort:   []string{"created_at desc"},
    Filter: &openmeter.MeterFilter{Key: &openmeter.StringFilter{Contains: openmeter.String("tokens")}},
})

// Query a meter for usage (POST body). JSON result:
gran := openmeter.MeterQueryGranularityDay
res, err := client.Meters.Query(ctx, m.ID, openmeter.MeterQueryRequest{Granularity: &gran})

// Same query, CSV via content negotiation (Accept: text/csv):
csv, err := client.Meters.QueryCSV(ctx, m.ID, openmeter.MeterQueryRequest{Granularity: &gran})
```

## Implemented endpoints

| Method | HTTP | Demonstrates |
|---|---|---|
| `Meters.Get` | `GET /openmeter/meters/{id}` | Path parameter |
| `Meters.List` | `GET /openmeter/meters` | Query-string serialization: deepObject `page`, form `sort`, nested deepObject `filter` |
| `Meters.Query` | `POST /openmeter/meters/{id}/query` | Request body |
| `Meters.QueryCSV` | `POST /openmeter/meters/{id}/query` | Content negotiation (JSON vs CSV) |

## Errors

Non-2xx responses return a typed `*APIError` carrying `StatusCode`, `Title`,
`Detail`, `Type`, `Instance` (correlation ID), and the raw body:

```go
_, err := client.Meters.Get(ctx, "missing")
var apiErr *openmeter.APIError
if errors.As(err, &apiErr) {
    log.Printf("status=%d detail=%s trace=%s", apiErr.StatusCode, apiErr.Detail, apiErr.Instance)
}
```

## Testing

Unit tests use `httptest` and make no network calls:

```bash
cd sdk/go
go test ./...
```

`TestLive` exercises the SDK against a real server. It is skipped unless
`OPENMETER_BASE_URL` is set, so it never runs during a normal `go test`. Set
`OPENMETER_TOKEN` for authenticated targets:

```bash
cd sdk/go
OPENMETER_BASE_URL=https://openmeter.cloud/api/v3 \
OPENMETER_TOKEN='om_your_token_here' \
  go test -run TestLive -v ./...
```

Verified against a local server (unauthenticated) and against
`https://dev.openmeter.cloud/api/v3` (bearer token).
