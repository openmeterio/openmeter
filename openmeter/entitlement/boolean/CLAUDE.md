# boolean

<!-- archie:ai-start -->

> Boolean entitlement sub-type connector: a type whose access is always granted (HasAccess() returns true) once the entitlement is active. Implements entitlement.SubTypeConnector with no persistence of its own.

## Patterns

**SubTypeConnector implementation** — Connector embeds entitlement.SubTypeConnector and implements GetValue/BeforeCreate/AfterCreate; the service dispatches to it by EntitlementType. (`type Connector interface { entitlement.SubTypeConnector }`)
**Reject metered-only fields in BeforeCreate** — BeforeCreate forces EntitlementType=Boolean and returns InvalidValueError if MeasureUsageFrom/IssueAfterReset/IsSoftLimit/Config are set. (`if model.MeasureUsageFrom != nil || model.Config != nil { return nil, &entitlement.InvalidValueError{Type: model.EntitlementType, Message: "Invalid inputs for type"} }`)
**Parse-from-generic type guard** — ParseFromGenericEntitlement checks EntitlementType and returns WrongTypeError on mismatch before exposing the typed Entitlement. (`if model.EntitlementType != entitlement.EntitlementTypeBoolean { return nil, &entitlement.WrongTypeError{...} }`)
**Compute currentUsagePeriod when a UsagePeriod is supplied** — If model.UsagePeriod is set, GetValue().Validate() then GetPeriodAt(clock.Now()) seeds CurrentUsagePeriod in the repo inputs. (`calculatedPeriod, err := usagePeriod.GetValue().GetPeriodAt(clock.Now())`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `connector.go` | Stateless Connector (NewBooleanEntitlementConnector); BeforeCreate validation, GetValue returns &BooleanEntitlementValue{} | BooleanEntitlementValue.HasAccess() is hardcoded true — inactivity is handled by the service, not here |
| `entitlement.go` | Typed Entitlement wrapper over entitlement.GenericProperties + ParseFromGenericEntitlement | No extra fields; do not add metered/static-only state to this type |

## Anti-Patterns

- Adding balance/usage logic to a boolean connector — booleans have no balance
- Returning access decisions based on time here instead of letting the service gate on IsActive
- Allowing Config/MeasureUsageFrom on boolean creation

## Decisions

- **Connector is a stateless struct{} with no repo dependency** — Boolean entitlements carry no balance or grants, so all state lives in the generic entitlement row.

## Example: GetValue for a boolean entitlement

```
func (c *connector) GetValue(ctx context.Context, e *entitlement.Entitlement, at time.Time) (entitlement.EntitlementValue, error) {
	if _, err := ParseFromGenericEntitlement(e); err != nil { return nil, err }
	return &BooleanEntitlementValue{}, nil
}
```

<!-- archie:ai-end -->
