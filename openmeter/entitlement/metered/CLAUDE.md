# metered

<!-- archie:ai-start -->

> Metered entitlement sub-type implementing credit-backed grant burn-down tracking. Owns balance queries (GetEntitlementBalance, GetEntitlementBalanceHistory), usage resets, grant lifecycle (CreateGrant, ListEntitlementGrants), and the grant.OwnerConnector bridge that feeds the credit engine. This is the most complex sub-type.

## Patterns

**connector struct wires credit sub-systems** — connector holds streamingConnector, ownerConnector, balanceConnector, grantConnector, grantRepo, entitlementRepo. NewMeteredEntitlementConnector takes all of these; Wire provides them. Never instantiate connector directly in production code. (`meteredentitlement.NewMeteredEntitlementConnector(streamingConnector, ownerConnector, balanceConnector, grantConnector, grantRepo, entitlementRepo, publisher, logger, tracer)`)
**ResetEntitlementUsage runs inside a transaction** — ResetEntitlementUsage calls transaction.Run(ctx, e.grantRepo, ...) to ensure the balance reset and usage-reset row write are atomic. Hooks and event publish happen inside the same transaction closure. (`return transaction.Run(ctx, e.grantRepo, func(ctx context.Context) (*EntitlementBalance, error) { ... e.balanceConnector.ResetUsageForOwner ... e.publisher.Publish ... })`)
**ParseFromGenericEntitlement strict type guard** — Every connector method that works with a specific sub-type calls ParseFromGenericEntitlement immediately and returns WrongTypeError if the type doesn't match. Required fields (MeasureUsageFrom, IsSoftLimit, UsagePeriod, LastReset, CurrentUsagePeriod, OriginalUsagePeriodAnchor) are validated. (`mEnt, err := ParseFromGenericEntitlement(ent); if err != nil { return nil, err }`)
**EndCurrentUsagePeriod requires an active transaction** — EndCurrentUsagePeriod checks transaction.GetDriverFromContext before running — it must be called in a transaction. Callers (balance-worker reset path) are responsible for starting the transaction first. (`_, err := transaction.GetDriverFromContext(ctx); if err != nil { return fmt.Errorf("end current usage period must be called in a transaction: %w", err) }`)
**Event versioning: always emit V3 events** — EntitlementResetEvent (v1) is deprecated; new code must emit EntitlementResetEventV3 which carries CustomerID instead of Subject. Events implement marshaler.Event with pinned event name including version. (`event := EntitlementResetEventV3{EntitlementID: ..., CustomerID: ent.CustomerID, ...}; e.publisher.Publish(ctx, event)`)
**ServiceHookRegistry for cross-domain lifecycle** — connector embeds models.ServiceHookRegistry[Entitlement] and calls hooks.PreUpdate before mutations (CreateGrant, ResetEntitlementUsage). RegisterHooks is exposed on the Connector interface. (`if err := e.hooks.PreUpdate(ctx, metered); err != nil { return EntitlementGrant{}, err }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `connector.go` | Defines Connector interface and struct, constructor, GetValue, BeforeCreate, AfterCreate (issues default grant if IssueAfterReset>0). Central orchestration file. | AfterCreate issues a default grant via grantConnector when HasDefaultGrant() is true — this happens outside a transaction; ensure idempotency. |
| `balance.go` | GetEntitlementBalance and GetEntitlementBalanceHistory. Balance queries round up to the nearest minute. History fills in zero-usage windows that ClickHouse omits. | queryMeter short-circuits to a zero row if from==to, preventing ClickHouse calls for zero-length periods. |
| `reset.go` | ResetEntitlementUsage and batch ResetEntitlementsWithExpiredUsagePeriod. Reset runs in transaction.Run wrapping grantRepo. | ResetEntitlementsWithExpiredUsagePeriod uses errors.Join to collect per-entitlement errors but continues iterating — partial failures are possible. |
| `grant_owner_adapter.go` | entitlementGrantOwner implements grant.OwnerConnector: DescribeOwner resolves customer/meter/feature for a given entitlement ID; EndCurrentUsagePeriod writes a usage reset row and updates the stored current period. | getRepoMaybeInTx is a documented hack to rebind featureRepo to ctx transaction. LockOwnerForTx requires an active transaction in ctx. |
| `entitlement.go` | Metered-specific Entitlement struct with MeasureUsageFrom, IssueAfterReset, IsSoftLimit, UsagePeriod, CurrentUsagePeriod, LastReset. ParseFromGenericEntitlement does field-level nil checks. | IssueAfterResetPriority requires IssueAfterReset — ParseFromGenericEntitlement enforces this. |
| `events.go` | EntitlementResetEvent (deprecated v1) and EntitlementResetEventV3. Both implement marshaler.Event with pinned EventName strings. | New events must implement Validate() and use metadata.GetEventName with a versioned EventType. |
| `repository.go` | UsageResetRepo interface and UsageResetUpdate input type with Validate(). UsageResetUpdate.UsagePeriodInterval must be a valid ISO duration. | Validate is called by the adapter before inserting; always keep it in sync with schema constraints. |

## Anti-Patterns

- Bypassing transaction.Run in ResetEntitlementUsage — balance reset and event publish must be atomic.
- Emitting EntitlementResetEvent (v1) from new code — use EntitlementResetEventV3.
- Calling EndCurrentUsagePeriod outside of a transaction context.
- Adding credit engine logic directly to connector methods without going through balanceConnector/grantConnector interfaces.
- Using context.Background() in production connector methods — always propagate caller ctx.

## Decisions

- **Balance history fills in zero-usage windows missing from ClickHouse results.** — ClickHouse only returns rows with non-zero usage; the API contract requires a window entry for every time bucket in the queried range.
- **ownerCustomer adapter in owner_customer.go wraps customer data without importing openmeter/customer into the credit package.** — Prevents circular imports between meteredentitlement and credit packages while still satisfying streaming.Customer interface required by grant.Owner.
- **IssueAfterReset with amount>0 causes AfterCreate to issue a default recurring grant.** — Allows operators to configure a standing balance that automatically replenishes each usage period without manual grant operations.

## Example: Set up and call the metered entitlement connector in tests

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
connector.RegisterHooks(
	meteredentitlement.ConvertHook(subscriptionHook),
)
balance, err := connector.GetEntitlementBalance(ctx,
// ...
```

<!-- archie:ai-end -->
