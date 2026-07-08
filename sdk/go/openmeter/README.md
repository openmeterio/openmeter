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

ctx := context.Background()

// Get one meter by ID.
m, err := client.Meters.Get(ctx, "01ABC...")

// List meters with pagination, sort, and filter (query-string params).
page, err := client.Meters.List(ctx, openmeter.MeterListParams{
Page:   &openmeter.PageParams{Size: openmeter.Int(20), Number: openmeter.Int(1)},
Sort:   []string{"created_at desc"},
Filter: &openmeter.MeterFilter{Key: &openmeter.StringFilter{Contains: openmeter.String("tokens")}},
})

// Iterate every meter across all pages (Go 1.23+ range-over-func).
for meter, err := range client.Meters.ListAll(ctx, openmeter.MeterListParams{}) {
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(meter.Key)
}

// Query a meter for usage (POST body). JSON result:
gran := openmeter.MeterQueryGranularityDay
res, err := client.Meters.Query(ctx, m.ID, openmeter.MeterQueryRequest{Granularity: &gran})

// Same query, CSV via content negotiation (Accept: text/csv):
csv, err := client.Meters.QueryCSV(ctx, m.ID, openmeter.MeterQueryRequest{Granularity: &gran})
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

Inject your own retry policy by building a client and passing its standard form:

```go
import "github.com/hashicorp/go-retryablehttp"

rc := retryablehttp.NewClient()
rc.RetryMax = 5
rc.CheckRetry = myCheckRetry // your own policy

client, err := openmeter.New(baseURL,
openmeter.WithToken("om_..."),
openmeter.WithHTTPClient(rc.StandardClient()),
)
```

Any transport works — a custom `http.RoundTripper`, a different retry library,
or a plain client to disable retries entirely:

```go
// No retries.
client, err := openmeter.New(baseURL, openmeter.WithHTTPClient(&http.Client{
Timeout: 10 * time.Second,
}))

// Custom transport (e.g. your own retrying RoundTripper, tracing, proxy).
client, err := openmeter.New(baseURL, openmeter.WithHTTPClient(&http.Client{
Transport: myRoundTripper{},
}))
```

### Third-party client libraries

The SDK builds its own `*http.Request` and calls `httpClient.Do(req)`. A
library's features (retries, circuit breaking, middleware) therefore apply only
if they run somewhere on that `Do` → `Transport.RoundTrip` path. Libraries fall
into three groups:

**A. Logic lives in a `RoundTripper` / `*http.Client` — composes directly.**

```go
// go-retryablehttp: retry logic sits behind a standard *http.Client, so
// StandardClient() carries the full retry behavior across.
rc := retryablehttp.NewClient()
rc.RetryMax = 5
client, _ := openmeter.New(baseURL, openmeter.WithHTTPClient(rc.StandardClient()))
```

`imroc/req` also belongs here: it is RoundTripper-based, so its client's
transport (`reqClient.GetClient()`) generally brings its middleware along.

**B. Library exposes `Do(*http.Request)` but is not an `*http.Client` — bridge
with a 3-line `RoundTripper` adapter.**

```go
// Heimdall (gojek/heimdall) runs retries and a circuit breaker inside its
// Do(*http.Request). Adapt it to a transport so http.Client.Do delegates to it:
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
meaningfully.**

`Resty` (`client.R().Get(url)`) and `Sling` (`sling.New().Get(url).Receive(...)`)
build and send their own requests from URLs/structs. The SDK has already built
the `*http.Request`, so their retry/middleware never sees it — they are
alternative SDKs, not transports. You can still hand the SDK their underlying
`*http.Client` for transport-level settings (connection, timeout, proxy, TLS),
but **not** their retry/middleware:

```go
// Resty: transport/timeout/TLS config only; Resty's retries do NOT apply.
rClient := resty.New().SetTimeout(15 * time.Second)
client, _ := openmeter.New(baseURL, openmeter.WithHTTPClient(rClient.GetClient()))
```

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
| `Meters.Get`            | `GET /openmeter/meters/{id}`        | Path parameter                                                                         |
| `Meters.List`           | `GET /openmeter/meters`             | Query-string serialization: deepObject `page`, form `sort`, nested deepObject `filter` |
| `Meters.ListAll`        | `GET /openmeter/meters` (paged)     | Auto-paginating iterator (`iter.Seq2[Meter, error]`, range-over-func)                   |
| `Meters.Query`          | `POST /openmeter/meters/{id}/query` | Request body                                                                           |
| `Meters.QueryCSV`       | `POST /openmeter/meters/{id}/query` | Content negotiation (JSON vs CSV)                                                      |
| `Meters.QueryCSVStream` | `POST /openmeter/meters/{id}/query` | Streaming CSV export (caller reads and closes the body)                                |

Buffered responses (JSON decoding and `QueryCSV`) are capped at 10 MiB to bound
memory. For CSV exports that may exceed that, use `QueryCSVStream`, which returns
an `io.ReadCloser` you consume and close without buffering the whole payload.

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
cd sdk/go/openmeter
go test ./...
```

`TestLive` exercises the SDK against a real server. It is skipped unless
`OPENMETER_BASE_URL` is set, so it never runs during a normal `go test`. Set
`OPENMETER_TOKEN` for authenticated targets:

```bash
cd sdk/go/openmeter
OPENMETER_BASE_URL=https://openmeter.cloud/api/v3 \
OPENMETER_TOKEN='om_your_token_here' \
  go test -run TestLive -v ./...
```

Verified against a local server (unauthenticated), `https://dev.openmeter.cloud/api/v3`
(bearer token), and Kong Konnect Dev (`https://us.api.konghq.tech/v3`, `kpat_` PAT).
