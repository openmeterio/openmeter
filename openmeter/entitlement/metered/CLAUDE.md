# metered

<!-- archie:ai-start -->

> Metered entitlement sub-system: the Connector that ties metered entitlements to credit grants, balance/overage calculation, usage resets, and the grant.OwnerConnector adapter. Bridges entitlements to openmeter/credit (balance/grant engine) and openmeter/streaming (usage).

## Patterns

**Connector aggregates credit + streaming + grant deps** — The connector struct holds streamingConnector, ownerConnector, balanceConnector, grantConnector, grantRepo, entitlementRepo, publisher, hooks, logger, tracer; built via NewMeteredEntitlementConnector(...). (`func NewMeteredEntitlementConnector(streamingConnector streaming.Connector, ownerConnector grant.OwnerConnector, balanceConnector credit.BalanceConnector, ...) Connector`)
**ParseFromGenericEntitlement before any metered op** — Every method that needs metered fields first calls ParseFromGenericEntitlement to assert EntitlementTypeMetered and presence of MeasureUsageFrom/UsagePeriod/LastReset/CurrentUsagePeriod. (`metered, err := ParseFromGenericEntitlement(entRepoEntity)`)
**Balance via credit engine + snapshots** — GetEntitlementBalance defers to balanceConnector.GetBalanceAt/GetBalanceForPeriod (credit engine), reading res.Snapshot.Balance()/Usage/Overage rather than computing from raw events. (`res, err := e.balanceConnector.GetBalanceAt(ctx, nsOwner, at)`)
**OpenTelemetry span per operation** — Public methods open e.tracer.Start(ctx, "meteredentitlement.X", ...) with defer span.End(); trace.go provides mtrace.WithOwner/WithPeriod option helpers. (`ctx, span := e.tracer.Start(ctx, "meteredentitlement.GetEntitlementBalance", trace.WithAttributes(...)); defer span.End()`)
**OwnerConnector adapter implements grant ownership** — entitlementGrantOwner (grant_owner_adapter.go) implements grant.OwnerConnector: DescribeOwner, GetUsagePeriodStartAt, GetResetTimelineInclusive, EndCurrentUsagePeriod, LockOwnerForTx so the credit engine can resolve entitlement owners. (`func (e *entitlementGrantOwner) GetUsagePeriodStartAt(ctx, owner, at) (time.Time, error)`)
**Default grant on AfterCreate** — AfterCreate issues a default grant via grantConnector.CreateGrant with ResetMaxRollover=ResetMinRollover=amount and IssueAfterResetMetaTag annotation when HasDefaultGrant(). (`Annotations: models.Annotations{IssueAfterResetMetaTag: true}`)
**Hook adapter bridges generic and typed hooks** — hook.go ConvertHook wraps a models.ServiceHook[entitlement.Entitlement] into a ServiceHook[Entitlement] by re-parsing the typed entitlement in each Pre/Post method. (`func ConvertHook(h models.ServiceHook[entitlement.Entitlement]) models.ServiceHook[Entitlement]`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `connector.go` | Connector interface + connector struct, BeforeCreate/AfterCreate, GetValue, MeteredEntitlementValue (HasAccess: soft-limit OR balance>0) | granularity is hardcoded time.Minute (FIXME); BeforeCreate truncates anchor and forces UsagePeriod.From to MeasureUsageFrom |
| `balance.go` | GetEntitlementBalance and GetEntitlementBalanceHistory (window filling + segment merge) | Heavy minute-truncation/window-fill logic with FIXMEs; queryMeter shortcuts a zero-length period to 0; ClickHouse only returns non-empty windows so gaps are filled manually |
| `reset.go` | ResetEntitlementUsage and ResetEntitlementsWithExpiredUsagePeriod | Reset must keep usage-period anchor/highwatermark logic consistent with grant rollover |
| `entitlement_grant.go` | CreateGrant/ListEntitlementGrants and EntitlementGrant wrapper over grant.Grant | Entitlement resolved by ID then falls back to GetActiveEntitlementOfCustomerAt by feature key; PreUpdate hook fired before CreateGrant |
| `grant_owner_adapter.go` | entitlementGrantOwner implementing grant.OwnerConnector for the credit engine | GetResetTimelineInclusive/EndCurrentUsagePeriod must stay aligned with usage-reset persistence |
| `repository.go` | UsageResetRepo interface, UsageResetUpdate (with Validate) and UsageResetNotFoundError | Implemented by the adapter package; keep Validate in sync with adapter Save |
| `events.go` | EntitlementResetEvent (v1 deprecated) / EntitlementResetEventV3 Watermill events | v1 keyed on Subject, v3 on CustomerID; pick V3 for new code and validate Namespace/Subject |
| `entitlement.go` | Typed metered Entitlement, IssueAfterReset, HasDefaultGrant, ParseFromGenericEntitlement / ToGenericEntitlement | ParseFromGenericEntitlement requires many non-nil fields — missing any yields InvalidValueError |

## Anti-Patterns

- Computing balance/overage from raw meter rows instead of the credit balanceConnector/engine
- Skipping ParseFromGenericEntitlement and reading metered fields off the generic entitlement
- Emitting the deprecated EntitlementResetEvent (v1) for new flows instead of EntitlementResetEventV3
- Calling grantConnector.CreateGrant without firing the PreUpdate hook on grant creation
- Bypassing the tracer (e.tracer.Start) on public connector operations

## Decisions

- **Metered balance delegates to the credit engine and snapshots** — Balance, overage and grant burn-down are credit-domain concerns; the entitlement connector only orchestrates and shapes EntitlementBalance.
- **An OwnerConnector adapter lives here rather than in credit** — Mapping a credit grant owner onto an entitlement (usage period, reset timeline, measurement start) needs entitlement knowledge, so it is implemented next to the connector.
- **Default grants are issued in AfterCreate** — IssueAfterReset semantics (rollover=amount) must run after the entitlement row exists so the grant has a valid owner.

## Example: Connector reads balance from the credit engine

```
func (e *connector) GetEntitlementBalance(ctx context.Context, id models.NamespacedID, at time.Time) (*EntitlementBalance, error) {
	ctx, span := e.tracer.Start(ctx, "meteredentitlement.GetEntitlementBalance")
	defer span.End()
	nsOwner := models.NamespacedID{Namespace: id.Namespace, ID: id.ID}
	startOfPeriod, err := e.ownerConnector.GetUsagePeriodStartAt(ctx, nsOwner, at)
	if err != nil { return nil, err }
	res, err := e.balanceConnector.GetBalanceAt(ctx, nsOwner, at)
	if err != nil { return nil, err }
	return &EntitlementBalance{EntitlementID: id.ID, Balance: res.Snapshot.Balance(), Overage: res.Snapshot.Overage, StartOfPeriod: startOfPeriod}, nil
}
```

<!-- archie:ai-end -->
