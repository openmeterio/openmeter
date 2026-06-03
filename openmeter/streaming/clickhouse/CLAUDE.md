# clickhouse

<!-- archie:ai-start -->

> Concrete streaming.Connector implementation against ClickHouse's single shared events table (ORDER BY namespace, type, subject, toStartOfHour(time)). Every query type is a distinct query-struct with a toSQL() method, keeping business logic out of the Connector and enabling SQL-assertion unit tests without a live ClickHouse.

## Patterns

**Query-struct with toSQL()** — Each query type (queryMeter, queryEventsTable, InsertEventsQuery, listSubjectsQuery) is a plain struct with a toSQL()/ToSQL() returning (string, []interface{}). Connector methods are thin delegators that build the struct and call toSQL(); all logic lives in the struct. (`query := queryMeter{Database: c.config.Database, EventsTableName: c.config.EventsTableName, Namespace: namespace, Meter: meter}; sql, args, err := query.toSQL()`)
**sqlbuilder.ClickHouse for all SQL** — Build SQL via github.com/huandu/go-sqlbuilder ClickHouse flavour. Never fmt.Sprintf WHERE values — use builder.Var()/Equal()/Where() to keep parameterisation safe and SQL assertions stable. (`sb := sqlbuilder.ClickHouse.NewSelectBuilder(); sb.Where(sb.Equal("namespace", d.Namespace))`)
**columnFactory for table-qualified columns** — Use columnFactory(eventsTableName) to emit table-qualified column names (om_events.namespace) whenever CTEs or joins could introduce ambiguity. (`getColumn := columnFactory(d.EventsTableName); query.Where(query.Equal(getColumn("namespace"), d.Namespace))`)
**toCountRowSQL companion for progress tracking** — Query structs that may be tracked via progressmanager also implement toCountRowSQL() counting only by ORDER-BY columns for a cheap estimate. Connector checks params.ClientID != nil before calling withProgressContext. (`if params.ClientID != nil { countSQL, countArgs := query.toCountRowSQL(); ctx, err = c.withProgressContext(ctx, namespace, *params.ClientID, countSQL, countArgs) }`)
**ClickHouse code 60 mapped to domain errors** — Map ClickHouse error string 'code: 60' (table/view not found) to models.NewNamespaceNotFoundError or meterpkg.NewMeterNotFoundError. Never let raw ClickHouse errors escape. (`if strings.Contains(err.Error(), "code: 60") { return nil, models.NewNamespaceNotFoundError(namespace) }`)
**Coordinated customer filtering** — Customer filtering needs two calls: selectCustomerIdColumn builds a WITH map CTE and adds customer_id to SELECT; customersWhere adds the IN (subjects) WHERE clause. Both take eventsTableName. (`query = selectCustomerIdColumn(d.EventsTableName, *d.Customers, query); query = customersWhere(d.EventsTableName, *d.Customers, query)`)
**SkipCreateTables=true in unit tests** — New() calls createTable on startup; mock connections have no DDL Exec setup. Set Config.SkipCreateTables=true when constructing against a mock. (`config := Config{..., SkipCreateTables: true}; connector, err := New(ctx, config)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `connector.go` | Public surface implementing streaming.Connector; owns Config.Validate(), createTable on startup, orchestrates all query structs. Method bodies are thin delegators. | Config.SkipCreateTables must be true in unit tests or New() calls createTable on a mock with no Exec setup. |
| `meter_query.go` | Most complex query struct: all aggregations (sum/avg/min/max/count/uniqExact/argMax), window tumble functions, group-by JSON extraction, stored-at filter, PREWHERE split, NullDecimal scanner, scanRows. | Month window size needs explicit toDateTime(..., tz) cast to avoid Date-vs-DateTime scan mismatch. from() merges query.From with Meter.EventFrom taking the later. |
| `event_query.go` | v1 event listing (queryEventsTable), count (queryCountEvents), batch insert (InsertEventsQuery), and table DDL (createEventsTable). | InsertEventsQuery.ToSQL is exported (capital T) unlike other structs. INSERT column order must match the table definition exactly. |
| `event_query_v2.go` | Cursor-paginated v2 listing using store_row_id (ULID) as tie-breaker; cursor conditions differ between ASC and DESC. | customersWhere in v2 passes full tableName (database.table) while v1 passes only eventsTableName — keep consistent with the helper expectation. |
| `queryhelper.go` | Shared helpers: selectCustomerIdColumn (WITH map CTE), customersWhere, subjectWhere, columnFactory. | GetUsageAttribution().GetValues() may return customer key + subject keys; both are mapped in the WITH clause. |
| `suite_test.go` | CHTestSuite: skips integration tests unless TEST_CLICKHOUSE_DSN is set, creates a unique temp DB per test, drops it on success only. | Use t.Context() not context.Background(); cleanup intentionally skips DROP on failure to aid debugging. |
| `mock.go` | MockClickHouse and MockRows for unit tests without a real connection. | MockClickHouse.Query must be set up before query methods are called — the Connector calls Query synchronously. |

## Anti-Patterns

- Using fmt.Sprintf string interpolation for query values instead of sqlbuilder.Var()/builder parameters.
- Calling c.config.ClickHouse.Query/Exec directly in a new method without a query-struct toSQL().
- Returning raw ClickHouse errors instead of mapping code 60 to NamespaceNotFoundError / MeterNotFoundError.
- Adding window-size logic to scanRows — scanRows only scans; toSQL handles window column selection.
- Skipping toCountRowSQL on a new progress-trackable query struct — withProgressContext must run before the main query.

## Decisions

- **Single shared events table across namespaces with namespace as leading ORDER BY column.** — Avoids per-namespace table churn; partition pruning by toYYYYMM(time) plus the sort key gives adequate multi-tenant read performance.
- **Query structs with toSQL() instead of inline SQL.** — Enables unit tests to assert exact SQL without a live ClickHouse and keeps each query independently testable.
- **store_row_id (ULID) as v2 cursor tie-breaker.** — ClickHouse DateTime is second-precision; same-second events would order non-deterministically without a unique time-ordered tie-breaker.

## Example: Add a new meter aggregation query that must be unit-testable

```
// In meter_query.go toSQL() switch on d.Meter.Aggregation:
case meterpkg.MeterAggregationNewType:
    selectColumns = append(selectColumns, fmt.Sprintf("newFunc(JSON_VALUE(%s, '%s')) AS value", getColumn("data"), escapeJSONPathLiteral(*d.Meter.ValueProperty)))
// Then add a table-driven case in meter_query_test.go asserting wantSQL/wantArgs.
```

<!-- archie:ai-end -->
