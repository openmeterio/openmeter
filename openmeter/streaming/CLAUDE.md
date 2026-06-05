# streaming

<!-- archie:ai-start -->

> Defines the streaming.Connector interface (embedding namespace.Handler) for reading/aggregating metered usage out of ClickHouse, plus the shared param structs (QueryParams, ListEventsParams/V2, ListSubjectsParams, ListGroupByValuesParams) and the RawEvent/CustomerUsageAttribution value types. Concrete impls live in clickhouse/, retry/, and testutils/.

## Patterns

**Self-validating param structs** — Every params type has a Validate() that collects into var errs []error and returns models.NewNillableGenericValidationError(errors.Join(errs...)) (or errors.Join for ListEvents*). Callers/impls must Validate before querying. (`return models.NewNillableGenericValidationError(errors.Join(errs...))`)
**RawEvent dual-tagged for ClickHouse + JSON** — RawEvent fields carry both `ch:"..."` (ClickHouse column) and `json:"..."` tags; Namespace is `json:"-"` and stored_at/ingested_at use omitempty,omitzero. (`Time time.Time `ch:"time" json:"time"``)
**Usage attribution by key/subject, never by ID** — CustomerUsageAttribution attributes usage via Key + SubjectKeys (GetValues); ID exists only to map subjects to customers and is excluded from attribution values. (`NewCustomerUsageAttribution(id, key, subjectKeys)`)
**Group-by/filter ambiguity guards** — QueryParams.Validate rejects multiple subject/customer filters unless the matching group-by is present, and requires a customer filter when grouping by customer_id. (`len(p.FilterCustomer) > 1 && !slices.Contains(p.GroupBy, "customer_id")`)
**EventSortField allowlist with zero-value default** — EventSortField.Validate treats "" as valid (resolved to time at query time) and rejects anything outside Values(); used by ListEventsV2Params.SortBy. (`EventSortFieldTime / EventSortFieldIngestedAt / EventSortFieldStoredAt`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `connector.go` | Declares the Connector interface (CountEvents/ListEvents/ListEventsV2/ListSubjects/ListGroupByValues/QueryMeter/BatchInsert/ValidateJSONPath), RawEvent, and the list-subjects/group-by param validators. | Connector embeds namespace.Handler — impls must implement CreateNamespace/DeleteNamespace too; ListGroupByValues caps the window at 30 days and rejects From older than 90 days. |
| `query_params.go` | QueryMeter input (QueryParams) with the filter/group-by ambiguity validation rules. | Adding a filter field without adding its ambiguity guard can produce ambiguous aggregation results. |
| `eventparams.go` | ListEventsParams / ListEventsV2Params plus EventSortField; V2 uses pkg/filter.FilterString/FilterTime and a pagination.Cursor. | V2 filters validate via ValidateWithComplexity(1); ListEvents Validate returns plain errors.Join (not the generic-validation wrapper). |
| `usageattribution.go` | Customer interface + CustomerUsageAttribution (Validate/GetValues/Equal). | Validate requires an ID and at least one of Key or non-empty SubjectKeys; Equal compares SubjectKeys order-sensitively. |
| `defaults.go` | MinimumWindowSize / MinimumWindowSizeDuration constants (1 second). | These bound the smallest representable aggregation window. |

## Anti-Patterns

- Adding a Connector method or params field without a Validate() rule on its param struct.
- Attributing usage to a customer by ID instead of Key/SubjectKeys.
- Dropping the `ch` or `json` tag on a RawEvent field, or giving Namespace a JSON tag (it is `json:"-"`).
- Allowing multiple subject/customer filters without the corresponding group-by, or customer_id group-by without a customer filter.
- Implementing CreateNamespace/DeleteNamespace to create/drop tables — the events table is shared across namespaces (see clickhouse/).

## Decisions

- **Connector embeds namespace.Handler.** — Usage storage participates in namespace provisioning lifecycle even though the ClickHouse events table is shared.
- **Usage attribution carries ID separately from attribution values.** — ID is needed to map subjects to customers but usage is attributed only by key/subject keys.

## Example: The usage read/aggregation contract over ClickHouse

```
type Connector interface {
	namespace.Handler

	CountEvents(ctx context.Context, namespace string, params CountEventsParams) ([]CountEventRow, error)
	ListEvents(ctx context.Context, namespace string, params ListEventsParams) ([]RawEvent, error)
	ListEventsV2(ctx context.Context, params ListEventsV2Params) ([]RawEvent, error)
	ListSubjects(ctx context.Context, params ListSubjectsParams) ([]string, error)
	ListGroupByValues(ctx context.Context, params ListGroupByValuesParams) ([]string, error)
	QueryMeter(ctx context.Context, namespace string, meter meter.Meter, params QueryParams) ([]meter.MeterQueryRow, error)
	BatchInsert(ctx context.Context, events []RawEvent) error
	ValidateJSONPath(ctx context.Context, jsonPath string) (bool, error)
}
```

<!-- archie:ai-end -->
