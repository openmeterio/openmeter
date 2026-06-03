# e2e

<!-- archie:ai-start -->

> End-to-end test suite that runs against a live OpenMeter stack bootstrapped via Docker Compose. Tests call the real HTTP API through the generated Go client (v1) and a hand-rolled v3Client (v3), asserting full-stack behavior including async Kafka->ClickHouse pipelines. No internal domain packages are imported — only HTTP clients and public type definitions.

## Patterns

**OPENMETER_ADDRESS env-gate** — Every test file obtains its client via initClient(t) or newV3Client(t), both of which call t.Skip when OPENMETER_ADDRESS is unset. New tests must never bypass this guard. (`address := os.Getenv("OPENMETER_ADDRESS"); if address == "" { t.Skip("OPENMETER_ADDRESS not set") }`)
**assert.EventuallyWithT for async pipeline assertions** — Any assertion on ClickHouse-derived data (meter queries, entitlement balances) uses assert.EventuallyWithT with a 1-minute timeout and 1-second interval, because Kafka->sink-worker->ClickHouse is async. Never assert synchronously after ingest. (`assert.EventuallyWithT(t, func(t *assert.CollectT) { resp, _ := client.QueryMeterWithResponse(...) }, time.Minute, time.Second)`)
**v3Client (status, body, problem) triple** — v3 API tests use *v3Client from v3helpers_test.go (not the generated SDK); methods return (statusCode, *T, *v3Problem). Assert statusCode first, then the typed body. (`status, plan, problem := c.CreatePlan(body); require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)`)
**Unique keys via timestamp/ULID to avoid shared-DB collisions** — Feature/plan/addon/customer keys carry a timestamp or ULID suffix (uniqueKey() helper). Never use hardcoded keys for entities that cannot be idempotently re-created in the shared persistent DB. (`randKey := fmt.Sprintf("entitlement_uc_test_feature_%d", time.Now().Unix())`)
**shouldRunSlowTests guard for minute-resolution assertions** — Tests that sleep >5s (waiting for a new minute for entitlement reset or credit granularity) must be gated behind shouldRunSlowTests(t), which checks the RUN_SLOW_TESTS env var. (`if !shouldRunSlowTests(t) { t.Skip("Skipping slow test...") }`)
**Dedicated v3 error-path assertion helpers** — Negative-path tests use assertValidationCode (extensions.validationErrors[].code), assertInvalidParameterRule (schema invalid_parameters[].rule), and assertProblemDetail (substring in Detail) from v3helpers_test.go. (`assertValidationCode(t, problem, "plan_phase_duplicated_key")`)
**Meter definitions in config.yaml before test ingest** — Every event type ingested by tests must have a corresponding meter slug in e2e/config.yaml; the sink-worker validates events against known meters and silently drops events for unknown meters. (`meters:
  - slug: plan_meter
    eventType: plan_meter
    aggregation: SUM`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `setup_test.go` | Provides initClient (v1 SDK client) and shouldRunSlowTests — the single source of truth for v1 bootstrapping and env-var guards. | Adding setup logic that skips rather than fails will silently ignore tests in CI when the env var is missing. |
| `v3helpers_test.go` | Defines *v3Client (raw HTTP), v3Problem (RFC 7807 parse), decodeTyped generic, all typed CRUD wrappers, validXxx fixture builders, uniqueKey(), and the assertion helpers. | v3Client.do uses t.Context() with v3RequestTimeout — do not bypass with context.Background(). Use uniqueKey() for all entity keys. |
| `helpers.go` | Shared cross-test helpers: CreateCustomerWithSubject, GetMeterIDBySlug, QueryMeterV3 (raw HTTP v3 meter query). | QueryMeterV3 uses http.DefaultClient and reads OPENMETER_ADDRESS directly; it skips if the address is empty. Do not inline these helpers in test files. |
| `config.yaml` | OpenMeter config injected into the Compose stack: meter definitions, dedupe (redis), credits.enabled=false. | credits.enabled=false is a deliberate test contract (TestSettlementMode); do not change without updating tests. Adding a meter slug in a test without a definition here silently drops events. |
| `Makefile` | test-local: tear down, bring up the compose stack, wait for sink-worker /healthz on port 30000, then run go test with TZ=UTC OPENMETER_ADDRESS=http://localhost:38888. | TZ=UTC is essential for deterministic time-window assertions. Port 38888 is the openmeter API; 30000 is the sink-worker healthz. |
| `docker-compose.openmeter-local.yaml` | Overrides image: with build: .. so local tests run the locally built binary. | docker-compose.openmeter-latest.yaml uses the published GHCR image for CI — do not merge the two approaches. |
| `ledger_backfill_test.go` | Tests the openmeter-jobs ledger backfill-accounts command via docker compose exec and verifies ledger_customer_accounts rows via direct pgxpool queries. | ensureLocalComposeBackfillSupport skips when OPENMETER_ADDRESS is not the localhost:38888 compose stack — this test cannot run against remote environments. |

## Anti-Patterns

- Asserting meter query results synchronously after ingest without assert.EventuallyWithT — data flows async through Kafka and ClickHouse
- Using hardcoded feature/plan/customer keys shared across test functions — causes 409 or silent state pollution in the shared DB
- Importing app/common, Wire provider packages, or Ent adapters — e2e tests must use only HTTP clients and public types
- Adding new meters to tests without a corresponding entry in e2e/config.yaml — the sink-worker silently drops events to unknown meters
- Using ports in the 30000 range for new services without checking they don't conflict with the root docker-compose.yaml or quickstart (40000 range)

## Decisions

- **Two client layers: generated api.ClientWithResponses for v1 and a hand-rolled v3Client for v3.** — The v3 API was still evolving; a thin wrapper returning (status, *T, *v3Problem) gives tests direct HTTP-level control without fighting the generated SDK's union-type machinery.
- **credits.enabled=false in e2e/config.yaml.** — TestSettlementMode depends on credits being off to assert that credit_only settlement mode is rejected — a deliberate, documented contract.
- **Separate docker-compose.openmeter-local.yaml overriding image: with build:.** — Makes 'make test-local' always test the locally built binary while CI swaps in a published image via docker-compose.openmeter-latest.yaml without changing test code.

## Example: A v3 resource lifecycle test (create -> assert status -> reuse ID)

```
func TestV3MyResource(t *testing.T) {
	c := newV3Client(t) // skips if OPENMETER_ADDRESS not set
	var resourceID string
	t.Run("Should create the resource", func(t *testing.T) {
		status, res, problem := c.CreateMyResource(validMyResourceRequest(uniqueKey("test_key")))
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		resourceID = res.Id
	})
}
```

<!-- archie:ai-end -->
