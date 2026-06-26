# mockadapter

<!-- archie:ai-start -->

> In-memory test double for meter.Service / meter.ManageService used across many test suites. Stores meters in a slice and can optionally mirror them into Postgres so feature.meter_id FK constraints are satisfied in DB-backed tests.

## Patterns

**Two constructors: read-only and manage** — New([]meter.Meter) returns *adapter (meter.Service); NewManage([]meter.Meter) wraps it in manageAdapter embedding meter.Service to satisfy meter.ManageService. Compile-time asserts exist for both interfaces. (`var _ meter.Service = (*adapter)(nil); var _ meter.ManageService = (*manageAdapter)(nil)`)
**Validate meters on ingress** — New and ReplaceMeters call m.Validate() on every meter and wrap failures in models.NewGenericValidationError. init() (sync.Once) ensures the meters slice is non-nil. (`if err := m.Validate(); err != nil { return nil, models.NewGenericValidationError(...) }`)
**Defensive slice copies** — getMeters() returns slices.Clone(c.meters) and New stores slices.Clone(meters) so callers cannot mutate internal state. ReplaceMeters clones before mutating IDs. (`return slices.Clone(c.meters)`)
**Optional PG sync via SetDBClient** — SetDBClient(*entdb.Client) stores the client and ReplaceMeters upserts meters into db.Meter so features.meter_id FKs resolve. PG sync runs before updating in-memory state to avoid partial-failure inconsistency; reuses existing (namespace,key) rows from shared template DBs. (`if c.dbClient != nil { ... synced[i].ID = existing.ID ... } c.meters = synced`)
**In-memory filter/pagination semantics mirror the real adapter** — ListMeters filters by namespace/IDFilter/Key.In in-memory and reproduces pagination (IsZero -> whole set, else page slice). GetMeterByIDOrSlug matches ID or Key within namespace. (`if params.Key != nil && params.Key.In != nil && !slices.Contains(*params.Key.In, meter.Key) { continue }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | adapter/manageAdapter structs, New/NewManage, init, SetDBClient, TestAdapter alias | TestAdapter = adapter is the exported test handle; SetDBClient nils dbClient back out if initial sync fails |
| `manage.go` | Mock CreateMeter/UpdateMeter/DeleteMeter on manageAdapter | Assigns ulid.Make() IDs; RegisterPreUpdateMeterHook returns NewGenericNotImplementedError; these mutate the slice but do NOT auto-sync to PG (use ReplaceMeters for that) |
| `meter.go` | ListMeters/GetMeterByIDOrSlug/ReplaceMeters/getMeters | ReplaceMeters is the only PG-syncing mutation; pagination math uses PageNumber-1 index and may produce an out-of-range empty page |

## Anti-Patterns

- Using this in production wiring — it is a test-only double (package adapter under mockadapter)
- Mutating the slice returned by getMeters and expecting it to persist (it is a clone)
- Relying on CreateMeter/UpdateMeter in manage.go to satisfy features.meter_id FKs (only ReplaceMeters syncs to PG)
- Updating in-memory meters before PG sync succeeds (breaks the partial-failure guarantee)

## Decisions

- **Optional SetDBClient/ReplaceMeters PG mirroring** — Tests that exercise features/entitlements need real meter rows for FK constraints, but most meter tests want a pure in-memory store; making PG sync opt-in keeps both fast
- **Reuse existing (namespace,key) DB rows when IDs differ** — Shared test template DBs may already contain a meter with the same key; reusing its ID keeps FK references valid instead of failing on a conflict

## Example: Construct a manage-capable mock and optionally back it with Postgres

```
svc, err := mockadapter.NewManage([]meter.Meter{m})
if err != nil { /* ... */ }
// optionally satisfy features.meter_id FKs in DB-backed tests
if err := svc.(interface{ SetDBClient(*entdb.Client) error }).SetDBClient(client); err != nil { /* ... */ }
```

<!-- archie:ai-end -->
