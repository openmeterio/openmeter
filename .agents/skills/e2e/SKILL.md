---
name: e2e
description: Write end-to-end tests for OpenMeter against a live server. Use when adding tests under e2e/ that exercise API endpoints over HTTP (v1 generated SDK or v3 SDK).
user-invocable: true
argument-hint: "[feature or scenario to test]"
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

# End-to-End Testing

You are helping the user write OpenMeter end-to-end tests that run against a live HTTP server with real dependencies (Postgres, Kafka, ClickHouse, Svix).

This is the black-box layer. Unlike the `/test` skill (which covers in-process unit/integration/service tests using `testutils.TestEnv` + `testutils.InitPostgresDB`), e2e tests hit the wire format: JSON in, JSON out, status codes, problem+json error bodies. Use this skill when the value of the test comes from exercising the HTTP contract, the OpenAPI binder, or cross-service behavior.

General test style from `AGENTS.md` and the `/test` skill still applies. Keep this skill as e2e-specific guidance, not a parallel set of test conventions.

## Two styles, same package

Both live in `e2e/` and share the build tag, environment, and skip-when-unset convention. Pick by what the endpoint under test offers:

| Style | When to use | Client | Reference |
|-------|-------------|--------|-----------|
| **v1 SDK** | Endpoint has generated Go SDK coverage (ingest, meters, subjects, customers, v1 plans, entitlements) | `initClient(t) *api.ClientWithResponses` from `setup_test.go` | `e2e_test.go`, `entitlement_test.go`, `multisubject_test.go` |
| **v3 SDK** | Endpoint lives under `/api/v3/...` — plans, addons, plan-addons, v3 meter query, billing invoices, customer credits, etc. | `newV3Client(t) *v3Client` from `v3helpers_test.go`. `v3Client` embeds the generated `*v3sdk.Client` (`api/v3/client`), so tests call SDK services directly — no per-endpoint wrappers | `plans_v3_test.go`, `addons_v3_test.go`, `planaddons_v3_test.go` |

Mixed files are fine — e.g., a v3 test that needs a v1 feature can call `initClient(t)` for the feature setup and `newV3Client(t)` for the assertion.

## Running tests

```bash
make etoe                 # Full suite (starts docker-compose stack + tests)
```

Prereqs: `make up` (or `docker compose -f e2e/docker-compose.infra.yaml up -d`) to bring up Postgres/Kafka/ClickHouse/Svix, plus an OpenMeter server reachable on `$OPENMETER_ADDRESS`.

`e2e/` is its own Go module (`github.com/openmeterio/openmeter/e2e`, `replace`s pinning it to the root module and `api/v3/client` at `../` and `../api/v3/client`) — it is never tagged or imported, and root `go build|test|vet ./...` no longer see it. Run it module-aware, from inside `e2e/` or with `-C e2e`:

Targeted run:

```bash
TZ=UTC OPENMETER_ADDRESS=http://localhost:8888 go test -C e2e -count=1 -v -run '^Test<Name>$' ./
```

Notes:

- No `-tags=dynamic` needed — e2e's import graph never reaches confluent-kafka-go (unlike the root module).
- Tests **skip automatically** when `OPENMETER_ADDRESS` is unset — the skip lives in `initClient` and `newV3Client`.
- `RUN_SLOW_TESTS=1` enables scenarios gated by `shouldRunSlowTests(t)` (`setup_test.go:26`).
- `-count=1` bypasses the go-test result cache; useful when iterating against a changing server.
- If `go`/`gofmt` are missing from the ambient shell, fall back to `nix develop --impure .#ci -c <command>` (see AGENTS.md).
- Run commands directly — do not wrap in `sh -lc`/`bash -lc`. For env vars, prefer `env KEY=VALUE <command>` or `KEY=VALUE <command>`.

## Shared conventions (both styles)

### Helper functions

Prefer a single explicit test body over single-use setup wrappers. Do not add helper functions for e2e setup, conversion, or assertions unless the helper is used by at least two tests in the same package or its name captures non-obvious domain semantics that would otherwise be easy to miss.

