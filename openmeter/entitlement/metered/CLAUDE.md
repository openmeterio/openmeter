# metered

<!-- archie:ai-start -->

> Metered entitlement sub-type implementing credit-backed grant burn-down via the credit engine and ClickHouse usage queries — owns balance queries, usage resets, grant lifecycle, and the grant.OwnerConnector bridge.

## Patterns

**Constructor wires all credit sub-systems** — NewMeteredEntitlementConnector takes streaming/owner/balance/grant connectors, repos, publisher, logger, tracer (all Wire-provided). Never instantiate directly in production. (`meteredentitlement.NewMeteredEntitlementConnector(streamingConnector, ownerConnector, balanceConnector, grantConnector, grantRepo, entitlementRepo, publisher, logger, tracer)`)
**ResetEntitlementUsage runs inside transaction.Run** — Balance reset, usage-reset write, hooks, and event publish all happen inside one transaction.Run(ctx, grantRepo, ...) closure for atomicity. (`transaction.Run(ctx, e.grantRepo, func(ctx) (*EntitlementBalance, error) { e.balanceConnector.ResetUsageForOwner(...); e.publisher.Publish(ctx, event) })`)
**ParseFromGenericEntitlement strict type guard** — Every connector method calls ParseFromGenericEntitlement immediately, validating required fields (MeasureUsageFrom, IsSoftLimit, UsagePeriod, LastReset, CurrentUsagePeriod) and returning WrongTypeError on mismatch. (`mEnt, err := ParseFromGenericEntitlement(ent); if err != nil { return nil, err }`)
**EndCurrentUsagePeriod requires active transaction** — grant_owner_adapter.go EndCurrentUsagePeriod checks transaction.GetDriverFromContext and errors if no tx is present. (`if _, err := transaction.GetDriverFromContext(ctx); err != nil { return fmt.Errorf("end current usage period must be called in a transaction: %w", err) }`)
**Emit EntitlementResetEventV3, never v1** — EntitlementResetEvent (v1) is deprecated. New code emits EntitlementResetEventV3 which carries CustomerID; events implement marshaler.Event with a version-pinned EventName. (`e.publisher.Publish(ctx, EntitlementResetEventV3{EntitlementID: ent.ID, CustomerID: ent.CustomerID})`)
**ServiceHookRegistry for cross-domain lifecycle** — connector embeds models.ServiceHookRegistry[Entitlement]; calls hooks.PreUpdate before mutations and exposes RegisterHooks for balance-worker registration. (`if err := e.hooks.PreUpdate(ctx, metered); err != nil { return EntitlementGrant{}, err }`)
**queryMeter zero-length short-circuit** — queryMeter returns a synthetic zero row when params.From==params.To without calling ClickHouse, avoiding degenerate edge-case queries. (`if params.From != nil && params.To != nil && params.From.Equal(*params.To) { return []meter.MeterQueryRow{{Value: 0, WindowStart: *params.From, WindowEnd: *params.To}}, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `connector.go` | Connector interface, struct, constructor. AfterCreate issues a default grant when HasDefaultGrant() is true. | AfterCreate issues the default grant outside a transaction — ensure idempotency if it may be called twice. |
| `balance.go` | GetEntitlementBalance / GetEntitlementBalanceHistory. Rounds up to nearest minute; fills zero-usage windows ClickHouse omits. | The zero-window fill loop must filter rows before MeasureUsageFrom or it returns stale windows. |
| `reset.go` | ResetEntitlementUsage and batch ResetEntitlementsWithExpiredUsagePeriod inside transaction.Run. | Batch reset uses errors.Join to collect per-entitlement errors but keeps iterating — partial failures are expected. |
| `grant_owner_adapter.go` | entitlementGrantOwner implements grant.OwnerConnector (DescribeOwner, EndCurrentUsagePeriod, LockOwnerForTx). | getRepoMaybeInTx rebinds featureRepo to ctx tx; LockOwnerForTx and EndCurrentUsagePeriod require an active transaction. |
| `events.go` | EntitlementResetEvent (deprecated v1) and EntitlementResetEventV3, both marshaler.Event with pinned EventName. | New events need Validate() and a versioned EventType whose subsystem aligns with watermill/eventbus routing. |
| `repository.go` | UsageResetRepo interface and UsageResetUpdate.Validate (ISO-duration UsagePeriodInterval). | Keep Validate() in sync with openmeter/ent/schema constraints; the adapter calls it before insert. |

## Anti-Patterns

- Bypassing transaction.Run in ResetEntitlementUsage — reset and event publish must be atomic.
- Emitting EntitlementResetEvent v1 from new code — always use V3.
- Calling EndCurrentUsagePeriod outside a transaction context.
- Adding credit-engine logic directly to connector methods instead of via balanceConnector/grantConnector.
- Using context.Background() in production connector methods.

## Decisions

- **Balance history fills zero-usage windows missing from ClickHouse results.** — ClickHouse only returns non-zero rows; the API contract requires an entry for every time bucket.
- **ownerCustomer adapter wraps customer data without importing openmeter/customer into credit.** — Prevents circular imports while satisfying the streaming.Customer interface required by grant.Owner.
- **IssueAfterReset amount>0 triggers AfterCreate to issue a default recurring grant.** — Lets operators configure a standing balance that replenishes each usage period without manual grant ops.

## Example: Wire and call the metered connector in tests

```
connector := meteredentitlement.NewMeteredEntitlementConnector(streamingConnector, ownerConnector, creditConnector, creditConnector, grantRepo, entitlementRepo, mockPublisher, testLogger, tracer)
connector.RegisterHooks(meteredentitlement.ConvertHook(subscriptionHook))
balance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: ns, ID: entID}, time.Now())
```

<!-- archie:ai-end -->
