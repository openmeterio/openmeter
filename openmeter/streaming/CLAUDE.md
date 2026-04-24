# streaming

<!-- archie:ai-start -->

> Defines the streaming.Connector interface (ClickHouse-backed usage query and event ingestion) and all query parameter types, plus the Customer/CustomerUsageAttribution attribution model. Concrete implementations live in clickhouse/, a retry decorator in retry/, and an in-memory test double in testutils/.

## Patterns

**Connector extends namespace.Handler** — streaming.Connector embeds namespace.Handler so every ClickHouse implementation participates in namespace lifecycle (CreateNamespace/DeleteNamespace). (`type Connector interface { namespace.Handler; CountEvents(...); QueryMeter(...); BatchInsert(...) }`)
**Validate() on every param type** — All Params structs (QueryParams, ListEventsParams, ListEventsV2Params, ListSubjectsParams, ListGroupByValuesParams) implement Validate() returning models.NewNillableGenericValidationError — callers invoke Validate() before passing params to the connector. (`if err := params.Validate(); err != nil { return nil, err }`)
**Customer interface + CustomerUsageAttribution value type** — Customer is an interface requiring GetUsageAttribution() — any type that carries subject/key attribution can be passed to QueryParams.FilterCustomer. CustomerUsageAttribution is the concrete value type with Validate(), GetValues(), and Equal(). (`type Customer interface { GetUsageAttribution() CustomerUsageAttribution }`)
**FilterStoredAt uses FilterTimeUnix (Unix-second precision)** — QueryParams.FilterStoredAt is *filter.FilterTimeUnix, not *filter.FilterTime, because ClickHouse DateTime columns have second precision — always use FilterTimeUnix for stored_at predicates. (`FilterStoredAt: &filter.FilterTimeUnix{Gte: lo.ToPtr(unix)}`)
**Query-struct with toSQL() in clickhouse/ sub-package** — Each query type in clickhouse/ is a struct with a toSQL() method using sqlbuilder; never put raw SQL in Connector method bodies — every new query type must follow this pattern. (`q := meterQuery{namespace: ns, meter: m, params: p}; sql, args := q.toSQL()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/streaming/connector.go` | Primary interface definition plus CountEventsParams, RawEvent, ListSubjectsParams, ListGroupByValuesParams with inline Validate(). | Adding new methods here without a matching implementation in clickhouse/, retry/, and testutils/. |
| `openmeter/streaming/query_params.go` | QueryParams struct with customer_id group-by constraints (requires FilterCustomer when groupBy=customer_id; max 1 FilterCustomer without customer_id groupBy). | Adding new group-by dimensions without updating the corresponding Validate() constraint. |
| `openmeter/streaming/eventparams.go` | ListEventsParams, ListEventsV2Params (cursor-based v2), EventSortField with Validate(). | ListEventsParams uses From (required, not pointer) while ListEventsV2Params uses Cursor — do not conflate the two listing APIs. |
| `openmeter/streaming/usageattribution.go` | Customer interface and CustomerUsageAttribution value type; GetValues() returns subject key + key values used for ClickHouse IN clauses. | CustomerUsageAttribution.Validate() requires at least Key or one SubjectKey — creating one without either will fail at the connector layer. |
| `openmeter/streaming/clickhouse/connector.go` | Concrete ClickHouse implementation; wires the query structs to the ClickHouse driver. | Direct ClickHouse errors must be mapped to domain errors (code 60 → NamespaceNotFoundError/MeterNotFoundError) before returning. |
| `openmeter/streaming/testutils/streaming.go` | MockStreamingConnector for unit tests; AddSimpleEvent/SetSimpleEvents/AddRow/Reset. | MockStreamingConnector truncates StoredAt to Unix seconds — do not rely on sub-second precision in stored-at filter tests. |

## Anti-Patterns

- Adding raw fmt.Sprintf string interpolation in clickhouse/ query methods — use sqlbuilder.Var() or builder parameters to prevent injection
- Wrapping BatchInsert with retry in the retry/ decorator — duplicate event inserts on retry break idempotency
- Adding application-level errors (MeterNotFoundError) to the retry predicate — only transient infrastructure errors should be retried
- Using filter.FilterTime instead of filter.FilterTimeUnix for stored_at predicates — ClickHouse DateTime has second precision
- Returning raw ClickHouse errors to callers — map code 60 to domain error types before returning

## Decisions

- **Single shared events table across all namespaces with namespace as leading ORDER BY column** — Simplifies ClickHouse schema management; namespace as the leading sort key provides efficient per-namespace partition pruning without per-namespace table DDL.
- **Query-struct pattern with toSQL() instead of inline SQL in Connector methods** — Makes each query type independently testable via SQL string assertions in _test.go; prevents inline string interpolation bugs.
- **store_row_id (ULID) as cursor tie-breaker in v2 pagination** — Timestamps alone are not unique at millisecond granularity; ULID provides a monotonic, sortable secondary key without adding a separate auto-increment column.

## Example: Building and validating QueryParams with customer filter

```
import (
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

params := streaming.QueryParams{
	From:           &from,
	To:             &to,
	FilterCustomer: []streaming.Customer{myCustomer},
	GroupBy:        []string{"customer_id"},
}
if err := params.Validate(); err != nil {
	return nil, err
}
rows, err := connector.QueryMeter(ctx, namespace, meter, params)

```

<!-- archie:ai-end -->
