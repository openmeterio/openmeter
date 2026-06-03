# meter

<!-- archie:ai-start -->

> Meter domain: defines usage aggregation rules (event type, COUNT/SUM/MAX/UNIQUE_COUNT, group-by JSON paths), manages meter lifecycle with soft-delete, validates events against meters at ingest via ParseEvent, and publishes Watermill events after mutations. Service is read-only; ManageService embeds it and adds mutation, pre-update hooks, and event publishing consumed by billing/entitlement.

## Patterns

**Service/ManageService split** — Read-only callers depend on meter.Service; mutating callers depend on meter.ManageService (embeds Service + CreateMeter/UpdateMeter/DeleteMeter/RegisterHooks). Wire injects ManageService where mutation is needed. (`type ManageService interface { Service; CreateMeter(...); UpdateMeter(...); DeleteMeter(...) }`)
**Validate() on every input type** — CreateMeterInput/UpdateMeterInput/DeleteMeterInput/GetMeterInput/ListMetersParams each have explicit Validate() called before any adapter/service logic. (`return models.NewNillableGenericValidationError(errors.Join(errs...))`)
**ValueProperty/GroupBy invariants enforced on both model and input** — COUNT forbids ValueProperty; other aggregations require it with a '$' prefix; GroupBy values start with '$' and keys match groupByKeyRegExp. Checked in both Meter.Validate() and CreateMeterInput.Validate() via shared helpers. (`if aggregation == MeterAggregationCount && valueProperty != nil { return errors.New(...) }`)
**Watermill event publishing after mutations** — The service layer (not the adapter) publishes MeterCreate/Update/DeleteEvent after a successful adapter write, using the metadata.EventType triple and ComposeResourcePath. (`ev := meter.NewMeterCreateEvent(ctx, &created); s.publisher.Publish(ctx, ev)`)
**Soft-delete via DeletedAt** — Meters are never hard-deleted; ListMeters excludes soft-deleted unless IncludeDeleted=true; DeleteMeter sets DeletedAt and is idempotent. (`IncludeDeleted bool // in ListMetersParams`)
**ParseEvent for ingest validation** — meter.ParseEvent(meter, data) validates and extracts value/group-by from CloudEvent JSON; ErrInvalidEvent=data error, ErrInvalidMeter=config error. COUNT meters get Value=1.0; UNIQUE_COUNT get ValueString. (`parsedEvent, err := meter.ParseEvent(m, rawData) // *ParsedEvent{Value, ValueString, GroupBy}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service/ManageService interfaces plus all input types with Validate() — the meter API surface. | WithoutNamespace in ListMetersParams is the only safe cross-namespace listing flag — never pass empty namespace without it. |
| `meter.go` | Meter type, MeterAggregation/WindowSize enums, aggregation/group-by validation helpers, MeterQueryRow. | validateMeterAggregation/validateMeterGroupBy are shared by Meter.Validate() and CreateMeterInput.Validate() — keep in sync. |
| `parse.go` | ParseEvent/ParseEventString extracting value/group-by at ingest. | Missing GROUP BY fields return empty string (not error) intentionally — do not change. |
| `event.go` | MeterCreate/Update/DeleteEvent implementing EventName()/EventMetadata() for Watermill via ComposeResourcePath. | MeterDeleteEvent.Validate() requires non-nil DeletedAt — set by the adapter before constructing the event. |
| `errors.go` | MeterNotFoundError wrapping GenericNotFoundError + IsMeterNotFoundError helper. | Use IsMeterNotFoundError for type-checked matching rather than errors.As on the inner error. |
| `adapter/adapter.go` | Ent adapter enforcing soft-delete, namespace isolation, and TransactingRepo wrapping. | All queries need DeletedAt IS NULL unless IncludeDeleted; never call a.db outside TransactingRepo. |
| `mockadapter/adapter.go` | In-memory test mock; optional PG sync via SetDBClient for FK constraints. | Test-only; SetDBClient must precede seeding to sync; RegisterHooks is a noop in the mock. |

## Anti-Patterns

- Passing empty Namespace to ListMeters without WithoutNamespace=true.
- Adding DB queries directly in service/manage.go — persistence goes through the adapter.
- Publishing events before the adapter mutation succeeds.
- Skipping validateJSONPaths (streaming.Connector) before CreateMeter/UpdateMeter — invalid paths fail silently until query time.
- Hard-deleting meter rows instead of setting DeletedAt — breaks FK integrity and idempotent delete.

## Decisions

- **The service layer publishes Watermill events after adapter mutations, not the adapter.** — Keeps the adapter purely persistence; event publishing is a cross-cutting orchestration concern.
- **ManageService holds a direct *adapter.Adapter alongside the embedded Service.** — Lets ManageService call write/soft-delete helpers not on the read-only Service interface without bloating it.
- **mockadapter PG sync is opt-in via SetDBClient rather than always requiring a DB.** — Most unit tests need only in-memory resolution; tests with feature meter_id FKs opt in.

## Example: Publishing a domain event after a meter mutation

```
// openmeter/meter/service/manage.go
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
