# clickhouse

<!-- archie:ai-start -->

> Implements streaming.Connector against a single shared ClickHouse events table (namespace+type+subject+time ORDER BY key). Every query type — event listing, meter aggregation, subject/group-by enumeration, batch insert — is a distinct query-struct with a toSQL()/toCountRowSQL() method and is exercised by a matching _test.go that asserts exact SQL strings.

## Patterns

**Query-struct pattern** — Each query is a plain Go struct (e.g. queryMeter, queryEventsTable, listSubjectsQuery) with a toSQL() method returning (string, []interface{}). The Connector delegates to the struct; business logic lives in the struct, not in the Connector methods. (`query := queryMeter{Database: c.config.Database, EventsTableName: c.config.EventsTableName, ...}; sql, args, err := query.toSQL()`)
**sqlbuilder.ClickHouse for all SQL construction** — All SQL is built via github.com/huandu/go-sqlbuilder with the ClickHouse flavour. Never use raw fmt.Sprintf for WHERE clauses or value interpolation — use builder.Var() / builder.Equal() / builder.Where() to keep parameterisation safe. (`sb := sqlbuilder.ClickHouse.NewSelectBuilder(); sb.Where(sb.Equal("namespace", d.Namespace))`)
**columnFactory for table-qualified column references** — Use columnFactory(eventsTableName) to get a func(string) string that emits table-qualified column names (e.g. om_events.namespace). Required whenever the query could join or use CTEs that introduce ambiguous column names. (`getColumn := columnFactory(d.EventsTableName); query.Where(query.Equal(getColumn("namespace"), d.Namespace))`)
**toCountRowSQL companion for progress tracking** — Every query struct that may be tracked via progressmanager must also implement toCountRowSQL() which counts only by ORDER-BY columns (namespace, type, subject, time) for a cheap estimate. The Connector checks params.ClientID != nil before calling withProgressContext. (`if params.ClientID != nil { countSQL, countArgs := query.toCountRowSQL(); ctx, err = c.withProgressContext(ctx, namespace, *params.ClientID, countSQL, countArgs) }`)
**ClickHouse error-code detection for not-found mapping** — Map ClickHouse code 60 (table/view not found) to models.NewNamespaceNotFoundError or meter.NewMeterNotFoundError by string-matching err.Error() for "code: 60". Never let the raw ClickHouse error escape to callers. (`if strings.Contains(err.Error(), "code: 60") { return nil, models.NewNamespaceNotFoundError(namespace) }`)
**selectCustomerIdColumn + customersWhere for customer filtering** — Customer filtering requires two coordinated calls: selectCustomerIdColumn builds a WITH map CTE and adds customer_id to SELECT; customersWhere adds the IN (subjects) WHERE clause. Both accept the eventsTableName for table-qualified references. (`query = selectCustomerIdColumn(d.EventsTableName, *d.Customers, query); query = customersWhere(d.EventsTableName, *d.Customers, query)`)
**PREWHERE optimisation for group-by filter pushdown** — When EnablePrewhere is true and FilterGroupBy is non-empty, the query is split: ordered-column filters move to PREWHERE and data-column JSON_VALUE filters stay in WHERE. This is done by capturing the SQL before data-where clauses and string-replacing WHERE with PREWHERE. (`if d.EnablePrewhere { sqlBeforeApplyingDataWheres, _ = query.Build() } // ... after adding data wheres: sql = prewhere-patched version`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `connector.go` | Public surface: implements streaming.Connector, owns Config validation, table creation on startup, and orchestrates all query structs. Method bodies are thin delegators. | Config.SkipCreateTables must be set to true in unit tests that use mock ClickHouse; otherwise New() calls createTable on a mock that has no Exec setup. |
| `meter_query.go` | Most complex query struct. Handles all aggregation types (sum/avg/min/max/count/uniqExact/argMax), window-size tumble functions, group-by JSON extraction, stored-at filter, and PREWHERE split. Also contains NullDecimal scanner and scanRows. | Month window size requires explicit toDateTime(..., tz) cast to avoid Date-vs-DateTime scan mismatch. The from() method merges query.From with Meter.EventFrom, always taking the later of the two. |
| `event_query.go` | Handles v1 event listing (queryEventsTable), count (queryCountEvents), and batch insert (InsertEventsQuery). Table DDL (createEventsTable) lives here too. | InsertEventsQuery.ToSQL is exported (capital T) unlike all other query structs which are unexported. The column order in INSERT must match the table definition exactly. |
| `event_query_v2.go` | Cursor-paginated v2 event listing using store_row_id as tie-breaker. Cursor conditions differ between ASC and DESC sort orders. | customersWhere in v2 passes the full tableName (database.table) while v1 passes only eventsTableName — keep consistent with the helper's expectation. |
| `queryhelper.go` | Shared helpers: selectCustomerIdColumn (WITH map CTE), customersWhere, subjectWhere, columnFactory. | GetUsageAttribution().GetValues() may return customer key + subject keys; both are mapped in the WITH clause so the customer_id lookup works for either identifier. |
| `suite_test.go` | Provides CHTestSuite: skips integration tests unless TEST_CLICKHOUSE_DSN is set, creates a unique temp database per test, and drops it on success only. | Use t.Context() not context.Background() in tests; CreateTempDatabase cleanup intentionally skips DROP on test failure to aid debugging. |
| `mock.go` | MockClickHouse and MockRows for unit tests that don't need a real ClickHouse connection. Used by connector_test.go tests. | MockClickHouse.Query must be set up before calls to queryMeter/queryEventsTable methods since the Connector calls Query synchronously. |

## Anti-Patterns

- Adding raw fmt.Sprintf string interpolation for query values — always use sqlbuilder.Var() or builder parameters to prevent injection and keep test SQL assertions stable.
- Calling c.config.ClickHouse.Query/Exec directly in a new Connector method without going through a query-struct toSQL() — loses the testable SQL-assertion pattern.
- Returning raw ClickHouse errors to callers — map code 60 to domain errors (NamespaceNotFoundError, MeterNotFoundError) before returning.
- Adding window-size logic to scanRows — scanRows only scans; toSQL handles all window column selection.
- Skipping toCountRowSQL on a new query struct that supports progress tracking via ClientID — withProgressContext must be called before executing the main query.

## Decisions

- **Single shared events table across all namespaces with namespace as the leading ORDER BY column** — Avoids per-namespace table creation/deletion overhead; ClickHouse partition pruning by toYYYYMM(time) and the (namespace, type, subject, toStartOfHour(time)) sort key give adequate read performance for typical multi-tenant query patterns.
- **Query structs with toSQL() instead of inline SQL in Connector methods** — Enables unit tests to assert exact SQL output without a real ClickHouse connection, and keeps each query's logic isolated and independently testable.
- **store_row_id (ULID) as cursor tie-breaker in v2 pagination** — ClickHouse DateTime precision is seconds; multiple events in the same second would be non-deterministically ordered without a unique tie-breaker. ULIDs are time-ordered and unique, so store_row_id > cursor.ID correctly pages through same-second events.

## Example: Add a new meter aggregation query that must be unit-testable

```
// In meter_query.go, add to the toSQL() switch on d.Meter.Aggregation:
case meterpkg.MeterAggregationNewType:
    selectColumns = append(selectColumns, fmt.Sprintf("newFunc(JSON_VALUE(%s, '%s')) AS value", getColumn("data"), escapeJSONPathLiteral(*d.Meter.ValueProperty)))

// In meter_query_test.go, add a table-driven test case:
{
    name: "Aggregate with new type",
    query: queryMeter{
        Database: "openmeter", EventsTableName: "om_events", Namespace: "ns",
        Meter: meter.Meter{Key: "m", EventType: "e", Aggregation: meter.MeterAggregationNewType, ValueProperty: lo.ToPtr("$.v")},
    },
    wantSQL:  "SELECT ... newFunc(JSON_VALUE(om_events.data, '$.v')) AS value ...",
    wantArgs: []interface{}{"ns", "e"},
}
```

<!-- archie:ai-end -->
