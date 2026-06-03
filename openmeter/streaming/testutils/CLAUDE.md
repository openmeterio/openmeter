# testutils

<!-- archie:ai-start -->

> In-memory MockStreamingConnector for tests needing a streaming.Connector without a real ClickHouse. Supports AddSimpleEvent/SetSimpleEvents (with WithStoredAt), AddRow for pre-canned exact results, and approximate aggregation mirroring ClickHouse tumble-window semantics.

## Patterns

**AddSimpleEvent for usage-based setup** — Register per-meter events with AddSimpleEvent(meterSlug, value, time, opts...). WithStoredAt(t) sets StoredAt distinct from Time to exercise stored-at cutoff logic. Events auto-sort by Time ASC after each add. (`conn.AddSimpleEvent("api-calls", 10, now, testutils.WithStoredAt(now.Add(-time.Minute)))`)
**AddRow for exact pre-canned QueryMeter results** — Use AddRow(meterSlug, meter.MeterQueryRow{...}) when approximation is insufficient. QueryMeter returns rows whose WindowStart/WindowEnd exactly match params.From/To. AddRow takes precedence over SimpleEvents for that slug. (`conn.AddRow("meter-key", meter.MeterQueryRow{WindowStart: from, WindowEnd: to, Value: 42.0, GroupBy: map[string]*string{}})`)
**Reset() between subtests** — Call conn.Reset() to clear rows and events between cases sharing a connector. AddSimpleEvent appends; rows accumulate unless Reset() is called. (`conn.Reset()`)
**filterStoredAt uses Unix-second precision** — The mock evaluates FilterTimeUnix via storedAt.Unix() seconds, matching ClickHouse DateTime precision. Sub-second StoredAt distinctions are ignored. (`// Use full-second boundaries for StoredAt filters in tests`)
**MeterNotFoundError for unregistered slugs** — If a slug was never registered via AddSimpleEvent/AddRow, QueryMeter returns meter.NewMeterNotFoundError. Register an empty slice via SetSimpleEvents to register without events. (`conn.SetSimpleEvents("my-meter", func(e []SimpleEvent) []SimpleEvent { return e })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `streaming.go` | Whole package: MockStreamingConnector, SimpleEvent, AddSimpleEvent, SetSimpleEvents, AddRow, QueryMeter aggregation, filterStoredAt. | aggregateEvents returns MeterNotFoundError for never-registered slugs — register the slug first. |
| `streaming_test.go` | Tests for approximation accuracy: windowed vs non-windowed, LATEST, StoredAt filtering, empty-window suppression. | Tests import openmeter/testutils (not this package) for GetRFC3339Time — beware import cycles when adding helpers. |

## Anti-Patterns

- Using this mock for tests needing real ClickHouse SQL behaviour (PREWHERE, decimal edge cases) — use CHTestSuite with TEST_CLICKHOUSE_DSN.
- Registering events for a slug after QueryMeter on it within the same test without Reset() — AddSimpleEvent appends.
- Relying on sub-second StoredAt precision — FilterTimeUnix truncates to Unix seconds.
- Calling NewMockStreamingConnector without passing testing.TB — t is required for helper registration.

## Decisions

- **Approximate aggregation rather than full SQL semantics.** — Full SQL emulation is fragile; the approximation suffices for service-layer tests while real ClickHouse via CHTestSuite covers exact SQL behaviour.
- **AddRow takes precedence over SimpleEvents for the same slug.** — Lets tests supply exact results for complex group-by/multi-window scenarios without reimplementing aggregation.

## Example: Test a service querying meter usage with a stored-at cutoff

```
conn := testutils.NewMockStreamingConnector(t)
conn.AddSimpleEvent("api-calls", 5, now, testutils.WithStoredAt(now.Add(-time.Hour)))
conn.AddSimpleEvent("api-calls", 3, now, testutils.WithStoredAt(now.Add(time.Hour)))
rows, err := conn.QueryMeter(ctx, ns, meter, streaming.QueryParams{From: &from, To: &to, FilterStoredAt: &filter.FilterTimeUnix{FilterTime: filter.FilterTime{Lt: &now}}})
// rows[0].Value == 5
```

<!-- archie:ai-end -->
