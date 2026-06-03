# quickstart

<!-- archie:ai-start -->

> Self-contained demo environment that boots all OpenMeter services (openmeter, sink-worker, balance-worker, notification-service, billing-worker, openmeter-jobs) via Docker Compose and validates end-to-end ingest->query behavior with quickstart_test.go. Uses port range 40000-49999 to avoid conflicts with the root dev environment and the e2e suite (30000 range).

## Patterns

**OPENMETER_ADDRESS env-gate on tests** — Both initClient and openmeterAddress check OPENMETER_ADDRESS and call t.Skip if unset; tests never run against an embedded server, only a live Compose stack. (`address := os.Getenv("OPENMETER_ADDRESS"); if address == "" { t.Skip("OPENMETER_ADDRESS not set") }`)
**assert.EventuallyWithT for async ingest assertions** — All ingest-then-query assertions use EventuallyWithT with a 30-second timeout and 1-second tick to tolerate Kafka/ClickHouse propagation latency. (`assert.EventuallyWithT(t, func(t *assert.CollectT) { resp := queryMeterV3(...); require.Len(t, resp.Data, 2) }, 30*time.Second, time.Second)`)
**Port range 40000-49999** — All Compose service ports use the 40000 range to avoid collisions with the root dev docker-compose.yaml and the e2e suite (30000 range): openmeter=48888, sink-worker=40000, balance-worker=40001, notification-service=40002, billing-worker=40003. (`ports:
  - "127.0.0.1:48888:8888"  # openmeter API
  - "127.0.0.1:40000:10000" # sink-worker healthz`)
**Separate docker-compose.debug-ports.yaml overlay** — docker-compose.debug-ports.yaml adds host port bindings for Kafka/ClickHouse/Redis/Postgres/Svix only when debugging; the base docker-compose.yaml exposes only application-level ports. make test-local merges both files. (`docker compose -f docker-compose.yaml -f docker-compose.debug-ports.yaml up -d`)
**Unique meter slugs per test run via timestamp suffix** — quickstart_test.go generates v1MeterSlug and v3MeterKey with a UnixNano suffix to avoid collisions on shared persistent DB state across runs. (`suffix := fmt.Sprintf("%d", time.Now().UnixNano()); v1MeterSlug := "quickstart_v1_" + suffix`)
**autoMigrate: migration in config.yaml** — Uses the Atlas SQL migration path (autoMigrate: migration), not the development-only ent schema create, exercising the real production migration pipeline. (`postgres:
  autoMigrate: migration`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `quickstart_test.go` | End-to-end smoke test: creates meters via v1 and v3 APIs, ingests CloudEvents with fixed IDs (00001/00002/00003), and asserts meter query values for both API versions using EventuallyWithT. | Uses context.Background() in helpers — acceptable for an integration test, not application code. Event IDs and expected count values (2 + 1 window split) are intentionally fixed; new tests must use unique suffixes. |
| `docker-compose.yaml` | Full-stack compose: all five OpenMeter binaries plus infra (Kafka, ClickHouse, Redis, Postgres, Svix), referencing docker-compose.base.yaml via extends. | New binaries must be added here with matching command, config volume mount, and depends_on health checks. openmeter-jobs runs 'quickstart cron' (not a long-running server) — do not give it a healthcheck. |
| `config.yaml` | Minimal production-like config: sample meters, Svix configured, portal enabled, autoMigrate: migration. | New required app/config fields must be reflected here or the stack silently fails to start. The Svix apiKey is a test JWT, not sensitive. |
| `Makefile` | test-local: spin up stack (both compose files), health-check sink-worker port 40000, run tests with OPENMETER_ADDRESS=http://localhost:48888, tear down. | The health check hits sink-worker port 40000 (not the openmeter API 48888). TZ is not set here (unlike e2e) — tests use fixed UTC timestamps in event data instead. |
| `README.md` | Operator-facing quickstart walkthrough: launch via docker compose up, ingest CloudEvents via curl, query meters, configure additional meters. | curl examples use port 48888 and event IDs 00001/00002/00003 matching the test; keep them in sync with config.yaml meter slugs and quickstart_test.go. |

## Anti-Patterns

- Using port numbers outside the 40000 range — they conflict with the root docker-compose.yaml or the e2e suite
- Running quickstart_test.go without a live Compose stack — tests must guard with t.Skip on missing OPENMETER_ADDRESS
- Hardcoding event IDs or meter slugs without a unique suffix (the fixed event IDs are intentional only for the canonical test)
- Omitting depends_on health checks when adding a new service to docker-compose.yaml
- Setting autoMigrate to 'ent' instead of 'migration' — this bypasses the Atlas migration pipeline and hides regressions

## Decisions

- **quickstart_test.go uses the generated Go SDK (api/client/go) for v1 but raw HTTP helpers for v3.** — Demonstrates intended SDK usage for v1 while covering the v3 surface, which lacks a generated client helper matching the test's assertion patterns.
- **All five OpenMeter binaries plus openmeter-jobs are declared in docker-compose.yaml.** — Quickstart is the canonical smoke-test for new operators and must demonstrate the full production topology, not a reduced subset.
- **Port range 40000-49999 reserved exclusively for quickstart.** — Avoids conflicts with both the root dev environment and the e2e suite (30000 range), letting all three run simultaneously on one machine.

## Example: Assert meter query value after async ingest using EventuallyWithT

```
assert.EventuallyWithT(t, func(t *assert.CollectT) {
	resp, err := client.QueryMeterWithResponse(context.Background(), v1MeterSlug, &api.QueryMeterParams{
		WindowSize: lo.ToPtr(api.WindowSize(meter.WindowSizeHour)),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode())
	require.Len(t, resp.JSON200.Data, 2)
	assert.Equal(t, float64(2), resp.JSON200.Data[0].Value)
}, 30*time.Second, time.Second)
```

<!-- archie:ai-end -->
