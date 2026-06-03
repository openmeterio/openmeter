# mockadapter

<!-- archie:ai-start -->

> In-memory mock of meter.Service and meter.ManageService for tests. Optionally syncs meters to PostgreSQL via SetDBClient to satisfy FK constraints on features.meter_id.

## Patterns

**Two-level struct: adapter + manageAdapter** — adapter implements meter.Service with an in-memory slice; manageAdapter embeds it and adds Create/Update/Delete/RegisterHooks, satisfying meter.ManageService. (`type manageAdapter struct { adapter *adapter; meter.Service }`)
**New vs NewManage factory selection** — Read-only tests call New(meters); mutation tests call NewManage(meters). Both validate input meters. (`func NewManage(meters []meter.Meter) (meter.ManageService, error)`)
**Optional PG sync via SetDBClient** — Call adapter.SetDBClient(*entdb.Client) after New to make ReplaceMeters/CreateMeter persist rows to PG; PG sync happens before in-memory state updates. (`adapter.SetDBClient(client)`)
**ReplaceMeters for bulk test seeding** — ReplaceMeters validates, syncs to PG if SetDBClient was set (reusing existing (namespace, key) rows), then updates in-memory state only after PG success. (`adapter.ReplaceMeters(ctx, meters)`)
**Zero-page pagination returns full dataset** — ListMeters returns all matching items when params.Page.IsZero(), skipping offset math. (`if params.Page.IsZero() { return pagination.Result[meter.Meter]{Items: meters, TotalCount: len(meters)}, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | In-memory adapter struct, New/NewManage constructors, SetDBClient, TestAdapter type alias. | TestAdapter exposes adapter internals — use only for field inspection, not to bypass the Service interface. |
| `manage.go` | In-memory Create/Update/Delete; RegisterHooks is a noop. | UpdateMeter/DeleteMeter match by ID or Key in the slice — both must match the namespace to avoid cross-namespace mutations. |
| `meter.go` | ListMeters (in-memory filter + pagination), GetMeterByIDOrSlug, ReplaceMeters (PG sync logic). | PG sync in ReplaceMeters reuses existing (namespace, key) rows; with shared template DBs, IDs may be silently remapped. |

## Anti-Patterns

- Using mockadapter in production wiring — it is test-only
- Calling adapter.meters directly from tests instead of the Service interface
- Calling SetDBClient after meters are seeded without ReplaceMeters — existing in-memory meters won't sync to PG
- Expecting RegisterHooks/RegisterPreUpdateMeterHook to fire — it is a noop in the mock
- Using context.Background() in tests when t.Context() is available

## Decisions

- **Optional PG sync via SetDBClient rather than always requiring a DB** — Pure in-memory tests need no DB; only tests that also create features (FK on features.meter_id) need PG sync.

<!-- archie:ai-end -->
