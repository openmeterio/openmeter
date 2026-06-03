# boolean

<!-- archie:ai-start -->

> Boolean entitlement sub-type implementing entitlement.SubTypeConnector for on/off feature-flag access — carries no balance or usage tracking and GetValue always returns HasAccess=true.

## Patterns

**SubTypeConnector three-method implementation** — Implement exactly GetValue, BeforeCreate, AfterCreate (no-op). Do not add a fourth method. (`func (c *connector) AfterCreate(ctx context.Context, entitlement *entitlement.Entitlement) error { return nil }`)
**ParseFromGenericEntitlement type guard first** — GetValue calls ParseFromGenericEntitlement immediately; it checks EntitlementType==EntitlementTypeBoolean and returns WrongTypeError otherwise. (`_, err := ParseFromGenericEntitlement(entitlement); if err != nil { return nil, err }`)
**BeforeCreate computes currentUsagePeriod when UsagePeriod set** — Boolean entitlements support UsagePeriod for scheduling; BeforeCreate calculates currentUsagePeriod via usagePeriod.GetValue().GetPeriodAt(clock.Now()) when non-nil. (`calculatedPeriod, err := usagePeriod.GetValue().GetPeriodAt(clock.Now()); currentUsagePeriod = &calculatedPeriod`)
**Reject metered-only fields in BeforeCreate** — BeforeCreate returns InvalidValueError if MeasureUsageFrom, IssueAfterReset, IsSoftLimit, or Config are set. UsagePeriod is explicitly allowed. (`if model.MeasureUsageFrom != nil || model.IssueAfterReset != nil || model.IsSoftLimit != nil || model.Config != nil { return nil, &entitlement.InvalidValueError{...} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `connector.go` | Full SubTypeConnector impl. GetValue returns BooleanEntitlementValue{} (HasAccess()==true). BeforeCreate rejects metered-only fields. | UsagePeriod IS allowed for boolean entitlements — do not add it to the rejection list. |
| `entitlement.go` | Entitlement struct (embeds GenericProperties only) and ParseFromGenericEntitlement type-narrowing function. | Do not add balance/usage fields — the struct is intentionally empty beyond GenericProperties. |

## Anti-Patterns

- Adding balance or usage fields to the boolean Entitlement struct.
- Returning HasAccess=false from BooleanEntitlementValue.
- Calling a credit engine or grant repo from AfterCreate — boolean entitlements have no grants.
- Adding MeasureUsageFrom/IssueAfterReset to the BeforeCreate acceptance list.

## Decisions

- **Boolean entitlement always returns HasAccess=true without credit computation.** — Boolean entitlements are pure feature flags with no credit burn-down; access is true as long as the entitlement is active.

<!-- archie:ai-end -->