### Unique fixture keys

The docker-compose DB is shared across re-runs and parallel tests. Fixed keys collide. Always generate keys with a suffix:

```go
// v3 (v3helpers_test.go)
uniqueKey("prefix")                    // "prefix_<millis>_<rand>"
validPlanRequest("prefix")             // calls uniqueKey internally
```

For v1 tests, use `ulid.Make().String()` or a `fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())` to the same effect.

### Page size for list-to-find

Default server page size is 20. When a test creates a fixture and then lists to locate it, bump the page size or the fresh row may sit past page 1 on a busy DB:

```go
// v3
c.Plans.List(t.Context(), v3sdk.PlanListParams{
    Page: &v3sdk.PageParams{Size: lo.ToPtr(1000)},
})
// v1: pass page_size via the generated params struct
```

### Decimal normalization

The server trims trailing zeros and canonicalizes decimals on round-trip: `"0.10"` comes back as `"0.1"`. Parse as float or use the normalized form; never assert on the raw input string.

### Per-request timeout

The generated v3 SDK (`api/v3/client`, `defaultRequestTimeout` in `transport.go`) applies a 30s deadline to any call whose context carries none. Tests just pass `t.Context()` — no per-call timeout plumbing needed. The one exception is `c.doMalformedRequest(...)`, the raw HTTP escape hatch, which builds its own inline 30s context since it bypasses the SDK transport entirely. A server-side hang still surfaces in seconds instead of eating the whole 10-minute `go test` deadline.

### Context

Use `t.Context()` in e2e tests too — it ties cancellation to the test harness and matches the rest of the repo.

## v1 SDK style

The generated client at `api/client/go` exposes `<Endpoint>WithResponse` methods that return typed response structs with `StatusCode()`, `JSON200`, `JSON201`, etc. Shared helpers in `e2e/helpers.go` wrap the common multi-step flows (create customer + subject, lookup meter by slug).

```go
func TestIngest(t *testing.T) {
    client := initClient(t)

    resp, err := client.UpsertSubjectWithResponse(t.Context(), api.UpsertSubjectJSONRequestBody{
        api.SubjectUpsert{Key: "customer-1"},
    })
    require.NoError(t, err)
    require.Equal(t, http.StatusOK, resp.StatusCode())
}
```

Patterns worth reusing:

- Error body access: `string(resp.Body)` — the generated client keeps the raw bytes for diagnostics.
- Eventual consistency: `assert.EventuallyWithT(...)` when the test writes an event and then queries the meter (ingestion is async through Kafka).
- Error shape on 4xx: the generated SDK parses into `resp.ApplicationproblemJSON400.Extensions.ValidationErrors[N].Code` for v1 domain validation — see the older `productcatalog_test.go` for examples.

## v3 SDK style

Use `newV3Client(t)` for v3 endpoints. `v3Client` embeds the generated `v3sdk "github.com/openmeterio/openmeter/api/v3/client"` SDK (Go package name `openmeter`), so tests call the service methods directly — `c.Plans.Create(...)`, `c.Addons.Publish(...)`, `c.Customers.Credits.Grants.Create(...)`, `c.Meters.Query(...)`. There are no per-endpoint wrapper methods to maintain; the harness only adds exact-2xx status pinning on top of the SDK and a raw HTTP escape hatch for requests the typed SDK cannot represent.

```go
func TestV3<Entity><Behavior>(t *testing.T) {
    c := newV3Client(t)

    body := validPlanRequest("descriptive_prefix")
    // mutate body as needed...

    plan, err := c.Plans.Create(t.Context(), body)
    c.requireStatus(http.StatusCreated, err)
    require.NotNil(t, plan)

    assert.Equal(t, v3sdk.PlanStatusDraft, plan.Status)

    // Error case: assert the exact status and problem shape.
    _, err = c.Plans.Create(t.Context(), invalidBody)
    problem := requireProblem(t, err, http.StatusBadRequest)
    assertValidationCode(t, problem, "plan_phase_duplicated_key")
}
```

