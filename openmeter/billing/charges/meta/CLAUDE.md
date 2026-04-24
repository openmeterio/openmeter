# meta

<!-- archie:ai-start -->

> Shared domain primitives for all charge types: ChargeID, ChargeType, ChargeStatus, ManagedResource, Intent, Expands, and the Patch interface (PatchDelete/Extend/Shrink) backed by stateless triggers. Also provides timestamp normalization (NormalizeTimestamp/NormalizeClosedPeriod) and the service.go type alias (Service = Adapter) — no separate service layer exists here.

## Patterns

**Patch interface + stateless.Trigger** — Every patch type implements Patch: Validate() + Trigger() stateless.Trigger + TriggerParams() any. Trigger constants live in triggers.go. New patch types must follow this pattern. (`type PatchExtend struct { ... }
func (p PatchExtend) Trigger() stateless.Trigger { return TriggerExtend }
func (p PatchExtend) TriggerParams() any { return p }`)
**NormalizeTimestamp on all time.Time fields** — All timestamps stored in Intent and patches must pass through NormalizeTimestamp (UTC + Truncate(streaming.MinimumWindowSizeDuration)) before persistence. Intent.Normalized() calls this for all period fields. (`i.ServicePeriod = NormalizeClosedPeriod(i.ServicePeriod)`)
**ChargeID = models.NamespacedID alias** — ChargeID is a type alias of models.NamespacedID; validate with ChargeID.Validate() which delegates to NamespacedID.Validate(). (`func (i ChargeID) Validate() error { return models.NamespacedID(i).Validate() }`)
**Service = Adapter type alias** — meta.Service is a type alias for meta.Adapter — no separate service implementation exists. If business logic is needed here in the future, add transaction.Run and a proper service layer. (`type Service = Adapter`)
**ValidateWith for period bounds checking** — PatchExtend and PatchShrink provide ValidateWith(intent Intent) to check new periods against existing intent periods. Always call this in addition to Validate(). (`if err := patch.ValidateWith(charge.Intent); err != nil { return err }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `charge.go` | ChargeID, ChargeType, ChargeStatus, Expand, Expands, ChargeAccessor, Charge, Charges enums and types. | ExpandNone is nil (not empty slice); ChargeAccessor requires GetChargeID() and ErrorAttributes(). |
| `intent.go` | Shared Intent struct embedded in all charge intents. Validates ManagedBy, CustomerID, Currency, all three periods, TaxConfig, Subscription, UniqueReferenceID. | UniqueReferenceID must be nil or non-empty string; never empty string. |
| `patch.go` | Patch interface definition. | TriggerParams() any is required for stateless parameter passing. |
| `patchdelete.go` | PatchDelete + PatchDeletePolicy (CreditRefundPolicy, InvoiceRefundPolicy). RefundAsCreditsDeletePolicy is the recommended default. | Always set both refund policies; partial policy leaves the other at zero value which fails Validate(). |
| `timestamps.go` | NormalizeTimestamp, NormalizeOptionalTimestamp, NormalizeClosedPeriod, Intent.Normalized(). | Truncates to streaming.MinimumWindowSizeDuration, not time.Microsecond — distinct from the lineage package. |
| `triggers.go` | All stateless.Trigger constants shared across charge state machines. | New triggers must be added here; never define trigger strings inline in state machine configuration files. |
| `service.go` | Type alias Service = Adapter. | No service layer exists — if complex orchestration is added, this alias must be removed and replaced with a proper service. |
| `resource.go` | ManagedResource embeds NamespacedModel + ManagedModel + ID. GetChargeID() derives ChargeID. | ID must be non-empty; validate via ManagedResource.Validate(). |

## Anti-Patterns

- Defining charge-type-specific copies of Intent period fields instead of embedding meta.Intent.
- Calling NormalizeTimestamp with different truncation precision than streaming.MinimumWindowSizeDuration.
- Adding complex business logic to meta package — it is a shared primitives package; logic belongs in type-specific service packages.
- Using string literals for trigger constants instead of the vars in triggers.go.
- Creating a PatchDelete with only one of the two refund policies set — the other defaults to empty string and fails Validate().

## Decisions

- **Service = Adapter type alias with no separate service layer** — meta is a persistence-thin registry; until orchestration is needed here, an alias avoids premature abstraction.
- **Patch interface uses stateless.Trigger rather than a string** — Compile-time type safety with the stateless library; callers pass trigger + params to statemachine.Machine.FireAndActivate.
- **All timestamps truncated to streaming.MinimumWindowSizeDuration** — ClickHouse window aggregation requires consistent truncation granularity across all charge-related timestamps.

<!-- archie:ai-end -->
