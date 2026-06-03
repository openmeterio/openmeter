# streaming

<!-- archie:ai-start -->

> Defines the streaming.Connector interface (ClickHouse-backed usage querying and event ingestion), all query-parameter types with Validate(), and the Customer/CustomerUsageAttribution attribution model. Concrete implementation lives in clickhouse/, a transient-retry decorator in retry/, and an in-memory test double in testutils/.

## Patterns

**Connector embeds namespace.Handler** — streaming.Connector embeds namespace.Handler so every implementation participates in namespace lifecycle (CreateNamespace/DeleteNamespace) alongside query/ingest methods. (`type Connector interface { namespace.Handler; QueryMeter(...); BatchInsert(...) }`)
**Validate() on every Params type** — QueryParams, ListEventsParams, ListEventsV2Params, ListSubjectsParams, ListGroupByValuesParams each implement Validate(); callers validate before calling the connector. (`if err := params.Validate(); err != nil { return nil, err }`)
**Customer interface + CustomerUsageAttribution value type** — Any type implementing GetUsageAttribution() can be a QueryParams.FilterCustomer; CustomerUsageAttribution is the concrete value with Validate()/GetValues()/Equal(). (`type Customer interface { GetUsageAttribution() CustomerUsageAttribution }`)
**FilterStoredAt uses FilterTimeUnix (second precision)** — stored_at predicates use *filter.FilterTimeUnix, not *filter.FilterTime, because ClickHouse DateTime columns are second-precision. (`FilterStoredAt: &filter.FilterTimeUnix{Gte: lo.ToPtr(unix)}`)
**Query-struct with toSQL() (in clickhouse/)** — Every query type in the clickhouse/ sub-package is a struct with a toSQL() method built via sqlbuilder; no raw SQL in Connector method bodies. (`q := meterQuery{namespace: ns, meter: m, params: p}; sql, args := q.toSQL()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/streaming/connector.go` | Connector interface plus CountEventsParams, RawEvent, ListSubjectsParams, ListGroupByValuesParams with inline Validate(). | Adding a method here without a matching implementation in clickhouse/, retry/, and testutils/. |
| `openmeter/streaming/query_params.go` | QueryParams with customer_id group-by constraints (requires FilterCustomer when groupBy=customer_id; max 1 FilterCustomer otherwise). | New group-by dimensions added without updating the Validate() constraint. |
| `openmeter/streaming/eventparams.go` | ListEventsParams (From required) and cursor-based ListEventsV2Params; EventSortField. | Conflating the two listing APIs — v1 uses From, v2 uses Cursor. |
| `openmeter/streaming/usageattribution.go` | Customer interface and CustomerUsageAttribution value type; GetValues() feeds ClickHouse IN clauses. | CustomerUsageAttribution.Validate() requires at least Key or one SubjectKey. |

## Anti-Patterns

- Putting raw SQL or fmt.Sprintf interpolation in Connector methods or clickhouse/ — use the query-struct toSQL() with sqlbuilder.Var()
- Wrapping BatchInsert with retry in retry/ — duplicate inserts break idempotency
- Adding application errors (MeterNotFoundError) to the retry predicate — only transient infra errors retry
- Using filter.FilterTime for stored_at predicates instead of filter.FilterTimeUnix
- Returning raw ClickHouse errors — map code 60 to NamespaceNotFoundError/MeterNotFoundError

## Decisions

- **Single shared events table across namespaces with namespace as leading ORDER BY column** — Simplifies schema management; the leading namespace sort key gives efficient per-namespace partition pruning without per-namespace DDL.
- **Query-struct toSQL() pattern instead of inline SQL** — Each query type becomes independently testable via SQL-string assertions and avoids inline interpolation bugs.
- **store_row_id (ULID) as the v2 cursor tie-breaker** — Timestamps are not unique at millisecond granularity; a sortable ULID provides a monotonic secondary key without a new column.

<!-- archie:ai-end -->
