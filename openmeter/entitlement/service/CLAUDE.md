# service

<!-- archie:ai-start -->

> The entitlement.Service facade: orchestrates CRUD, scheduling/superseding, access resolution and value computation across the metered/static/boolean sub-type connectors, the entitlement repo, customer/feature services and the event bus.

## Patterns

**Service built from ServiceConfig** — NewEntitlementService(ServiceConfig) wires the three sub-type connectors, EntitlementRepo, FeatureConnector, CustomerService, MeterService, Publisher and *lockr.Locker into the private service struct. (`func NewEntitlementService(config ServiceConfig) entitlement.Service`)
**Mutations run inside transaction.Run** — Create/Override/Schedule/Supersede/Delete wrap their bodies in transaction.Run(ctx, c.entitlementRepo, func(ctx){...}) so repo writes and event publishing share one tx. (`return transaction.Run(ctx, c.entitlementRepo, func(ctx context.Context) (*entitlement.Entitlement, error) {...})`)
**Dispatch to sub-type connector via getTypeConnector** — getTypeConnector(inp) switches on inp.GetType() to return metered/static/boolean SubTypeConnector; BeforeCreate/AfterCreate/GetValue always go through it. (`switch entitlementType { case entitlement.EntitlementTypeMetered: return c.meteredEntitlementConnector, nil ... }`)
**Uniqueness enforced with advisory lock + constraint check** — ScheduleEntitlement takes lockUniqueScope(customerID, featureKey) (lockr key fk/cid) then runs entitlement.ValidateUniqueConstraint over scheduled entitlements + a dummy, translating UniquenessConstraintError into AlreadyExistsError. (`err = c.lockUniqueScope(ctx, input.UsageAttribution.ID, feat.Key)`)
**Create/Override compose Schedule/Supersede** — CreateEntitlement forbids ActiveTo/ActiveFrom and calls ScheduleEntitlement then issues grants; OverrideEntitlement validates customer match then calls SupersedeEntitlement (DeactivateEntitlement + ScheduleEntitlement). (`ent, err := c.ScheduleEntitlement(ctx, input)`)
**Publish lifecycle events with customer payload** — Mutations publish NewEntitlementCreatedEventPayloadV2 / NewEntitlementDeletedEventPayloadV2 after loading the customer, inside the tx. (`err = c.publisher.Publish(ctx, entitlement.NewEntitlementCreatedEventPayloadV2(*ent, cust))`)
**Bounded-concurrency access fan-out** — GetAccess runs GetEntitlementValue per entitlement in an errgroup with a semaphore (maxConcurrency=10), collecting into sync.Map keyed by FeatureKey. (`sem := semaphore.NewWeighted(int64(maxConcurrency)); g.Go(func() error { ... })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | service struct, NewEntitlementService, all CRUD/list/value/access methods and getTypeConnector dispatch | GetEntitlementOfCustomerAt uses ulid.Parse to skip a guaranteed-miss ID lookup for feature keys; GetEntitlementValue returns NoAccessValue when !ent.IsActive(at) |
| `scheduling.go` | ScheduleEntitlement, SupersedeEntitlement, lockUniqueScope | Dummy entitlement with id "new-entitlement-id" is used for the constraint check; an inconsistency in scheduled entitlements (neither side is the dummy) is a hard error, not AlreadyExists |
| `lock.go` | NewEntitlementUniqueScopeLock builds the lockr.Key("fk", featureKey, "cid", customerID) | Key fields/order define the uniqueness scope — changing them changes the lock domain |
| `service_test.go / access_test.go / scheduling_test.go` | service_test integration suites (setupDependecies, createCustomerAndSubject, createMeterInPG) | Tests freeze time via clock.SetTime/ResetTime and require a meter row in PG (createMeterInPG) for metered access |
| `utils_test.go` | Shared test deps and mockTypeConnector/mockTypeValue | dependencies.Teardown closes db/drivers; reuse this harness instead of hand-rolling new setup |

## Anti-Patterns

- Performing entitlement writes without transaction.Run, so repo + event publish can diverge
- Hardcoding type behavior in the service instead of dispatching via getTypeConnector
- Scheduling without lockUniqueScope + ValidateUniqueConstraint, allowing overlapping entitlements
- Accepting ActiveFrom/ActiveTo in CreateEntitlement/OverrideEntitlement (they are rejected)
- Computing access serially or unbounded instead of the errgroup+semaphore fan-out

## Decisions

- **Create/Override are thin compositions over Schedule/Supersede** — All creation funnels through scheduling so the uniqueness lock + constraint check is enforced exactly once.
- **Uniqueness uses an advisory lock plus an in-memory constraint check** — Overlap rules span multiple scheduled rows and can't be expressed as a simple DB unique index, so the lock serializes the check.
- **GetAccess limits concurrency to 10** — Per-entitlement value computation (esp. metered balance) is expensive; the semaphore caps load while still parallelizing.

## Example: Transactional schedule with uniqueness lock and event publish

```
func (c *service) ScheduleEntitlement(ctx context.Context, input entitlement.CreateEntitlementInputs) (*entitlement.Entitlement, error) {
	return transaction.Run(ctx, c.entitlementRepo, func(ctx context.Context) (*entitlement.Entitlement, error) {
		if err := input.Validate(); err != nil { return nil, models.NewGenericValidationError(err) }
		feat, err := c.featureConnector.GetFeature(ctx, input.Namespace, *input.FeatureKey, feature.IncludeArchivedFeatureFalse)
		if err != nil { return nil, &feature.FeatureNotFoundError{ID: *input.FeatureKey} }
		if err := c.lockUniqueScope(ctx, input.UsageAttribution.ID, feat.Key); err != nil { return nil, err }
		connector, err := c.getTypeConnector(input)
		if err != nil { return nil, err }
		repoInputs, err := connector.BeforeCreate(input, *feat)
		if err != nil { return nil, err }
		ent, err := c.entitlementRepo.CreateEntitlement(ctx, *repoInputs)
		if err != nil { return nil, err }
		if err := connector.AfterCreate(ctx, ent); err != nil { return nil, err }
		return ent, c.publisher.Publish(ctx, entitlement.NewEntitlementCreatedEventPayloadV2(*ent, cust))
	})
// ...
```

<!-- archie:ai-end -->
