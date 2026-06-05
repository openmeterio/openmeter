# clickhouse

<!-- archie:ai-start -->

> Concrete ClickHouse implementation of streaming.Connector — reads/aggregates metered usage events out of a single MergeTree events table. Each public method validates params, delegates to a query-builder struct, then scans rows; the connector holds no per-request state, only Config.

## Patterns

**Per-operation query-builder struct with toSQL()** — Each read/write operation has a dedicated struct (queryMeter, queryEventsTable, queryEventsTableV2, queryCountEvents, listSubjectsQuery, InsertEventsQuery, createEventsTable) carrying Database/EventsTableName/Namespace plus params, and a toSQL() / ToSQL() (and often toCountRowSQL()) method returning (sql, args). Connector methods build this struct, call toSQL, and run c.config.ClickHouse.Query/Exec. (`query := queryMeter{Database: c.config.Database, EventsTableName: c.config.EventsTableName, Namespace: namespace, Meter: meter, ...}; sql, args, err := query.toSQL()`)
**go-sqlbuilder.ClickHouse builders, never string concat** — All SQL is built via sqlbuilder.ClickHouse.NewSelectBuilder / NewInsertBuilder / NewCreateTableBuilder / NewCTEBuilder. Args are bound positionally through query.Build(); never interpolate user values into the SQL string except via sqlbuilder.Escape (see selectCustomerIdColumn). (`query := sqlbuilder.ClickHouse.NewSelectBuilder(); query.Where(query.Equal("namespace", d.Namespace))`)
**Validate then map known errors to typed domain errors** — Public methods first reject empty namespace / call params.Validate() wrapping in models.NewGenericValidationError, then translate ClickHouse error codes: 'code: 60' -> models.NewNamespaceNotFoundError or meterpkg.NewMeterNotFoundError; 'code: 36' (ValidateJSONPath bad args) -> (false, nil). Re-check the typed error before wrapping with fmt.Errorf. (`if strings.Contains(err.Error(), "code: 60") { return nil, meterpkg.NewMeterNotFoundError(query.Meter.Key) }`)
**Optional query progress tracking via ClientID** — When params.ClientID != nil, build a toCountRowSQL() count query and call c.withProgressContext(ctx, namespace, clientID, countSQL, countArgs) to attach a progressmanager-tracking context. Progress failures are logged via c.config.Logger but never returned. (`ctx, err = c.withProgressContext(ctx, namespace, *params.ClientID, countSQL, countArgs); if err != nil { c.config.Logger.Error("failed track progress", ...) }`)
**columnFactory aliasing for join/filter columns** — Shared where/select helpers (subjectWhere, customersWhere, selectCustomerIdColumn) use columnFactory(eventsTableName) to qualify column names as alias.column. Customer attribution is resolved with a ClickHouse map(...) CTE keyed on subject via customer.GetUsageAttribution(). (`getColumn := columnFactory(eventsTableName); subjectColumn := getColumn("subject")`)
**NullDecimal scan wrapper for decimal aggregates** — meter_query.go defines NullDecimal wrapping decimal.NullDecimal with a Scan that also accepts a raw decimal.Decimal; scanRows uses it so SUM/MIN/MAX results that ClickHouse may return as decimal-or-null are handled uniformly. (`nullDecimal := dest[2].(*NullDecimal); nullDecimal.Valid = true; nullDecimal.Decimal = decimal.NewFromFloat(value)`)
**Config-validated constructor that DDLs the events table** — New(ctx, Config) runs Config.Validate() (Logger, ClickHouse, Database, EventsTableName, ProgressManager all required) then calls createTable unless SkipCreateTables. The events table schema (createEventsTable.toSQL) is MergeTree, PARTITION BY toYYYYMM(time), ORDER BY (namespace, type, subject, toStartOfHour(time)). (`if !config.SkipCreateTables { if err := connector.createTable(ctx); err != nil { return nil, fmt.Errorf("create tables: %w", err) } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `connector.go` | Connector struct + all streaming.Connector methods (QueryMeter, ListEvents/V2, CountEvents, ListSubjects, ListGroupByValues, BatchInsert, ValidateJSONPath) and namespace.Handler no-ops | CreateNamespace/DeleteNamespace are intentional no-ops (events table is shared across namespaces); QueryMeter sorts GroupBy for deterministic SQL and rewrites WindowStart/End to the period when WindowSize is nil |
| `meter_query.go` | queryMeter struct, NullDecimal, from() event-window resolution, toSQL/toCountRowSQL/scanRows | from() merges params.From with Meter.EventFrom (takes the later); EnableDecimalPrecision / EnablePrewhere flags change generated SQL |
| `event_query.go` | createEventsTable DDL, queryEventsTable, queryCountEvents, InsertEventsQuery, getTableName | events table ORDER BY/PARTITION drive query performance; toCountRowSQL only filters on ordered columns; InsertEventsQuery column order must match BatchInsert |
| `event_query_v2.go` | queryEventsTableV2 builder for the advanced ListEventsV2 filtering path | v2 carries full params struct rather than discrete fields; keep filter handling aligned with pkg/filter |
| `queryhelper.go` | subjectWhere, customersWhere, selectCustomerIdColumn (subject_to_customer_id map CTE), columnFactory | customer subject values come from customer.GetUsageAttribution(); escape literals with sqlbuilder.Escape; empty customer/subject sets must not emit a WHERE that excludes all rows |
| `subject_query.go / group_by_values_query.go` | listSubjectsQuery and group-by value enumeration builders | these map missing-meter errors via meterpkg.IsMeterNotFoundError |
| `connector_test.go` | GetMockConnector + MockClickHouse/MockRows unit tests with SkipCreateTables | mock-based tests assert exact bound args and code:60 -> MeterNotFoundError translation |
| `connector_query_test.go / suite_test.go` | CHTestSuite integration tests against a real ClickHouse instance | tests skip when ClickHouse is unavailable (s.T().Skipped()); cover every MeterAggregation x EnableDecimalPrecision combination |

## Anti-Patterns

- Building SQL with fmt.Sprintf/string concatenation of user input instead of sqlbuilder builders + bound args (only sqlbuilder.Escape'd literals are acceptable)
- Returning raw ClickHouse errors without translating code:60 to NamespaceNotFound/MeterNotFound — callers branch on those typed errors
- Adding a new read method without a matching query-builder struct + toSQL(), or skipping params.Validate()
- Letting progress-tracking (withProgressContext) failures abort the query instead of logging and continuing
- Implementing CreateNamespace/DeleteNamespace to actually create/drop tables — the events table is shared across namespaces

## Decisions

- **Single shared events MergeTree table partitioned by month and ordered (namespace, type, subject, toStartOfHour(time))** — Lowest-cardinality always-filtered columns left-most so ClickHouse prunes partitions and rows; namespaces share the table so create/delete-namespace are no-ops
- **Query progress is opt-in via ClientID and best-effort** — Progress estimation uses a separate count query and must never fail the actual data query
- **NullDecimal wraps decimal.NullDecimal with a permissive Scan** — ClickHouse may return aggregate values as decimal or null depending on EnableDecimalPrecision; a single scan target keeps scanRows uniform

## Example: Adding a validated read method backed by a query-builder struct

```
func (c *Connector) QueryMeter(ctx context.Context, namespace string, meter meterpkg.Meter, params streaming.QueryParams) ([]meterpkg.MeterQueryRow, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("validate params: %w", err)
	}
	query := queryMeter{Database: c.config.Database, EventsTableName: c.config.EventsTableName, Namespace: namespace, Meter: meter /* ... */}
	sql, args, err := query.toSQL()
	if err != nil {
		return nil, fmt.Errorf("build sql query: %w", err)
	}
	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		if strings.Contains(err.Error(), "code: 60") {
// ...
```

<!-- archie:ai-end -->
