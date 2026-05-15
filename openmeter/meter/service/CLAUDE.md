# service

<!-- archie:ai-start -->

> Business logic layer for the meter domain. Service delegates reads to the adapter; ManageService adds mutation orchestration including event type validation, cross-domain conflict checks, pre-update hooks, and Watermill event publishing.

## Patterns

**Service embeds adapter for read operations** — Service struct holds *adapter.Adapter and delegates ListMeters/GetMeterByIDOrSlug directly to it — no additional logic for reads. (`type Service struct { adapter *adapter.Adapter }`)
**ManageService embeds Service + adds mutation orchestration** — ManageService embeds meter.Service (the read-only Service), holds *adapter.Adapter separately for mutations, and adds publisher, namespaceManager, eventTypeValidator, preUpdateHooks. (`type ManageService struct { meter.Service; hooks *models.ServiceHookRegistry[meter.Meter]; adapter *adapter.Adapter; publisher eventbus.Publisher; ... }`)
**Publish domain events after each successful mutation** — After successful Create/Update/Delete, ManageService publishes a typed event (NewMeterCreateEvent, NewMeterUpdateEvent, NewMeterDeleteEvent) via eventbus.Publisher. (`meterCreatedEvent := meter.NewMeterCreateEvent(ctx, &createdMeter); s.publisher.Publish(ctx, meterCreatedEvent)`)
**Pre-update hook registry** — RegisterPreUpdateMeterHook appends hooks to hooks registry; UpdateMeter calls s.hooks.PreUpdate before calling adapter.UpdateMeter, enabling cross-domain validation. (`if err := s.hooks.PreUpdate(ctx, &currentMeter); err != nil { return meter.Meter{}, err }`)
**Cross-domain conflict checks before delete** — DeleteMeter checks HasActiveFeatureForMeter and HasEntitlementForMeter via adapter; returns GenericConflictError if active references exist. (`if hasFeatures { return models.NewGenericConflictError(fmt.Errorf("meter has active features and cannot be deleted")) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `manage.go` | ManageService with Create/Update/Delete + hook/event plumbing and all cross-domain validation logic. | CreateMeter calls namespaceManager.CreateNamespace as a side effect (workaround); this namespace creation happens after DB write and publishes an event. Publishing events before adapter mutation succeeds is a violation. |
| `service.go` | Read-only Service struct wrapping adapter; thin delegation layer. | Service is instantiated inside NewManage via New(adapter) — do not instantiate Service separately when ManageService is needed. |

## Anti-Patterns

- Adding DB queries directly to service.go or manage.go — all persistence goes through the adapter.
- Publishing events before the adapter mutation succeeds — events must follow successful writes.
- Skipping the pre-update hooks in UpdateMeter — downstream billing/entitlement hooks rely on them.
- Adding adapter methods directly to Service interface — Service is read-only; ManageService holds direct *adapter.Adapter reference for cross-domain adapter methods not on the interface.

## Decisions

- **Service layer publishes Watermill events after mutations rather than in the adapter.** — Adapter is a pure persistence layer; event publishing is a business concern that belongs above it to keep the adapter testable without an event bus.
- **ManageService holds a direct *adapter.Adapter reference in addition to the embedded meter.Service.** — Cross-domain checks (HasActiveFeatureForMeter, HasEntitlementForMeter) are adapter methods not exposed on the meter.Service interface, requiring direct adapter access.

## Example: Delete a meter with cross-domain conflict guard and post-delete event publish

```
func (s *ManageService) DeleteMeter(ctx context.Context, input meter.DeleteMeterInput) error {
	getMeter, err := s.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{Namespace: input.Namespace, IDOrSlug: input.IDOrSlug})
	if err != nil { return err }
	if getMeter.DeletedAt != nil { return nil } // idempotent
	hasFeatures, err := s.adapter.HasActiveFeatureForMeter(ctx, input.Namespace, getMeter.ID)
	if err != nil { return fmt.Errorf("check features: %w", err) }
	if hasFeatures { return models.NewGenericConflictError(fmt.Errorf("meter has active features")) }
	if err = s.adapter.DeleteMeter(ctx, getMeter); err != nil { return err }
	// re-fetch to get DeletedAt populated
	deletedMeter, err := s.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{Namespace: input.Namespace, IDOrSlug: input.IDOrSlug})
	if err != nil { return err }
	if err := s.publisher.Publish(ctx, meter.NewMeterDeleteEvent(ctx, &deletedMeter)); err != nil {
		return fmt.Errorf("publish delete event: %w", err)
	}
	return nil
// ...
```

<!-- archie:ai-end -->
