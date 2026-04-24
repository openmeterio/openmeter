# quickstart

<!-- archie:ai-start -->

> Self-contained demo environment that boots all OpenMeter services via Docker Compose and validates end-to-end ingest→query behavior with quickstart_test.go. It is the canonical smoke-test for new operators and uses port range 40000–49999 to avoid conflicts with the main dev environment.

## Patterns

**OPENMETER_ADDRESS env-gate on tests** — Every test helper calls t.Skip if OPENMETER_ADDRESS is unset; tests are never run against an embedded server, only against a live Docker Compose stack. (`func initClient(t *testing.T) { if address == "" { t.Skip("OPENMETER_ADDRESS not set") } }`)
**require.EventuallyWithT for async ingest assertions** — All ingest-then-query assertions use assert.EventuallyWithT with a 30-second timeout and 1-second tick to tolerate Kafka/ClickHouse propagation latency. (`assert.EventuallyWithT(t, func(t *assert.CollectT) { resp := queryMeter...; require.Len(t, resp.JSON200.Data, 2) }, 30*time.Second, time.Second)`)
**Port range 40000–49999** — All Docker Compose service ports in this folder use the 40000 range to avoid collisions with the root dev docker-compose.yaml which binds standard ports. (`openmeter: 48888:8888, sink-worker: 40000:10000, kafka debug: 49092:29092`)
**Separate debug-ports overlay** — docker-compose.debug-ports.yaml adds host port bindings for Kafka/ClickHouse/Redis/Postgres/Svix only when debugging; base docker-compose.yaml exposes only application-level ports. (`make test-local merges both files: docker compose -f docker-compose.yaml -f docker-compose.debug-ports.yaml`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `quickstart/quickstart_test.go` | End-to-end smoke test: creates meters, ingests CloudEvents, asserts meter query values for both v1 and v3 APIs | Uses context.Background() in helper functions (acceptable here as integration test — not application code); uses api/client/go and api/v3 generated clients directly |
| `quickstart/docker-compose.yaml` | Full-stack compose file referencing docker-compose.base.yaml services; declares all five OpenMeter binaries as services | New binaries added to the main stack must also be added here with matching command and config volume mount |
| `quickstart/config.yaml` | Minimal production-like config for the quickstart stack; uses autoMigrate: migration (not ent) | If new required config fields are added to app/config, this file must be updated or the quickstart stack will fail to start |
| `quickstart/Makefile` | Defines test-local target: spin up stack, health-check sink-worker, run tests, tear down | Health check hits sink-worker port 40000 (not the openmeter API port 48888) |

## Anti-Patterns

- Using port numbers outside the 40000 range — they conflict with the root docker-compose.yaml dev environment
- Running quickstart_test.go without a live Docker Compose stack — tests must guard with t.Skip on missing OPENMETER_ADDRESS
- Hardcoding event IDs or meter slugs without a unique suffix — parallel test runs collide on shared meter state
- Omitting depends_on health checks when adding a new service to docker-compose.yaml — services start before dependencies are ready

## Decisions

- **quickstart_test.go uses the generated Go SDK clients (api/client/go, api/v3) rather than raw HTTP for v1 but raw HTTP for v3** — Demonstrates the intended SDK usage to new users while covering the v3 API surface that lacks a fully generated client helper at the time of writing
- **autoMigrate set to 'migration' (not 'ent') in config.yaml** — Production-like behavior: exercises the Atlas-generated SQL migration path rather than the development-only ent.Schema.Create upsertion

## Example: Assert meter query value after async ingest using EventuallyWithT

```
assert.EventuallyWithT(t, func(t *assert.CollectT) {
	windowSize := meter.WindowSizeHour
	resp, err := client.QueryMeterWithResponse(context.Background(), v1MeterSlug, &api.QueryMeterParams{
		WindowSize: lo.ToPtr(api.WindowSize(windowSize)),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode())
	require.Len(t, resp.JSON200.Data, 2)
	assert.Equal(t, float64(2), resp.JSON200.Data[0].Value)
}, 30*time.Second, time.Second)
```

<!-- archie:ai-end -->
