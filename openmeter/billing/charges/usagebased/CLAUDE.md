# usagebased

<!-- archie:ai-start -->

> Public domain contract package for usage-based charges: it declares the domain types (ChargeBase/Charge/Intent/State/Status, RealizationRun(s), DetailedLine(s), Expands, RatingEngine), the Adapter interface (five fine-grained sub-interfaces + entutils.TxCreator), the Service interface, the ledger-callback Handler interface, and ValidationIssue error sentinels. It is the boundary that the service/ and adapter/ subtrees implement against — no Ent, persistence, or business logic lives in the root.

## Patterns

**Root holds contracts; subtrees hold implementations** — The root .go files declare interfaces (Service, Adapter, Handler) and pure domain value types with Validate()/Normalized() methods only. service/ implements Service (state-machine-driven advancement, rating, LineEngine), adapter/ implements Adapter (Ent persistence). New persistence belongs in adapter/, new orchestration in service/, new shared types/contracts here. (`type Service interface { UsageBasedService; GetLineEngine() billing.LineEngine } and type Adapter interface { RealizationRunAdapter; ChargeAdapter; entutils.TxCreator }`)
**Input.Validate() gates every cross-boundary call** — Every Input/domain struct implements Validate() collecting errors via errors.Join wrapped in models.NewNillableGenericValidationError. Callers (service/ and adapter/) call Validate() before acting; AdvanceChargeInput additionally requires an expanded Customer and a valid MergedProfile that Validate() does not deeply guard. (`func (i CreateInput) Validate() error { var errs []error; if i.Namespace == "" { errs = append(errs, errors.New("namespace is required")) }; return models.NewNillableGenericValidationError(errors.Join(errs...)) }`)
**Adapter is a composition of five sub-interfaces + TxCreator** — Adapter embeds ChargeAdapter, RealizationRunAdapter, RealizationRunCreditAllocationAdapter, RealizationRunInvoiceUsageAdapter, RealizationRunPaymentAdapter, and entutils.TxCreator. New DB operations must be added to the matching sub-interface so callers (and the adapter/ subtree) that depend only on a narrower interface keep compiling. (`RealizationRunAdapter.UpsertRunDetailedLines(ctx, chargeID, runID, lines) — a run-detail operation, not a charge operation`)
**Dot-separated Status sub-states with ToMetaChargeStatus()** — Status constants use dot notation for sub-states (active.partial_invoice.started). ToMetaChargeStatus splits on the first dot to derive the canonical meta.ChargeStatus, keeping the meta layer stable as the usagebased state graph evolves. Every new sub-state must be added to Values(). (`split := strings.SplitN(string(s), ".", 2); metaStatus := meta.ChargeStatus(split[0])`)
**Voided billing history excluded before aggregation** — Runs that are deleted (DeletedAt != nil) or Type=RealizationRunTypeInvalidDueToUnsupportedCreditNote are voided audit-only history. Always call WithoutVoidedBillingHistory() before Sum(), MapToBillingMeteredQuantity, or BisectByTimestamp so they don't count toward billing or balances. (`func (r RealizationRuns) Sum() totals.Totals { return totals.Sum(lo.Map(r.WithoutVoidedBillingHistory(), ...)) }`)
**ValidationIssue sentinels with HTTP status in errors.go** — Business-rule violations are package-level models.ValidationIssue vars carrying commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest). Use these sentinels instead of fmt.Errorf so the commonhttp encoder maps them to 4xx instead of 500. (`var ErrChargeTotalIsNegative = models.NewValidationIssue(ErrCodeChargeTotalIsNegative, "charge total is negative", models.WithCriticalSeverity(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest))`)
**mo.Option update inputs + UnimplementedHandler embedding** — UpdateRealizationRunInput uses mo.Option fields so the adapter treats absent options as no-op; set fields via mo.Some(...). Partial Handler implementations embed UnimplementedHandler so newly added Handler methods don't break existing implementers. (`r.StoredAtLT = mo.Some(meta.NormalizeTimestamp(storedAtLT)) ; type partialHandler struct { usagebased.UnimplementedHandler }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Declares Service (= UsageBasedService + GetLineEngine), all service Input/Output types, and validateExpands. AdvanceChargeInput requires both expanded Customer and MergedProfile. | Omitting Customer or MergedProfile in AdvanceChargeInput passes Validate() but panics at runtime inside the service state machine. |
| `adapter.go` | Declares the Adapter interface and its five sub-interfaces. New DB operations go into the matching sub-interface (ChargeAdapter, RealizationRunAdapter, etc.), never directly onto Adapter. | Adding a method directly to Adapter breaks callers/implementers that embed only the narrower sub-interface. |
| `charge.go` | Core domain types ChargeBase, Charge, Intent, State, Expands. GetFeatureKeyOrID has status-aware dispatch: created->key, deleted prefers State.FeatureID then falls back to key, all other states use ID. | GetFeatureKeyOrID status dispatch must stay in sync with usagebased/service feature resolution — a silent mismatch causes feature lookup failures. |
| `statemachine.go` | Status type, all status constants (including dot-separated sub-states), Values(), Validate(), ToMetaChargeStatus(), and IsMutableFinalRealizationStatus / IsMutableInvoiceBackedRealizationStatus classifiers. | New sub-states must be added to Values() AND the mutable-status slices, or the dot-split classification and mutability checks silently go wrong. |
| `realizationrun.go` | RealizationRun(Base), RealizationRuns with BisectByTimestamp / MapToBillingMeteredQuantity / Sum / WithoutVoidedBillingHistory. Carries the cumulative-to-line-period quantity mapping logic. | RealizationRunTypeInvalidDueToUnsupportedCreditNote cannot be a Create or InitialType. UpdateRealizationRunInput fields must use mo.Some(...) or the adapter no-ops them. |
| `handler.go` | Handler interface for ledger callbacks (OnInvoiceUsageAccrued, OnPaymentAuthorized/Settled, OnCreditsOnlyUsageAccrued + Correction) plus the UnimplementedHandler base and its Input types. | New Handler methods must be added to UnimplementedHandler or partial implementations fail to compile. |
| `errors.go` | Package-level ValidationIssue sentinels with HTTP status codes for domain business-rule violations. | Generic fmt.Errorf returns for these violations produce 500 instead of 400 at the HTTP boundary — reuse the sentinel. |
| `rating.go` | RateableIntent adapts Intent to rating.StandardLineAccessor for the rating engine. IsProgressivelyBilled always returns false and GetPreviouslyBilledAmount always returns zero. | Charges are never progressively billed; adding progressive-billing logic here breaks the rating invariant the service relies on. |

