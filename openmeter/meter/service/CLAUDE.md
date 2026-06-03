# service

<!-- archie:ai-start -->

> Business logic layer for the meter domain. Service delegates reads to the adapter; ManageService adds mutation orchestration — event-type validation, cross-domain conflict checks, pre-update hooks, and Watermill event publishing.

## Patterns

**Service embeds adapter for reads** — Service holds *adapter.Adapter and delegates ListMeters/GetMeterByIDOrSlug directly — no extra logic for reads. (`type Service struct { adapter *adapter.Adapter }`)
**ManageService embeds Service + adds mutation orchestration** — ManageService embeds meter.Service, holds *adapter.Adapter separately for mutations, plus publisher, namespaceManager, eventTypeValidator, hooks. (`type ManageService struct { meter.Service; hooks *models.ServiceHookRegistry[meter.Meter]; adapter *adapter.Adapter; publisher eventbus.Publisher }`)
**Publish domain events after each successful mutation** — After successful Create/Update/Delete, publish a typed event (NewMeterCreateEvent/Update/Delete) via eventbus.Publisher — never before the write. (`s.publisher.Publish(ctx, meter.NewMeterCreateEvent(ctx, &createdMeter))`)
**Pre-update hook registry** — RegisterHooks appends to the hooks registry; UpdateMeter calls s.hooks.PreUpdate before adapter.UpdateMeter for cross-domain validation. (`if err := s.hooks.PreUpdate(ctx, &currentMeter); err != nil { return meter.Meter{}, err }`)
**Cross-domain conflict checks before delete** — DeleteMeter checks HasActiveFeatureForMeter and HasEntitlementForMeter via the adapter; returns GenericConflictError if active references exist. (`if hasFeatures { return models.NewGenericConflictError(fmt.Errorf("meter has active features and cannot be deleted")) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `manage.go` | ManageService Create/Update/Delete plus hook/event plumbing and all cross-domain validation. | CreateMeter calls namespaceManager.CreateNamespace as a side effect (TODO workaround) after the DB write; publishing events before the adapter mutation succeeds is a violation. |
| `service.go` | Read-only Service wrapping the adapter; thin delegation. | Service is created inside NewManage via New(adapter) — do not instantiate Service separately when ManageService is needed. |

## Anti-Patterns

- Adding DB queries directly to service.go or manage.go — persistence goes through the adapter
- Publishing events before the adapter mutation succeeds
- Skipping pre-update hooks in UpdateMeter — downstream billing/entitlement hooks rely on them
- Adding adapter-only methods to the read-only Service interface

## Decisions

- **Service layer publishes Watermill events after mutations rather than the adapter** — Adapter is pure persistence; event publishing is a business concern that keeps the adapter testable without an event bus.
- **ManageService holds a direct *adapter.Adapter in addition to the embedded meter.Service** — Cross-domain checks (HasActiveFeatureForMeter, HasEntitlementForMeter) are adapter methods not on the meter.Service interface.

## Example: Delete a meter with conflict guard and post-delete event

```
func (s *ManageService) DeleteMeter(ctx context.Context, input meter.DeleteMeterInput) error {
	getMeter, err := s.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{Namespace: input.Namespace, IDOrSlug: input.IDOrSlug})
	if err != nil { return err }
	if getMeter.DeletedAt != nil { return nil }
	hasFeatures, err := s.adapter.HasActiveFeatureForMeter(ctx, input.Namespace, getMeter.ID)
	if err != nil { return fmt.Errorf("check features: %w", err) }
	if hasFeatures { return models.NewGenericConflictError(fmt.Errorf("meter has active features")) }
	if err = s.adapter.DeleteMeter(ctx, getMeter); err != nil { return err }
	deletedMeter, err := s.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{Namespace: input.Namespace, IDOrSlug: input.IDOrSlug})
	if err != nil { return err }
	return s.publisher.Publish(ctx, meter.NewMeterDeleteEvent(ctx, &deletedMeter))
}
```

<!-- archie:ai-end -->