Two call shapes to know:
- **Success**: `c.requireStatus(want int, err error)` asserts `require.NoError(t, err)` and then that the exact HTTP status captured by an injected `RoundTripper` equals `want` — this is how tests distinguish 201 vs 200 vs 204 without the SDK response type carrying a status field. It fails `t` directly; never call it inside `assert.EventuallyWithT` or any helper that receives a `*assert.CollectT` — use `require.NoError(collect, err)` there instead (see `queryMeterV3` and the v1 polling patterns in `e2e_test.go`).
- **Error**: every non-2xx response surfaces as `*v3sdk.APIError` (`v3sdk.AsAPIError(err)`). `requireProblem(t, err, wantStatus)` asserts the status and returns the decoded `*v3Problem` for the three assertion helpers below (`assertValidationCode`, `assertProblemDetail`, `assertInvalidParameterRule`).
- `c.doMalformedRequest(method, path, body) (int, []byte, *v3Problem)` is the raw HTTP escape hatch — use it only for values the typed SDK cannot represent at all, such as an unparseable query parameter or malformed timestamp string. Everything else goes through the SDK.
- `queryMeterV3(t, meterID, v3sdk.MeterQueryRequest) (*v3sdk.MeterQueryResult, error)` builds its own client and returns the raw error instead of failing `t`, so it's safe to call inside `assert.EventuallyWithT` while waiting for ingested events to land.

Delete/detach calls (`c.Plans.Delete`, `c.Addons.Delete`, plan-addon detach) have no response body, so they just return `error` — pin the status the same way: `err := c.Plans.Delete(t.Context(), id); c.requireStatus(http.StatusNoContent, err)`.

SDK unions (`Price`, `RateCardEntitlement`, `UpdateInvoiceLine`, ...) are discriminated types that must be built with their `XFromY(...)` package constructors — e.g. `lo.Must(v3sdk.PriceFromPriceFlat(v3sdk.PriceFlat{Amount: "10"}))` — not by assigning fields to a zero-valued struct. A zero union value marshals as JSON `null` and the server rejects it. The constructor sets the discriminator field; read a union back with its `AsX()` methods (they return pointers) or check the exported `.Type` field directly, as `assertUnitPriceAmount` does.

The constructor only stamps the discriminator of the union value it builds — variant structs NESTED inside the payload are marshaled as-is. Where a variant struct is used directly as a field (e.g. `PriceTier.UnitPrice *PriceUnit` inside a graduated/volume price), set its `Type` explicitly (`v3sdk.PriceUnit{Type: v3sdk.PriceTypeUnit, ...}`) or the wire body carries `"type":""` and the request fails schema validation with a 400 (see `validGraduatedRateCard`).

