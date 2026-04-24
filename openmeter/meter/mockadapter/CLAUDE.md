# mockadapter

<!-- archie:ai-start -->

> In-memory mock implementation of meter.Service and meter.ManageService for use in tests. Optionally syncs meters to PostgreSQL via SetDBClient to satisfy FK constraints on features.meter_id.

## Patterns

**Two-level struct: adapter (Service) + manageAdapter (ManageService)** — adapter implements meter.Service with in-memory slice; manageAdapter embeds adapter and adds Create/Update/Delete/RegisterPreUpdateMeterHook, satisfying meter.ManageService. (`type manageAdapter struct { adapter *adapter; meter.Service }`)
**NewManage factory for ManageService** — Tests needing full mutation access call NewManage(meters); tests needing read-only call New(meters). Both validate input meters via m.Validate(). (`func NewManage(meters []meter.Meter) (meter.ManageService, error)`)
**Optional PG sync via SetDBClient** — Call adapter.SetDBClient(*entdb.Client) after New to make CreateMeter/ReplaceMeters also persist rows to PostgreSQL for FK integrity. PG sync happens before in-memory state updates. (`adapter.SetDBClient(client) // enables FK-safe test scenarios`)
**ReplaceMeters for bulk test seeding** — ReplaceMeters atomically replaces all in-memory meters and syncs to PG if SetDBClient was called. Reuses existing PG rows by (namespace, key) to handle shared template DBs. (`adapter.ReplaceMeters(ctx, meters)`)
**Zero-page pagination returns full dataset** — ListMeters returns all matching items when params.Page.IsZero(), skipping offset math — mirrors real pagination behavior for unpaginated callers. (`if params.Page.IsZero() { return pagination.Result[meter.Meter]{Items: meters, TotalCount: len(meters)}, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Core in-memory adapter struct, New/NewManage constructors, SetDBClient, TestAdapter type alias. | TestAdapter type alias exposes adapter internals to tests; only use for direct field inspection, not for bypassing the service interface. |
| `manage.go` | In-memory Create/Update/Delete implementations. RegisterPreUpdateMeterHook returns NotImplemented. | UpdateMeter matches by ID or Key in the in-memory slice — both must match the namespace to avoid cross-namespace mutations. |
| `meter.go` | ListMeters (in-memory filter + pagination) and GetMeterByIDOrSlug + ReplaceMeters (PG sync logic). | PG sync in ReplaceMeters reuses existing (namespace, key) rows; if tests share a template DB, IDs may be remapped silently. |

## Anti-Patterns

- Using mockadapter in production wiring — it is test-only.
- Calling adapter.meters directly from tests instead of going through the Service interface.
- Calling SetDBClient after meters are already seeded without calling ReplaceMeters — existing in-memory meters won't be synced to PG.
- Expecting RegisterPreUpdateMeterHook to work — it returns NotImplemented in the mock.

## Decisions

- **Optional PG sync via SetDBClient rather than always requiring a DB.** — Pure in-memory tests need no DB; only tests that also create features (FK on features.meter_id) need PG sync.

<!-- archie:ai-end -->
