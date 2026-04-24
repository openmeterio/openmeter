# meter

<!-- archie:ai-start -->

> Meter domain: defines usage aggregation rules (event type, aggregation function, group-by JSON paths), manages meter lifecycle with soft-delete semantics, validates events against meters at ingest time, and publishes Watermill events after mutations. ManageService extends Service with mutation and pre-update hooks consumed by billing and entitlement.

## Patterns

**Service/ManageService interface split** — Callers that only need reads depend on meter.Service; callers needing mutations depend on meter.ManageService (which embeds Service). Wire providers inject ManageService where mutation is needed. (`type ManageService interface { Service; CreateMeter(...); UpdateMeter(...); DeleteMeter(...); RegisterPreUpdateMeterHook(...) }`)
**Validate() on every input type** — All input types (CreateMeterInput, UpdateMeterInput, DeleteMeterInput, GetMeterInput, ListMetersParams) have explicit Validate() methods called before any adapter or service logic runs. (`func (i CreateMeterInput) Validate() error { ... return models.NewNillableGenericValidationError(errors.Join(errs...)) }`)
**models.GenericError wrapping for domain errors** — MeterNotFoundError wraps models.NewGenericNotFoundError and implements models.GenericError. Use IsMeterNotFoundError(err) for type-checked error inspection. (`return &MeterNotFoundError{err: models.NewGenericNotFoundError(fmt.Errorf("meter not found: %s", id))}`)
**Watermill event publishing after mutations** — Service layer publishes domain events (MeterCreateEvent, MeterUpdateEvent, MeterDeleteEvent) after successful adapter writes using metadata.EventType triple and ComposeResourcePath. (`NewMeterCreateEvent(ctx, &meter) // implements EventName()+EventMetadata() for Watermill routing`)
**ValueProperty rules enforced on model and input** — COUNT aggregation forbids ValueProperty; all others require it with a '$' prefix. GroupBy values must start with '$'; keys must match groupByKeyRegExp. These invariants are checked in both Meter.Validate() and CreateMeterInput.Validate(). (`if aggregation == MeterAggregationCount && valueProperty != nil { return errors.New(...) }`)
**Soft-delete via DeletedAt** — Meters are never hard-deleted. ListMeters excludes soft-deleted meters unless IncludeDeleted=true. DeleteMeter sets DeletedAt; idempotent on already-deleted meters. (`IncludeDeleted bool // in ListMetersParams`)
**ParseEvent for ingest validation** — meter.ParseEvent(meter, data) validates and extracts value and group-by fields from CloudEvent JSON. Returns ErrInvalidEvent for bad data, ErrInvalidMeter for configuration errors. COUNT meters get Value=1.0; UNIQUE_COUNT meters get ValueString. (`parsedEvent, err := meter.ParseEvent(m, rawData) // returns *ParsedEvent{Value, ValueString, GroupBy}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/meter/service.go` | Service and ManageService interface definitions plus all input types with Validate(). Single source of truth for the meter API surface. | WithoutNamespace flag in ListMetersParams is the only safe way to list meters across namespaces — never pass empty namespace without this flag. |
| `openmeter/meter/meter.go` | Meter domain type, MeterAggregation/WindowSize enums, aggregation/group-by validation helpers, and MeterQueryRow result type. | validateMeterAggregation and validateMeterGroupBy are shared between Meter.Validate() and CreateMeterInput.Validate() — keep them in sync. |
| `openmeter/meter/parse.go` | ParseEvent and ParseEventString: validate and extract value/group-by from CloudEvent JSON at ingest time. ErrInvalidEvent = data error, ErrInvalidMeter = config error. | GROUP BY missing fields return empty string (not error) — this is intentional and must not be changed. |
| `openmeter/meter/event.go` | Domain event structs (MeterCreateEvent, MeterUpdateEvent, MeterDeleteEvent) implementing EventName()/EventMetadata() for Watermill. Each uses metadata.ComposeResourcePath for source/subject. | MeterDeleteEvent.Validate() requires DeletedAt to be non-nil — always set by the adapter before constructing the event. |
| `openmeter/meter/errors.go` | MeterNotFoundError wrapping models.GenericNotFoundError and IsMeterNotFoundError helper. | Use IsMeterNotFoundError for type-checked matching rather than errors.As directly on the inner error. |
| `openmeter/meter/adapter/adapter.go` | Ent/PostgreSQL adapter; enforces soft-delete, namespace isolation, and TransactingRepo wrapping. | All queries must include DeletedAt IS NULL predicate unless IncludeDeleted is true. |
| `openmeter/meter/mockadapter/adapter.go` | In-memory mock for tests; optionally syncs to PG via SetDBClient for FK constraint satisfaction. | Test-only — never wire mockadapter into production. SetDBClient must be called before meters are seeded to sync to PG. |

## Anti-Patterns

- Passing empty Namespace to ListMeters without WithoutNamespace=true — silently returns no results or errors.
- Using float64 for MeterQueryRow.Value instead of the typed field — NaN/Inf are explicitly rejected in ParseEvent.
- Adding DB queries directly in service/manage.go — all persistence must go through the adapter interface.
- Skipping validateJSONPaths (streaming.Connector) before CreateMeter/UpdateMeter — invalid paths fail silently until query time.
- Hard-deleting meter rows instead of setting DeletedAt — breaks FK integrity and idempotent delete semantics.

## Decisions

- **Service layer publishes Watermill events after adapter mutations rather than in the adapter** — Keeps adapter purely concerned with persistence; event publishing is a cross-cutting side-effect that belongs at the orchestration layer.
- **ManageService holds a direct *adapter.Adapter reference alongside embedded Service** — Allows ManageService to call adapter methods not exposed on the Service interface (e.g. soft-delete helpers) without bloating Service with write operations.
- **mockadapter optional PG sync via SetDBClient rather than always requiring DB** — Most unit tests need only in-memory meter resolution; tests that create features with meter FK constraints opt-in by calling SetDBClient.

## Example: Publishing a domain event after a meter mutation

```
// In openmeter/meter/service/manage.go
func (s *ManageService) CreateMeter(ctx context.Context, input meter.CreateMeterInput) (meter.Meter, error) {
    if err := input.Validate(); err != nil {
        return meter.Meter{}, err
    }
    created, err := s.adapter.CreateMeter(ctx, input)
    if err != nil {
        return meter.Meter{}, err
    }
    ev := meter.NewMeterCreateEvent(ctx, &created)
    if err := s.publisher.Publish(ctx, ev); err != nil {
        s.logger.Error("failed to publish meter create event", "error", err)
    }
    return created, nil
}
```

<!-- archie:ai-end -->
