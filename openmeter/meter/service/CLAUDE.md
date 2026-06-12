# service

<!-- archie:ai-start -->

> Business-logic layer for meter management. Service is a thin read wrapper over the Ent adapter; ManageService adds reserved-event-type validation, pre-update hooks, namespace provisioning, feature/entitlement dependency guards, and event publishing.

## Patterns

**Service vs ManageService split** — Service (service.go) embeds nothing and only forwards ListMeters/GetMeterByIDOrSlug to the adapter. ManageService (manage.go) embeds meter.Service (set to New(adapter)) and adds the write methods plus orchestration. Asserts `var _ meter.Service`/`var _ meter.ManageService`. (`func NewManage(...) *ManageService { return &ManageService{ Service: New(adapter), adapter: adapter, ... } }`)
**Reserved event-type validation** — Create/Update/Delete run s.eventTypeValidator (meter.NewEventTypeValidator(reservedEventTypes)) unless input.AllowReservedEventTypes is set, returning models.NewGenericValidationError on failure. (`if !input.AllowReservedEventTypes { if err := s.eventTypeValidator(input.EventType); err != nil { return Meter{}, models.NewGenericValidationError(...) } }`)
**Dependency guards before destructive changes** — DeleteMeter blocks (NewGenericConflictError) if adapter.HasActiveFeatureForMeter or HasEntitlementForMeter is true. UpdateMeter blocks dropping a group-by key still referenced by any feature.MeterGroupByFilters via adapter.ListFeaturesForMeter. (`if hasFeatures { return models.NewGenericConflictError(fmt.Errorf("meter has active features and cannot be deleted")) }`)
**Publish lifecycle events after mutation** — Each successful mutation publishes via s.publisher (eventbus.Publisher): meter.NewMeterCreateEvent / NewMeterUpdateEvent / NewMeterDeleteEvent. Delete re-fetches the (soft-deleted) meter to publish its final state. (`if err := s.publisher.Publish(ctx, meter.NewMeterCreateEvent(ctx, &createdMeter)); err != nil { ... }`)
**Pre-update hooks** — RegisterPreUpdateMeterHook appends to s.preUpdateHooks; UpdateMeter runs all hooks (ctx, input) before calling adapter.UpdateMeter. Annotations default to the current meter's when input.Annotations is nil. (`for _, hook := range s.preUpdateHooks { if err := hook(ctx, input); err != nil { return Meter{}, err } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Read-only Service (New, ListMeters, GetMeterByIDOrSlug) forwarding to adapter | Pure pass-through; do not add write methods or business rules here |
| `manage.go` | ManageService: NewManage constructor, CreateMeter/UpdateMeter/DeleteMeter, hook registration, event publishing, dependency guards | CreateMeter still calls namespaceManager.CreateNamespace (TODO to remove); DeleteMeter is idempotent on already-deleted meters (returns nil); UpdateMeter validates input against currentMeter.ValueProperty |

## Anti-Patterns

- Putting persistence/Ent queries in the service layer instead of the adapter
- Mutating a meter without running reserved-event-type validation (unless AllowReservedEventTypes)
- Deleting a meter or dropping a group-by key without the feature/entitlement guard checks
- Skipping event publication after a successful mutation
- Bypassing pre-update hooks in UpdateMeter

## Decisions

- **ManageService embeds Service (read API) and delegates writes to the adapter** — Keeps reads uniform between Service and ManageService while concentrating orchestration (hooks, events, guards, namespace provisioning) in one place
- **Delete/update guard on dependent features and entitlements** — Meters back features and metered entitlements; allowing deletion or incompatible group-by drops would orphan or break those references, so the service refuses with a conflict error

## Example: Mutation orchestration: validate, persist via adapter, publish event

```
func (s *ManageService) CreateMeter(ctx context.Context, input meter.CreateMeterInput) (meter.Meter, error) {
	if err := input.Validate(); err != nil { return meter.Meter{}, fmt.Errorf("invalid create meter params: %w", err) }
	if !input.AllowReservedEventTypes {
		if err := s.eventTypeValidator(input.EventType); err != nil { return meter.Meter{}, models.NewGenericValidationError(fmt.Errorf("invalid event type: %w", err)) }
	}
	createdMeter, err := s.adapter.CreateMeter(ctx, input)
	if err != nil { return createdMeter, err }
	if err := s.namespaceManager.CreateNamespace(ctx, input.Namespace); err != nil { return createdMeter, fmt.Errorf("failed to create namespace: %w", err) }
	if err := s.publisher.Publish(ctx, meter.NewMeterCreateEvent(ctx, &createdMeter)); err != nil { return createdMeter, fmt.Errorf("failed to publish meter created event: %w", err) }
	return createdMeter, nil
}
```

<!-- archie:ai-end -->
