# testutils

<!-- archie:ai-start -->

> In-memory MockStreamingConnector implementing streaming.Connector for unit/integration tests across billing, credit, entitlement, ledger and subscription packages. Approximates ClickHouse meter aggregation over a slice of SimpleEvent without a real database.

## Patterns

**Two-mode QueryMeter: pre-set rows or event aggregation** — If rows were registered via AddRow for the meter key, QueryMeter returns the exact rows whose WindowStart/End match params.From/To. Otherwise it falls back to aggregateEvents over registered SimpleEvents. Use AddRow for fully-controlled output; use Add/SetSimpleEvents to exercise real window aggregation. (`_, rowOk := m.rows[mm.Key]; if rowOk { /* exact match */ } else { row, err := m.aggregateEvents(mm, params) }`)
**SimpleEvent with explicit StoredAt for late-arriving usage** — SimpleEvent carries Time and StoredAt; AddSimpleEvent defaults StoredAt to Time unless WithStoredAt(t) is passed. aggregateEvents pre-filters by params.FilterStoredAt using filterStoredAt (Unix-second precision, recursive $and/$or) so tests can model events visible only after a stored-at cutoff. (`m.AddSimpleEvent(slug, 1, eventTime, testutils.WithStoredAt(storedAt))`)
**Events kept sorted by Time ASC** — AddSimpleEvent and SetSimpleEvents call sortMeterEvents (slices.SortStableFunc by Time.Compare) after mutation. MeterAggregationLatest relies on this ordering — it takes the last matching event's value. (`slices.SortStableFunc(m.events[meterSlug], func(a, b SimpleEvent) int { return a.Time.Compare(b.Time) })`)
**ClickHouse-faithful windowing semantics** — aggregateEvents truncates from/to to streaming.MinimumWindowSizeDuration (second precision), truncates each event by effectiveWindowSize, treats window end as exclusive, supports SUM (default) and LATEST only, and drops rows with Value == 0 to mimic ClickHouse not emitting empty tumbled windows. (`rows = lo.Filter(rows, func(row meter.MeterQueryRow, _ int) bool { return row.Value != 0 })`)
**Other interface methods are trivial stubs** — CountEvents/ListEvents/ListEventsV2/ListSubjects/ListGroupByValues return empty slices; BatchInsert is a no-op; Create/DeleteNamespace return nil; ValidateJSONPath returns strings.HasPrefix(jsonPath, "$."). Don't rely on these for behavior — only QueryMeter is meaningfully implemented. (`func (m *MockStreamingConnector) ValidateJSONPath(ctx, jsonPath) (bool, error) { return strings.HasPrefix(jsonPath, "$."), nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `streaming.go` | MockStreamingConnector, SimpleEvent, AddSimpleEvent/SetSimpleEvents/AddRow/Reset, WithStoredAt, aggregateEvents, filterStoredAt | aggregateEvents only supports SUM and LATEST — other aggregations silently sum; requires non-nil params.From and params.To; meter-not-found is returned only when no events exist for the key |
| `streaming_test.go` | Table-driven tests proving window boundary, exclusive-end, LATEST ordering, and FilterStoredAt semantics | Tests use testutils.GetRFC3339Time and convert.ToPointer; they document edge cases (end-exclusive, non-aligned from/to, larger-than-period windows) that new aggregation logic must preserve |

## Anti-Patterns

- Expecting full ClickHouse aggregation fidelity (avg/min/max/uniqueCount/groupBy are not implemented — only SUM and LATEST)
- Querying without setting both params.From and params.To (aggregateEvents errors)
- Relying on ListEvents/CountEvents/ListSubjects returning data — they are empty stubs
- Setting StoredAt to model late usage without using WithStoredAt — bare AddSimpleEvent defaults StoredAt to Time
- Mutating m.events directly instead of via Add/SetSimpleEvent, bypassing the Time-ASC sort that LATEST depends on

## Decisions

- **Mock implements aggregation rather than recording calls** — Downstream billing/credit/entitlement lifecycle tests need realistic meter rows over time, including late-arriving usage via StoredAt, not just call assertions
- **Zero-value rows are filtered out of windowed results** — Mirrors ClickHouse not returning tumbled rows for windows with no events, so consumers see the same gaps as production

## Example: Modeling late-arriving usage with an explicit stored-at cutoff

```
streaming := testutils.NewMockStreamingConnector(t)
streaming.AddSimpleEvent("tokens", 5, eventTime, testutils.WithStoredAt(storedAt))
rows, err := streaming.QueryMeter(t.Context(), namespace, meter.Meter{Key: "tokens", Aggregation: meter.MeterAggregationSum}, streaming.QueryParams{
	From: &from,
	To:   &to,
	FilterStoredAt: &filter.FilterTimeUnix{FilterTime: filter.FilterTime{Lte: &cutoff}},
})
```

<!-- archie:ai-end -->
