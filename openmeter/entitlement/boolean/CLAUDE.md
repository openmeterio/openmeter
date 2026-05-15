# boolean

<!-- archie:ai-start -->

> Boolean entitlement sub-type implementing entitlement.SubTypeConnector for on/off access entitlements that carry no balance or usage tracking — always returns HasAccess=true.

## Patterns

**SubTypeConnector interface implementation** — Implement all three methods of entitlement.SubTypeConnector: GetValue, BeforeCreate (validates inputs and returns CreateEntitlementRepoInputs), AfterCreate (no-op for boolean). Do not add a fourth method. (`func (c *connector) AfterCreate(ctx context.Context, entitlement *entitlement.Entitlement) error { return nil }`)
**ParseFromGenericEntitlement type guard at top of GetValue** — Call ParseFromGenericEntitlement immediately in GetValue. It checks EntitlementType == EntitlementTypeBoolean and returns WrongTypeError if not. Skip this and type misrouting is silently swallowed. (`_, err := ParseFromGenericEntitlement(entitlement); if err != nil { return nil, err }`)
**BeforeCreate computes currentUsagePeriod if UsagePeriod is set** — Boolean entitlements do support UsagePeriod for scheduling. BeforeCreate must calculate currentUsagePeriod via usagePeriod.GetValue().GetPeriodAt(clock.Now()) when UsagePeriod is non-nil. (`if model.UsagePeriod != nil { calculatedPeriod, err := usagePeriod.GetValue().GetPeriodAt(clock.Now()); currentUsagePeriod = &calculatedPeriod }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `connector.go` | Full SubTypeConnector implementation. GetValue always returns BooleanEntitlementValue{HasAccess: true}. BeforeCreate rejects metered-only fields (MeasureUsageFrom, IssueAfterReset, IsSoftLimit, Config). | UsagePeriod IS allowed for boolean entitlements. Do not add it to the rejection list in BeforeCreate. |
| `entitlement.go` | Defines the Entitlement struct (embeds GenericProperties only) and ParseFromGenericEntitlement type-narrowing function. | Do not add balance or usage fields here — the struct is intentionally empty beyond GenericProperties. |

## Anti-Patterns

- Adding balance or usage fields to the boolean Entitlement struct.
- Returning HasAccess=false from BooleanEntitlementValue — boolean access is always true.
- Calling a credit engine or grant repo from AfterCreate — boolean entitlements have no grants.
- Adding MeasureUsageFrom or IssueAfterReset validation to BeforeCreate acceptance list.

## Decisions

- **Boolean entitlement always returns HasAccess=true without credit computation.** — Boolean entitlements are pure feature flags — there is no concept of credit burn-down, so the value is always true as long as the entitlement exists and is active.

<!-- archie:ai-end -->
