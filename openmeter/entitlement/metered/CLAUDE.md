# metered

<!-- archie:ai-start -->

> Metered entitlement sub-type: implements credit-backed grant burn-down tracking via the credit engine and ClickHouse usage queries, owns balance queries, usage resets, grant lifecycle, and the grant.OwnerConnector bridge.

## Patterns

**connector wires all credit sub-systems via constructor** — NewMeteredEntitlementConnector takes streamingConnector, ownerConnector, balanceConnector, grantConnector, grantRepo, entitlementRepo, publisher, logger, tracer. Wire provides these. Never instantiate connector directly in production code. (`meteredentitlement.NewMeteredEntitlementConnector(streamingConnector, ownerConnector, balanceConnector, grantConnector, grantRepo, entitlementRepo, publisher, logger, tracer)`)
**ResetEntitlementUsage runs inside transaction.Run** — ResetEntitlementUsage wraps the balance reset and usage-reset row write in transaction.Run(ctx, grantRepo, ...) for atomicity. Hooks and event publish happen inside the same transaction closure. (`return transaction.Run(ctx, e.grantRepo, func(ctx context.Context) (*EntitlementBalance, error) { e.balanceConnector.ResetUsageForOwner(ctx, ...); e.publisher.Publish(ctx, event) })`)
**ParseFromGenericEntitlement strict type guard** — Every connector method calls ParseFromGenericEntitlement immediately and returns WrongTypeError on mismatch. Required fields (MeasureUsageFrom, IsSoftLimit, UsagePeriod, LastReset, CurrentUsagePeriod) are validated inside the parse function. (`mEnt, err := ParseFromGenericEntitlement(ent); if err != nil { return nil, err }`)
**EndCurrentUsagePeriod requires active transaction in ctx** — EndCurrentUsagePeriod in grant_owner_adapter.go checks transaction.GetDriverFromContext before running. Callers must start a transaction before calling this method or it returns an error. (`_, err := transaction.GetDriverFromContext(ctx); if err != nil { return fmt.Errorf("end current usage period must be called in a transaction: %w", err) }`)
**Emit EntitlementResetEventV3 — never v1** — EntitlementResetEvent (v1) is deprecated. New code must emit EntitlementResetEventV3 which carries CustomerID. Events implement marshaler.Event with a pinned event name including version suffix. (`event := EntitlementResetEventV3{EntitlementID: ent.ID, CustomerID: ent.CustomerID, ...}; e.publisher.Publish(ctx, event)`)
**ServiceHookRegistry for cross-domain lifecycle** — connector embeds models.ServiceHookRegistry[Entitlement] and calls hooks.PreUpdate before mutations (CreateGrant, ResetEntitlementUsage). RegisterHooks is exposed on the Connector interface for balance-worker to register. (`if err := e.hooks.PreUpdate(ctx, metered); err != nil { return EntitlementGrant{}, err }`)
**queryMeter short-circuit for zero-length periods** — queryMeter returns a synthetic zero row immediately when params.From == params.To without calling ClickHouse. This prevents degenerate queries during edge-case period calculations. (`if params.From != nil && params.To != nil && params.From.Equal(*params.To) { return []meter.MeterQueryRow{{Value: 0, WindowStart: *params.From, WindowEnd: *params.To}}, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `connector.go` | Defines Connector interface and struct, constructor. AfterCreate issues a default grant via grantConnector when HasDefaultGrant() is true. | AfterCreate issues a default grant outside a transaction — ensure idempotency if called more than once. |
| `balance.go` | GetEntitlementBalance and GetEntitlementBalanceHistory. Balance queries round up to nearest minute. History fills in zero-usage windows that ClickHouse omits. | The zero-window fill loop (for current := start; current.Before(end); current, _ = WindowSize.AddTo(current)) must filter rows before MeasureUsageFrom — missing this filter returns stale windows. |
| `reset.go` | ResetEntitlementUsage and batch ResetEntitlementsWithExpiredUsagePeriod. Reset runs in transaction.Run wrapping grantRepo. | ResetEntitlementsWithExpiredUsagePeriod uses errors.Join to collect per-entitlement errors but continues iterating — partial failures are possible and expected. |
| `grant_owner_adapter.go` | entitlementGrantOwner implements grant.OwnerConnector: DescribeOwner resolves customer/meter/feature; EndCurrentUsagePeriod writes a usage_reset row and updates current period. | getRepoMaybeInTx is a documented workaround to rebind featureRepo to ctx transaction. LockOwnerForTx requires an active transaction in ctx. |
| `events.go` | EntitlementResetEvent (deprecated v1) and EntitlementResetEventV3. Both implement marshaler.Event with pinned EventName strings. | New events must implement Validate() and use metadata.GetEventName with a versioned EventType; check event subsystem constant aligns with watermill/eventbus routing. |
| `repository.go` | UsageResetRepo interface and UsageResetUpdate with Validate(). UsageResetUpdate.UsagePeriodInterval must be a valid ISO duration. | Validate() is called by the adapter before insert; always keep it in sync with schema constraints in openmeter/ent/schema. |

## Anti-Patterns

- Bypassing transaction.Run in ResetEntitlementUsage — balance reset and event publish must be atomic.
- Emitting EntitlementResetEvent v1 from new code — always use EntitlementResetEventV3.
- Calling EndCurrentUsagePeriod outside of a transaction context.
- Adding credit engine logic directly to connector methods without going through balanceConnector/grantConnector interfaces.
- Using context.Background() in production connector methods — always propagate caller ctx.

## Decisions

- **Balance history fills in zero-usage windows missing from ClickHouse results.** — ClickHouse only returns rows with non-zero usage; the API contract requires a window entry for every time bucket in the queried range.
- **ownerCustomer adapter in owner_customer.go wraps customer data without importing openmeter/customer into the credit package.** — Prevents circular imports between meteredentitlement and credit packages while satisfying streaming.Customer interface required by grant.Owner.
- **IssueAfterReset with amount>0 causes AfterCreate to issue a default recurring grant.** — Allows operators to configure a standing balance that replenishes automatically each usage period without manual grant operations.

## Example: Wire and call the metered entitlement connector in tests

```
connector := meteredentitlement.NewMeteredEntitlementConnector(
	streamingConnector,
	ownerConnector,
	creditConnector, // implements credit.BalanceConnector
	creditConnector, // implements credit.GrantConnector
	grantRepo,
	entitlementRepo,
	mockPublisher,
	testLogger,
	tracer,
)
connector.RegisterHooks(meteredentitlement.ConvertHook(subscriptionHook))
balance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: ns, ID: entID}, time.Now())
```

<!-- archie:ai-end -->
