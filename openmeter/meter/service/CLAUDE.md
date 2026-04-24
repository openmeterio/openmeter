# service

<!-- archie:ai-start -->

> Business logic layer for the meter domain. Service wraps adapter for read-only queries; ManageService adds mutation orchestration including event type validation, reserved event type enforcement, cross-domain conflict checks, pre-update hooks, and Watermill event publishing.

## Patterns

**Service embeds adapter for read operations** — Service struct holds *adapter.Adapter and delegates ListMeters/GetMeterByIDOrSlug directly to it — no additional logic needed for reads. (`type Service struct { adapter *adapter.Adapter }`)
**ManageService embeds Service + adds mutation orchestration** — ManageService struct embeds meter.Service (the read-only Service), holds *adapter.Adapter separately for mutations, and adds publisher, namespaceManager, eventTypeValidator, preUpdateHooks. (`type ManageService struct { meter.Service; adapter *adapter.Adapter; publisher eventbus.Publisher; ... }`)
**Publish domain events after each mutation** — After successful Create/Update/Delete, ManageService publishes a typed event (NewMeterCreateEvent, NewMeterUpdateEvent, NewMeterDeleteEvent) via eventbus.Publisher. (`meterCreatedEvent := meter.NewMeterCreateEvent(ctx, &createdMeter); s.publisher.Publish(ctx, meterCreatedEvent)`)
**Pre-update hook registry** — RegisterPreUpdateMeterHook appends hooks to preUpdateHooks slice; UpdateMeter runs all hooks before calling adapter.UpdateMeter, enabling cross-domain validation (e.g., billing pre-checks). (`for _, hook := range s.preUpdateHooks { if err := hook(ctx, input); err != nil { return meter.Meter{}, err } }`)
**Cross-domain conflict checks before delete** — DeleteMeter checks HasActiveFeatureForMeter and HasEntitlementForMeter via adapter; returns GenericConflictError if any active references exist. (`if hasFeatures { return models.NewGenericConflictError(fmt.Errorf("meter has active features...")) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `manage.go` | ManageService with Create/Update/Delete + hook/event plumbing. Contains all cross-domain validation logic. | CreateMeter calls namespaceManager.CreateNamespace — this is a workaround (see TODO comment) and creates the namespace as a side effect. |
| `service.go` | Read-only Service struct wrapping adapter; thin delegation layer. | Service is instantiated inside NewManage via New(adapter) — do not instantiate separately when ManageService is needed. |

## Anti-Patterns

- Adding DB queries directly to service.go or manage.go — all persistence goes through the adapter.
- Publishing events before the adapter mutation succeeds — events must follow successful writes.
- Skipping the pre-update hooks in UpdateMeter — downstream billing/entitlement hooks rely on them.
- Returning nil error when meter is already soft-deleted in DeleteMeter (idempotent) — current behavior is correct, do not change to error.

## Decisions

- **Service layer publishes Watermill events after mutations rather than in the adapter.** — Adapter is a pure persistence layer; event publishing is a business concern that belongs above it to keep adapter testable without an event bus.
- **ManageService holds a direct *adapter.Adapter reference in addition to the embedded meter.Service.** — Cross-domain checks (HasActiveFeatureForMeter, HasEntitlementForMeter) are adapter methods not exposed on meter.Service interface, requiring direct adapter access.

<!-- archie:ai-end -->
