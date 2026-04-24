# e2e

<!-- archie:ai-start -->

> End-to-end test suite that runs against a live OpenMeter stack (openmeter + sink-worker + all infra) bootstrapped via Docker Compose. Tests call the real HTTP API through the generated Go client and v3 raw HTTP client, asserting full stack behavior including async Kafka→ClickHouse pipelines.

## Patterns

**OPENMETER_ADDRESS guard** — Every test file calls initClient(t) or newV3Client(t), both of which call t.Skip when OPENMETER_ADDRESS env var is not set. New tests must never bypass this; tests must not run without a live server. (`address := os.Getenv("OPENMETER_ADDRESS"); if address == "" { t.Skip(...) }`)
**assert.EventuallyWithT for async pipelines** — Any assertion on ClickHouse data (meter queries, entitlement balances) must use assert.EventuallyWithT with a 1-minute timeout and 1-second interval — Kafka→sink-worker→ClickHouse is async. Never assert synchronously after ingest. (`assert.EventuallyWithT(t, func(t *assert.CollectT) { /* query meter */ }, time.Minute, time.Second)`)
**v3Client typed wrapper (plans/addons/planaddons)** — v3 API tests use *v3Client from v3helpers_test.go, not the generated SDK. Methods return (statusCode, *T, *v3Problem). Assertions check statusCode first, then the typed body. New v3 tests must follow the same (status, body, problem) triple pattern. (`status, plan, problem := c.CreatePlan(body); require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)`)
**unique keys via timestamp/ULID to avoid shared-DB collisions** — Feature keys, plan keys, and customer keys use a timestamp or ULID suffix to avoid conflicts when tests run against a shared persistent DB. Never use hardcoded keys for entities that cannot be idempotently re-created. (`randKey := fmt.Sprintf("entitlement_uc_test_feature_%d", time.Now().Unix())`)
**slow-test guard for minute-resolution assertions** — Tests that sleep >5 s (waiting for a new minute for entitlement reset or credit granularity) must be gated behind shouldRunSlowTests(t), which checks RUN_SLOW_TESTS env var. Gate new slow tests the same way. (`if !shouldRunSlowTests(t) { t.Skip("Skipping slow test...") }`)
**CreateCustomerWithSubject helper for customer+subject setup** — Tests that need a customer with a subject mapping call helpers.go:CreateCustomerWithSubject(t, client, customerKey, subjectKey). Do not inline this boilerplate; the helper enforces the create-then-upsert-subject order required by the API. (`cust := CreateCustomerWithSubject(t, client, "cust-key", "subj-key")`)
**assertProblemDetail / assertValidationCode / assertInvalidParameterRule for error assertions** — Negative-path tests must use the v3helpers assertion helpers to verify error shapes. assertValidationCode checks extensions.validationErrors[].code; assertInvalidParameterRule checks schema-layer invalid_parameters[].rule; assertProblemDetail checks a substring in Detail. (`assertValidationCode(t, problem, "plan_phase_duplicated_key")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `setup_test.go` | Provides initClient (v1 SDK client) and shouldRunSlowTests. Single source of truth for test bootstrapping guards. | Adding test setup logic here that skips rather than fails will silently ignore tests in CI when the env var is missing. |
| `v3helpers_test.go` | Defines *v3Client (raw HTTP), v3Problem (RFC 7807 parse), decodeTyped generic, all typed CRUD wrappers for plans/addons/planaddons, validXxx fixture builders, and assertion helpers (assertProblemDetail, assertValidationCode, assertInvalidParameterRule). | The v3Client.do method uses t.Context() timeout (v3RequestTimeout). Do not bypass this with context.Background(). |
| `helpers.go` | CreateCustomerWithSubject (creates customer then upserts subject), GetMeterIDBySlug, QueryMeterV3 (raw HTTP v3 meter query). Shared cross-test helpers. | QueryMeterV3 uses raw http.DefaultClient and reads OPENMETER_ADDRESS directly — it will skip if the address is empty. |
| `config.yaml` | OpenMeter config injected into the Docker Compose stack. credits.enabled=false — tests in productcatalog_test.go (TestSettlementMode) depend on this being false. Meter definitions here must exist before any test ingests to those slugs. | Adding a new meter slug in a test without adding the corresponding meter definition here will cause the test to silently produce no data. |
| `ledger_backfill_test.go` | Tests the openmeter-jobs ledger backfill-accounts command by running docker compose exec and then verifying ledger_customer_accounts rows via direct pgxpool queries against the e2e Postgres. | ensureLocalComposeBackfillSupport skips when OPENMETER_ADDRESS is not the localhost:38888 compose stack — this test cannot run against remote envs. |
| `docker-compose.infra.yaml + docker-compose.openmeter.yaml` | Compose files for infra (Kafka, ClickHouse, Redis, Postgres) and openmeter/sink-worker services. The Makefile always combines both sets. Port range 30000+ to avoid conflicts with main dev environment. | docker-compose.openmeter-local.yaml overrides image: with build: .. so local tests use the locally built binary, not a released image. |
| `Makefile` | test-local target: tears down, brings up compose stack, waits for sink-worker /healthz, then runs go test with TZ=UTC OPENMETER_ADDRESS=http://localhost:38888. | TZ=UTC is essential for deterministic time-window assertions. Do not run e2e tests without it. |

## Anti-Patterns

- Asserting meter query results synchronously after ingest without EventuallyWithT — data flows async through Kafka and ClickHouse.
- Using hardcoded feature/plan/customer keys shared across test functions — causes 409 or silent state pollution in a shared persistent DB.
- Importing app/common or any Wire provider package — e2e tests must only use the HTTP client, never internal domain packages (except for constant types from openmeter/ledger etc.).
- Running tests without OPENMETER_ADDRESS set and expecting them to do anything other than skip.
- Adding new meters to tests without a corresponding entry in e2e/config.yaml — the sink-worker validates events against known meters and will silently drop events to unknown meters.

## Decisions

- **Two client layers: generated api.ClientWithResponses for v1 and a hand-rolled v3Client for v3.** — The v3 API is still evolving and the generated client requires heavy union-type machinery; a thin hand-rolled wrapper returning (status, *T, *v3Problem) gives tests direct control over HTTP-level assertions without fighting the generated code.
- **credits.enabled=false in e2e config.yaml.** — E2e tests for settlement-mode validation depend on credits being off to assert that credit_only settlement mode is rejected. This is explicitly documented in the config comment.
- **Separate docker-compose.openmeter-local.yaml that overrides image: with build: for local runs.** — Makes 'make test-local' always test the locally built binary while CI can swap in a published image by using docker-compose.openmeter-latest.yaml without changing test code.

## Example: Writing a new v3 resource lifecycle test (create → assert status → use resource ID in subsequent steps)

```
func TestV3MyResource(t *testing.T) {
	c := newV3Client(t) // skips if OPENMETER_ADDRESS not set

	var resourceID string

	t.Run("Should create the resource", func(t *testing.T) {
		status, res, problem := c.CreateMyResource(validMyResourceRequest("test_key"))
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, res)
		resourceID = res.Id
	})

	t.Run("Should get the resource", func(t *testing.T) {
		require.NotEmpty(t, resourceID)
		status, res, problem := c.GetMyResource(resourceID)
// ...
```

<!-- archie:ai-end -->
