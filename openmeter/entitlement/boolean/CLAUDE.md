# boolean

<!-- archie:ai-start -->

> Boolean entitlement sub-type: implements entitlement.SubTypeConnector for on/off access entitlements that carry no balance or usage tracking. Single responsibility: validate creation inputs and always return HasAccess=true.

## Patterns

**SubTypeConnector implementation** — Implement entitlement.SubTypeConnector (GetValue, BeforeCreate, AfterCreate). BeforeCreate validates that no metered fields (MeasureUsageFrom, IssueAfterReset, IsSoftLimit, Config) are set and returns CreateEntitlementRepoInputs. (`func (c *connector) BeforeCreate(model entitlement.CreateEntitlementInputs, feature feature.Feature) (*entitlement.CreateEntitlementRepoInputs, error) { ... }`)
**ParseFromGenericEntitlement type guard** — ParseFromGenericEntitlement checks EntitlementType == EntitlementTypeBoolean and returns WrongTypeError if not. Always call this at the top of GetValue. (`_, err := ParseFromGenericEntitlement(entitlement); if err != nil { return nil, err }`)
**AfterCreate no-op** — AfterCreate returns nil immediately — boolean entitlements require no post-creation side effects (no default grant, no balance snapshot). (`func (c *connector) AfterCreate(ctx context.Context, entitlement *entitlement.Entitlement) error { return nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `connector.go` | Full SubTypeConnector implementation: GetValue always returns BooleanEntitlementValue{HasAccess: true}; BeforeCreate rejects metered-only fields. | UsagePeriod IS allowed for boolean entitlements — BeforeCreate computes currentUsagePeriod if UsagePeriod is set. |
| `entitlement.go` | Defines Entitlement struct (embeds GenericProperties) and ParseFromGenericEntitlement type-narrowing function. | Struct has no extra fields beyond GenericProperties — do not add balance fields here. |

## Anti-Patterns

- Adding balance or usage fields to the boolean Entitlement struct.
- Returning HasAccess=false from BooleanEntitlementValue — boolean access is always true.
- Calling the credit engine from AfterCreate — boolean entitlements have no grants.

## Decisions

- **Boolean entitlement always returns HasAccess=true without any credit computation.** — Boolean entitlements are pure feature flags; there is no concept of credit burn-down, so the value is always true as long as the entitlement exists and is active.

<!-- archie:ai-end -->
