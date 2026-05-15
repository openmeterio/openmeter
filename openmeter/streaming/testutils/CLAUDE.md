# testutils

<!-- archie:ai-start -->

> In-memory MockStreamingConnector for tests that need a streaming.Connector without a real ClickHouse. Supports AddSimpleEvent/SetSimpleEvents with optional WithStoredAt, AddRow for pre-canned exact results, and approximate aggregation that mirrors ClickHouse tumble-window semantics.

## Patterns

**AddSimpleEvent for usage-based test setup** — Register per-meter events with AddSimpleEvent(meterSlug, value, time, opts...). Use WithStoredAt(t) to set a StoredAt different from the event Time, exercising stored-at cutoff logic in billing finalisation. Events are auto-sorted by Time ASC after each add. (`conn.AddSimpleEvent("api-calls", 10, now, testutils.WithStoredAt(now.Add(-time.Minute)))`)
**AddRow for exact pre-canned QueryMeter results** — When the approximate aggregation is insufficient, use AddRow(meterSlug, meter.MeterQueryRow{...}) to register exact results. QueryMeter returns rows whose WindowStart/WindowEnd match params.From/To exactly. AddRow takes precedence over SimpleEvents for that slug. (`conn.AddRow("meter-key", meter.MeterQueryRow{WindowStart: from, WindowEnd: to, Value: 42.0, GroupBy: map[string]*string{}})`)
**Reset() between subtests** — Call conn.Reset() to clear all rows and events between test cases that share a connector instance. AddSimpleEvent appends; rows accumulate across calls unless Reset() is called. (`conn.Reset()`)
**filterStoredAt uses Unix-second precision** — The mock's filterStoredAt evaluates FilterTimeUnix using Unix() seconds, matching ClickHouse's DateTime column precision. Tests relying on sub-second StoredAt distinctions will not work correctly. (`// Use full-second boundaries for StoredAt filters in tests; sub-second differences are ignored`)
**aggregateEvents returns MeterNotFoundError for unregistered slugs** — If a meterSlug has never been registered via AddSimpleEvent or AddRow, QueryMeter returns meter.NewMeterNotFoundError. Always call AddSimpleEvent at least once, or use SetSimpleEvents(slug, func(e []SimpleEvent) []SimpleEvent { return e }) to register an empty slice. (`conn.SetSimpleEvents("my-meter", func(e []SimpleEvent) []SimpleEvent { return e }) // register without events`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `streaming.go` | Entire package implementation: MockStreamingConnector, SimpleEvent, AddSimpleEvent, SetSimpleEvents, AddRow, QueryMeter aggregation logic, filterStoredAt helper. | aggregateEvents returns MeterNotFoundError if the meterSlug has never been registered — always register the meter slug before calling QueryMeter. |
| `streaming_test.go` | Comprehensive test cases for the mock's approximation accuracy: windowed vs non-windowed, LATEST aggregation, StoredAt filtering, empty-window row suppression. | Tests import openmeter/testutils (not this package) for GetRFC3339Time helpers — be careful about import cycles when adding new test helpers. |

## Anti-Patterns

- Using MockStreamingConnector for tests that need real ClickHouse SQL behaviour (PREWHERE, decimal precision edge cases) — use ConnectorTestSuite with TEST_CLICKHOUSE_DSN instead.
- Registering events for a meter slug after calling QueryMeter on that slug within the same test without Reset() — AddSimpleEvent appends; rows accumulate across calls.
- Relying on sub-second StoredAt precision in FilterTimeUnix predicates — the mock truncates to Unix seconds matching ClickHouse DateTime behaviour.
- Using MockStreamingConnector without passing a testing.TB to NewMockStreamingConnector — always pass t for proper test helper registration.

## Decisions

- **Approximate aggregation rather than full SQL semantics** — A full SQL emulation would be complex and fragile. The approximation is accurate enough for service-layer tests (billing charges, entitlement balance) while keeping the mock simple. Tests that need exact SQL behaviour use real ClickHouse via CHTestSuite.
- **AddRow takes precedence over SimpleEvents for the same slug** — Allows tests to provide exact pre-canned results for complex scenarios (group-by, multi-window) without implementing full aggregation logic in the mock.

## Example: Test a service that queries meter usage with a stored-at cutoff

```
conn := testutils.NewMockStreamingConnector(t)
conn.AddSimpleEvent("api-calls", 5, now, testutils.WithStoredAt(now.Add(-time.Hour)))
conn.AddSimpleEvent("api-calls", 3, now, testutils.WithStoredAt(now.Add(time.Hour)))
// Only the first event should be visible before cutoff:
rows, err := conn.QueryMeter(ctx, ns, meter, streaming.QueryParams{
    From: &from, To: &to,
    FilterStoredAt: &filter.FilterTimeUnix{FilterTime: filter.FilterTime{Lt: &now}},
})
// rows[0].Value == 5
```

<!-- archie:ai-end -->