Testify trap: SDK integer fields (`Plan.Version`, `Addon.Version`, pagination meta counts, etc.) are typed `int64`/similar, not `int`. `assert.Equal(t, 1, plan.Version)` fails on the type mismatch even when the values match. Use `assert.EqualValues` or a matching literal type instead (see `plans_v3_test.go`'s `assert.EqualValues(t, 1, plan.Version)`).

Extending the harness:
- New endpoint family → call the matching `api/v3/client` SDK service directly; no wrapper method to add.
- New fixture kind → add a `valid<Thing>Request("prefix")` builder that internally calls `uniqueKey` so callers never have to think about collisions, and constructs any union fields via their `XFromY(...)` constructor.
- New assertion shape → add `assert<Shape>(t, problem, ...)` next to the existing helpers.

## Error-shape triage (v3)

v3 handlers return **three** distinct error shapes on 4xx responses. The harness parses all three into the same `*v3Problem`. Pick the assertion helper by **shape**, not by scenario intent.

| Shape | Produced by | Example | Helper |
|-------|-------------|---------|--------|
| Domain validation | `commonhttp.HandleIssueIfHTTPStatusKnown` — any handler that returns `models.ValidationIssue`s | `extensions.validationErrors[].code` = `"plan_phase_duplicated_key"` | `assertValidationCode(t, problem, "<code>")` |
| API error with free-text `Detail` | `api/v3/apierrors/errors.go` — `BaseAPIError` and typed errors like `FeatureNotFoundError` | `"only Plans in [draft scheduled] can be updated"`, `"feature with ID … not found"` | `assertProblemDetail(t, problem, "<substring>")` |
| Schema / request binder | oapi-codegen binder (fires before any handler) | `invalid_parameters[].rule` = `"min_items"`, `"required"`, `"enum"` | `assertInvalidParameterRule(t, problem, "<rule>")` |

You cannot predict which shape a new check uses until you see the response. **Write the test, run it once, inspect the raw problem via the `"problem: %+v"` failure message, then pick the helper.** If `extensions.validationErrors` is empty but `Detail` carries the reason, switch to `assertProblemDetail`. If neither is set but `invalid_parameters` is populated, switch to `assertInvalidParameterRule`.

A word of caution on `assertProblemDetail`: the substring you match is free-text server output. It's a fragile assertion — any edit to the error message will break the test. Use it only when the other two shapes don't apply, and keep the substring short and distinctive.

## Validation moments: create vs. publish vs. get

Some v3 entities (plans, addons, plan-addons today) support **draft** lifecycle states. Not every defect is rejected at create — several are accepted as drafts, surface as `validation_errors` on GET, and fire only at publish. Three moments, three assertion sites:

1. **Create-time** (`POST /<resource>` → 400) — schema errors (min_items, required) and a small set of domain checks.
2. **Get-time** (`GET /<resource>/{id}` → 200 with `validation_errors` populated on the body) — soft surface for UIs.
3. **Publish-time** (`POST /<resource>/{id}/publish` → 400) — most domain rules land here.

Before asserting 400-at-create, **run the request**. If you get 201, pivot to the draft-with-errors shape (see `TestV3PlanInvalidDraftLifecycle` for the canonical three-step flow: create draft → GET shows errors → publish rejects with the same code → fix via PUT → publish succeeds).

Rule of thumb: the moment a check fires is a server-side choice that can shift between releases. Pin tests to one moment and you risk spurious failures when the server tightens or loosens. If a check is important, exercise both the draft-with-errors GET and the publish rejection; it costs little extra and survives reasonable server evolution.

## Patterns

### Lifecycle (ordered subtests sharing state)

When the scenario reads as "create → update → publish → archive → delete", group the steps as `t.Run` subtests under a single outer-test client. Subtest names describe the **step**, not the expected status.

Reference: `e2e/plans_v3_test.go` `TestV3PlanLifecycle`, `e2e/addons_v3_test.go` `TestV3Addon`.

```go
func TestV3<Entity>Lifecycle(t *testing.T) {
    c := newV3Client(t)

    createBody := valid<Entity>Request("lifecycle")
    var entityID string

    t.Run("Should create the entity in draft status", func(t *testing.T) {
        e, err := c.<Entities>.Create(t.Context(), createBody)
        c.requireStatus(http.StatusCreated, err)
        require.NotNil(t, e)
        entityID = e.ID
    })

    t.Run("Should publish the entity", func(t *testing.T) {
        require.NotEmpty(t, entityID)
        e, err := c.<Entities>.Publish(t.Context(), entityID)
        c.requireStatus(http.StatusOK, err)
        assert.Equal(t, v3sdk.<Entity>StatusActive, e.Status)
    })

    // ... archive, delete, etc.
}
```

### Table-driven validation (independent subtests)

For validation matrices (status × status, instance-type × quantity, etc.), each row gets a fresh client. This scopes `require.X` failures to the row, not the outer table.

Reference: `e2e/planaddons_v3_test.go` `TestV3PlanAddonAttachStatusMatrix`.

```go
func TestV3<Something>Matrix(t *testing.T) {
    cases := []struct {
        name             string
        mutate           func(*v3sdk.Create<X>Request)
        expectedStatus   int
        expectedCode     string // domain-validation code; empty for 2xx or non-PC shapes
        expectedDetailIn string // substring of Detail; alternative to expectedCode
    }{
        {name: "valid baseline → 201", mutate: func(*v3sdk.Create<X>Request) {}, expectedStatus: http.StatusCreated},
        // ... more rows
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            c := newV3Client(t)

            body := valid<X>Request("matrix")
            tc.mutate(&body)

            got, err := c.<Xs>.Create(t.Context(), body)

            switch {
            case tc.expectedCode != "":
                problem := requireProblem(t, err, tc.expectedStatus)
                assertValidationCode(t, problem, tc.expectedCode)
            case tc.expectedDetailIn != "":
                problem := requireProblem(t, err, tc.expectedStatus)
                assertProblemDetail(t, problem, tc.expectedDetailIn)
            default:
                c.requireStatus(tc.expectedStatus, err)
                require.NotNil(t, got)
            }
        })
    }
}
```

### Eventual consistency (v1 ingestion flow)

Kafka is in the path for ingestion. Don't assert the meter value immediately after ingest — wrap the read in `assert.EventuallyWithT` with a reasonable ceiling.

Reference: `e2e/e2e_test.go` `TestIngest`.

For async billing flows, make timeout failures self-diagnosing. Log the created resource IDs that connect the test to service logs (customer, subscription, invoice, pending line, charge when available), and during the polling loop log the externally visible state as JSON, such as invoice list entries and customer charge statuses. The E2E CI job streams docker compose logs during each test phase and archives per-service docker logs before each compose teardown/restart; keep those artifacts broad enough that a failed async hop can be correlated from the test output to the relevant service log.

## Testing conventions

- **`require` vs `assert`**: `require` for fatal preconditions (no point continuing), `assert` for soft per-field checks. In table rows, branch on the expected outcome (`requireProblem` for the error rows, `c.requireStatus` for the 2xx rows) rather than asserting status and body separately — see `TestV3PlanAddonAttachStatusMatrix`. Reserve `require` for lifecycle tests where later steps depend on the earlier status being correct.
- **`t.Helper()`** in every helper function — so `require` failures blame the caller.
- **`t.Context()`** over `context.Background()` — cancellation ties to the test.
- **Test naming**: when both v1 and v3 tests live in the same package, prefix v3 tests with `TestV3` to disambiguate (`TestV3PlanLifecycle`, `TestV3AddonVersioningAndAutoArchive`). For single-style packages, the `V3` prefix is unnecessary.
- **Client lifetime**: one `newV3Client(t)` at the top for lifecycle tests (shared state); one per `t.Run` for table-driven validation (independent rows).
- **Parallelism**: the current suite does not opt in to `t.Parallel()`. Fixtures are unique-keyed, so it's safe in principle — but the shared DB means intermittent list-ordering flakiness is possible. Opt in deliberately, row by row, not globally.

## Gotchas worth knowing before you write a new v3 test

Captured from real live-server runs. Most are v3-wide; a few call out plans/addons specifically because they're the only v3 surface today that exposes drafts.

- **Deep-object query params** like `?page[number]=1&page[size]=20` are encoded by `url.Values.Encode()` with percent-encoded brackets; the server decodes them back. Both forms work.
- **Some delete paths return 400 `"plan is deleted"` rather than 404** for entities in the deleted state. Don't assume 404 by default.

## Further reading

- **`AGENTS.md`** — repo-wide conventions: toolchain fallback, build tag, `POSTGRES_HOST` for in-process tests, general coding rules.
- **Generated v3 SDK** — `api/v3/client` (Go package `openmeter`; regenerated by `make gen-api`; don't edit). This is the SDK `newV3Client(t)` embeds and the only client v3 e2e tests use.
- **Generated v3 server types** — `api/v3/api.gen.go` (regenerated by `make gen-api`; don't edit). No longer used by e2e fixtures; the v3 harness and all fixture builders build SDK types (`v3sdk.*`) directly.
