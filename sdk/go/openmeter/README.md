# OpenMeter Go SDK (baseline)

A hand-written, idiomatic Go SDK for the OpenMeter v3 API.

> **Status: baseline / reference.** It implements the meter endpoints plus a
> nested plan-addons resource as worked examples of the target SDK shape. This
> shape is intended to be reproduced by a TypeSpec emitter. It is not yet a
> complete SDK.

## Design principles

- **Standard-library public surface.** The `Client` exposes only `net/http`,
  `context`, typed request/response structs, and a typed `APIError`. No
  third-party type appears in the public API.
- **Swappable transport.** Requests run through an `*http.Client`. The default
  one retries with exponential backoff (via an internal dependency hidden behind
  the `*http.Client` seam). Inject your own with `WithHTTPClient` to own retry,
  timeout, proxy, TLS, and tracing behavior.
- **Resource-grouped operations.** Operations hang off resource sub-clients,
  e.g. `client.Meters.List(...)`. Nested sub-resources take the parent ID as the
  first argument, e.g. `client.PlanAddons.List(ctx, planID, ...)`, mirroring the
  tag-grouped shape a TypeSpec emitter produces.
- **Auth survives injection.** The bearer token is set during request
  construction, not via a transport wrapper, so it is applied even when you
  inject your own `*http.Client`.

## Install

```bash
go get github.com/openmeterio/openmeter/sdk/go/openmeter
```

```go
import "github.com/openmeterio/openmeter/sdk/go/openmeter"
```

The import base matches the package name, so no import alias is needed —
callers use `openmeter.New(...)` directly.

## Usage

The base URL must include the API version prefix, and the prefix differs by
deployment:

| Deployment                        | Base URL                                               |
|-----------------------------------|--------------------------------------------------------|
| Local server                      | `http://127.0.0.1:8888/api/v3`                         |
| OpenMeter Cloud                   | `https://openmeter.cloud/api/v3`                       |
| Kong Konnect (metering & billing) | `https://<region>.api.konghq.com/v3` (e.g. `us`, `eu`) |

Konnect mounts the same API under `/v3` (not `/api/v3`); the SDK preserves that
prefix, so only the base URL changes. Authenticate Konnect with a personal
access token (`kpat_...`) via `WithToken` — the same bearer mechanism as an
OpenMeter Cloud token.

```go
client, err := openmeter.New("https://openmeter.cloud/api/v3", openmeter.WithToken("om_..."))
if err != nil {
	log.Fatal(err)
}
```

Full, runnable usage for every operation (create, get, update, delete, list,
paginate, query, CSV stream, error handling) lives in compiler-checked
[`Example` functions](./example_test.go). Browse them with:

```bash
go doc -all github.com/openmeterio/openmeter/sdk/go/openmeter
```

## Custom HTTP client and retries

Retries and transport are a single, injectable concern: the SDK exposes only
`*http.Client`, and `WithHTTPClient` replaces it wholesale. There is
deliberately no retry-specific option — a retry-library type on the public API
would leak a third-party dependency and break the stdlib-only contract.

By default (no `WithHTTPClient`), requests go through an internal
`go-retryablehttp` client that retries on 5xx and connection errors, but only
for idempotent methods (`GET`, `HEAD`); non-idempotent methods are never retried
once a response arrives, so a 5xx on a write can't be silently duplicated.

Timeouts come from the **request context**, not `http.Client.Timeout` (which
would also cut off streamed body reads). When a buffered call's context carries
no deadline, the SDK applies a default 30s bound covering the whole call
(retries included) so it can't hang forever; pass your own context deadline to
override. `QueryCSVStream` intentionally applies no default bound — a stream is
governed solely by the context you pass, so it can run as long as needed.

Inject your own client to control retries and transport. Compiler-checked
examples — a custom `go-retryablehttp` policy, a plain client with no retries,
and a custom `http.RoundTripper` — are the `ExampleWithHTTPClient*` functions in
[`example_test.go`](./example_test.go).

