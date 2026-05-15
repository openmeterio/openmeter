# e2e

<!-- archie:ai-start -->

> End-to-end test suite that runs against a live OpenMeter stack bootstrapped via Docker Compose. Tests call the real HTTP API through the generated Go client (v1) and a hand-rolled v3Client (v3), asserting full-stack behavior including async Kafka→ClickHouse pipelines. No internal domain packages are imported — only HTTP clients and public type definitions.

## Patterns

**OPENMETER_ADDRESS env-gate** — Every test file calls initClient(t) or newV3Client(t), both of which call t.Skip when OPENMETER_ADDRESS env var is not set. New tests must never bypass this guard. (`address := os.Getenv("OPENMETER_ADDRESS"); if address == "" { t.Skip("OPENMETER_ADDRESS not set") }`)
**assert.EventuallyWithT for async pipeline assertions** — Any assertion on ClickHouse data (meter queries, entitlement balances) must use assert.EventuallyWithT with a 1-minute timeout and 1-second interval. Kafka→sink-worker→ClickHouse is async; never assert synchronously after ingest. (`assert.EventuallyWithT(t, func(t *assert.CollectT) { resp, _ := client.QueryMeterWithResponse(...) }, time.Minute, time.Second)`)
**v3Client typed wrapper (status, body, problem) triple** — v3 API tests use *v3Client from v3helpers_test.go, not the generated SDK. Methods return (statusCode, *T, *v3Problem). Assertions check statusCode first, then the typed body. New v3 tests must follow the same triple pattern. (`status, plan, problem := c.CreatePlan(body); require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)`)
**Unique keys via timestamp/ULID to avoid shared-DB collisions** — Feature keys, plan keys, addon keys, and customer keys use a timestamp or ULID suffix (via uniqueKey() helper). Never use hardcoded keys for entities that cannot be idempotently re-created. (`randKey := fmt.Sprintf("entitlement_uc_test_feature_%d", time.Now().Unix())`)
**shouldRunSlowTests guard for minute-resolution assertions** — Tests that sleep >5s (waiting for a new minute for entitlement reset or credit granularity) must be gated behind shouldRunSlowTests(t), which checks RUN_SLOW_TESTS env var. (`if !shouldRunSlowTests(t) { t.Skip("Skipping slow test...") }`)
**assertProblemDetail / assertValidationCode / assertInvalidParameterRule for error paths** — Negative-path tests must use the v3helpers assertion helpers. assertValidationCode checks extensions.validationErrors[].code; assertInvalidParameterRule checks schema-layer invalid_parameters[].rule; assertProblemDetail checks a substring in Detail. (`assertValidationCode(t, problem, "plan_phase_duplicated_key")`)
**Meter definitions in e2e/config.yaml before test ingest** — Every event type ingested by tests must have a corresponding meter slug defined in e2e/config.yaml. The sink-worker validates events against known meters and silently drops events to unknown meters. (`meters:\n  - slug: plan_meter\n    eventType: plan_meter\n    aggregation: SUM`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `setup_test.go` | Provides initClient (v1 SDK client) and shouldRunSlowTests. Single source of truth for test bootstrapping and env-var guards. | Adding test setup logic here that skips rather than fails will silently ignore tests in CI when the env var is missing. |
| `v3helpers_test.go` | Defines *v3Client (raw HTTP), v3Problem (RFC 7807 parse), decodeTyped generic, all typed CRUD wrappers for plans/addons/planaddons, validXxx fixture builders, and assertion helpers. | v3Client.do uses t.Context() timeout (v3RequestTimeout). Do not bypass with context.Background(). uniqueKey() helper is defined here — use it for all entity keys. |
| `helpers.go` | CreateCustomerWithSubject (creates customer then upserts subject), GetMeterIDBySlug, QueryMeterV3 (raw HTTP v3 meter query). Shared cross-test helpers. | QueryMeterV3 uses raw http.DefaultClient and reads OPENMETER_ADDRESS directly — it will skip if the address is empty. Do not inline these helpers in test files. |
| `config.yaml` | OpenMeter config injected into the Docker Compose stack. credits.enabled=false — tests in productcatalog_test.go (TestSettlementMode) depend on this being false. | Adding a new meter slug in a test without adding the corresponding meter definition here causes the sink-worker to silently drop events. credits.enabled=false is a deliberate test contract — do not change without updating tests. |
| `Makefile` | test-local target: tears down, brings up compose stack, waits for sink-worker /healthz on port 30000, then runs go test with TZ=UTC OPENMETER_ADDRESS=http://localhost:38888. | TZ=UTC is essential for deterministic time-window assertions. Port 38888 is the openmeter API; port 30000 is the sink-worker healthz endpoint. |
| `docker-compose.openmeter-local.yaml` | Overrides image: with build: .. so local tests use the locally built binary, not a released image. | docker-compose.openmeter-latest.yaml uses the published GHCR image for CI — do not merge the two approaches. |
| `ledger_backfill_test.go` | Tests the openmeter-jobs ledger backfill-accounts command via docker compose exec and verifies ledger_customer_accounts rows via direct pgxpool queries. | ensureLocalComposeBackfillSupport skips when OPENMETER_ADDRESS is not the localhost:38888 compose stack — this test cannot run against remote envs. |

## Anti-Patterns

- Asserting meter query results synchronously after ingest without assert.EventuallyWithT — data flows async through Kafka and ClickHouse.
- Using hardcoded feature/plan/customer keys shared across test functions — causes 409 or silent state pollution in a shared persistent DB.
- Importing app/common, Wire provider packages, or Ent adapters — e2e tests must only use the HTTP clients and public type constants.
- Adding new meters to tests without a corresponding entry in e2e/config.yaml — the sink-worker silently drops events to unknown meters.
- Using port numbers in the 30000 range for new services without verifying they do not conflict with the root docker-compose.yaml or quickstart (40000 range).

## Decisions

- **Two client layers: generated api.ClientWithResponses for v1 and a hand-rolled v3Client for v3.** — The v3 API was still evolving when these tests were written. A thin hand-rolled wrapper returning (status, *T, *v3Problem) gives tests direct control over HTTP-level assertions without fighting the generated SDK's union-type machinery.
- **credits.enabled=false in e2e/config.yaml.** — E2e tests for settlement-mode validation (TestSettlementMode in productcatalog_test.go) depend on credits being off to assert that credit_only settlement mode is rejected. This is a deliberate, documented contract.
- **Separate docker-compose.openmeter-local.yaml overriding image: with build: for local runs.** — Makes 'make test-local' always test the locally built binary while CI can swap in a published image via docker-compose.openmeter-latest.yaml without changing test code.

## Example: Writing a new v3 resource lifecycle test (create → assert status → use resource ID in subsequent steps)

```
func TestV3MyResource(t *testing.T) {
	c := newV3Client(t) // skips if OPENMETER_ADDRESS not set

	var resourceID string

	t.Run("Should create the resource", func(t *testing.T) {
		body := validMyResourceRequest(uniqueKey("test_key"))
		status, res, problem := c.CreateMyResource(body)
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, res)
		resourceID = res.Id
	})

	t.Run("Should get the resource", func(t *testing.T) {
		require.NotEmpty(t, resourceID)
// ...
```

<!-- archie:ai-end -->