## Anti-Patterns

- Adding Ent or other persistence imports to the root package — it is a pure contract package; all DB access belongs in usagebased/adapter and orchestration in usagebased/service.
- Using fmt.Errorf for domain business-rule violations that have a ValidationIssue sentinel in errors.go — callers rely on error type/HTTP-status mapping.
- Adding a method directly to Adapter instead of the appropriate fine-grained sub-interface — breaks implementers/callers embedding only the narrower interface.
- Setting UpdateRealizationRunInput fields without mo.Some() — absent options are silently treated as no-op updates by the adapter.
- Aggregating or rating over raw RealizationRuns without WithoutVoidedBillingHistory() — deleted and unsupported-credit-note runs corrupt billing totals and balance calculations.

## Decisions

- **Split the package into a contract root plus service/ and adapter/ subtrees that implement Service and Adapter.** — Keeps domain types and interfaces free of Ent/persistence imports so the rating, run, and persistence layers can evolve independently while every cross-layer call is type-checked against the same contracts.
- **Dot-separated Status sub-states (active.partial_invoice.started) reduced to a stable meta.ChargeStatus via ToMetaChargeStatus.** — Lets the usagebased state graph add fine-grained transient states without churning the shared meta layer; the top-level status is derived by splitting on the first dot.
- **RealizationRunTypeInvalidDueToUnsupportedCreditNote is retained as voided billing history rather than hard-deleted.** — Audit trail requires the run to persist after its invoice line is removed; IsVoidedBillingHistory() excludes it from future rating and balance calculations without losing the record.

## Example: Adding a new service-layer input type for a usage-based charge operation, following the Validate() contract used across the package.

```
import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SnapshotUsageInput struct {
	ChargeID meta.ChargeID
	AsOf     time.Time
}

func (i SnapshotUsageInput) Validate() error {
// ...
```

<!-- archie:ai-end -->