### Third-party client libraries

The SDK builds its own `*http.Request` and calls `httpClient.Do(req)`. A
library's features (retries, circuit breaking, middleware) therefore apply only
if they run somewhere on that `Do` → `Transport.RoundTrip` path. Libraries fall
into three groups. The snippets below reference external libraries for
illustration and are not compiled (this SDK depends on none of them).

**A. Logic lives in a `RoundTripper` / `*http.Client` — composes directly.**
`go-retryablehttp` exposes its retries through `StandardClient()`; see the
compiler-checked `ExampleWithHTTPClient` in [`example_test.go`](./example_test.go).
`imroc/req` also belongs here: it is RoundTripper-based, so its client's
transport (`reqClient.GetClient()`) generally brings its middleware along.

**B. Library exposes `Do(*http.Request)` but is not an `*http.Client` — bridge
with a 3-line `RoundTripper` adapter.** Heimdall (`gojek/heimdall`) runs retries
and a circuit breaker inside its `Do(*http.Request)`; adapt it to a transport so
`http.Client.Do` delegates to it:

```go
type heimdallRT struct{ c heimdall.Client }

func (h heimdallRT) RoundTrip(r *http.Request) (*http.Response, error) {
	return h.c.Do(r)
}

hc := &http.Client{Transport: heimdallRT{myHeimdallClient}}
client, _ := openmeter.New(baseURL, openmeter.WithHTTPClient(hc))
```

This adapter is the general escape hatch: anything exposing
`Do(*http.Request) (*http.Response, error)` becomes a transport this way.

**C. Library owns request construction (fluent builders) — cannot be injected
meaningfully.** `Resty` (`client.R().Get(url)`) and `Sling`
(`sling.New().Get(url).Receive(...)`) build and send their own requests from
URLs/structs. The SDK has already built the `*http.Request`, so their
retry/middleware never sees it — they are alternative SDKs, not transports. You
can still pass their underlying `*http.Client` (`resty.New().GetClient()`) for
transport-level settings (connection, timeout, proxy, TLS), but **not** their
retry/middleware.

Summary:

| Library            | Features apply?   | How                                     |
|--------------------|-------------------|-----------------------------------------|
| `go-retryablehttp` | ✅                 | `StandardClient()`                      |
| `imroc/req`        | ✅ (mostly)        | `GetClient()` (RoundTripper-based)      |
| `gojek/heimdall`   | ✅                 | 3-line `RoundTripper` adapter (group B) |
| `go-resty/resty`   | ⚠️ transport only | `GetClient()` — no retry/middleware     |
| `dghubble/sling`   | ❌                 | request builder, not a transport        |

Note: injecting a client replaces the SDK's default idempotent-only retry guard,
so if you opt into retries you own the idempotency policy. The bearer token is
still applied (it is set during request construction, not in the transport).

## Implemented endpoints

| Method                  | HTTP                                | Demonstrates                                                                           |
|-------------------------|-------------------------------------|----------------------------------------------------------------------------------------|
| `Meters.Create`         | `POST /openmeter/meters`            | Request body, 201 Created response                                                     |
| `Meters.Get`            | `GET /openmeter/meters/{id}`        | Path parameter                                                                         |
| `Meters.Update`         | `PUT /openmeter/meters/{id}`        | Path parameter + request body, 200 response                                            |
| `Meters.Delete`         | `DELETE /openmeter/meters/{id}`     | 204 No Content (returns only an error)                                                 |
| `Meters.List`           | `GET /openmeter/meters`             | Query-string serialization: deepObject `page`, form `sort`, nested deepObject `filter` |
| `Meters.ListAll`        | `GET /openmeter/meters` (paged)     | Auto-paginating iterator (`iter.Seq2[Meter, error]`, range-over-func)                  |
| `Meters.Query`          | `POST /openmeter/meters/{id}/query` | Request body                                                                           |
| `Meters.QueryCSV`       | `POST /openmeter/meters/{id}/query` | Content negotiation (JSON vs CSV)                                                      |
| `Meters.QueryCSVStream` | `POST /openmeter/meters/{id}/query` | Streaming CSV export (caller reads and closes the body)                                |

Buffered responses (JSON decoding and `QueryCSV`) are capped at 10 MiB to bound
memory. For CSV exports that may exceed that, use `QueryCSVStream`, which returns
an `io.ReadCloser` you consume and close without buffering the whole payload.

`PlanAddons` is a **nested sub-resource** under `/plans/{planId}/addons`; every
operation takes the parent plan ID as its first argument.

| Method               | HTTP                                           | Demonstrates                                               |
|----------------------|------------------------------------------------|------------------------------------------------------------|
| `PlanAddons.List`    | `GET /openmeter/plans/{planId}/addons`         | Nested collection; parent ID as first arg; page pagination |
| `PlanAddons.ListAll` | `GET /openmeter/plans/{planId}/addons` (paged) | Nested auto-paginating iterator (shared `paginate[T]`)     |
| `PlanAddons.Create`  | `POST /openmeter/plans/{planId}/addons`        | Nested create, 201 Created                                 |
| `PlanAddons.Get`     | `GET .../addons/{planAddonId}`                 | Two path parameters (parent + child), each encoded once    |
| `PlanAddons.Update`  | `PUT .../addons/{planAddonId}`                 | Two path parameters + request body                         |
| `PlanAddons.Delete`  | `DELETE .../addons/{planAddonId}`              | 204 No Content (returns only an error)                     |

## Errors

Non-2xx responses return a typed `*openmeter.APIError` carrying `StatusCode`,
`Title`, `Detail`, `Type`, `Instance` (correlation ID), and the raw body. Match
it with `errors.As`; see `ExampleAPIError` in
[`example_test.go`](./example_test.go).

## Testing

Unit tests use `httptest` and make no network calls:

```bash
cd sdk/go/openmeter
go test ./...
```

The `TestLive*` tests exercise the SDK against a real server. They are skipped
unless `OPENMETER_BASE_URL` is set, so they never run during a normal `go test`.
Set `OPENMETER_TOKEN` for authenticated targets:

```bash
cd sdk/go/openmeter
OPENMETER_BASE_URL=https://openmeter.cloud/api/v3 \
OPENMETER_TOKEN='om_your_token_here' \
  go test -run TestLive -v ./...
```

Additional gates:

- **Mutating tests** (`TestLiveMetersReadWrite`, `TestLivePlanAddonsReadWrite`)
  write to the target, so they are additionally gated behind
  `OPENMETER_LIVE_MUTATE=1` and stay skipped otherwise.
- **Plan-addon tests** need a plan to operate on (the SDK does not list plans):
  set `OPENMETER_LIVE_PLAN_ID`. The read-write cycle additionally needs
  `OPENMETER_LIVE_ADDON_ID` (a published, unassociated add-on) and
  `OPENMETER_LIVE_PLAN_PHASE` (the phase the add-on becomes available from), and
  the plan must be in `draft`.

```bash
cd sdk/go/openmeter
OPENMETER_BASE_URL=https://us.api.konghq.tech/v3 OPENMETER_TOKEN='kpat_...' \
OPENMETER_LIVE_MUTATE=1 \
OPENMETER_LIVE_PLAN_ID='01...' OPENMETER_LIVE_ADDON_ID='01...' OPENMETER_LIVE_PLAN_PHASE='default' \
  go test -run TestLivePlanAddons -v -count=1 ./...
```

Verified against a local server (unauthenticated), `https://dev.openmeter.cloud/api/v3`
(bearer token), and Kong Konnect Dev (`https://us.api.konghq.tech/v3`, `kpat_` PAT).
