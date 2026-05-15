# usagebased

<!-- archie:ai-start -->

> Public domain contract package for usage-based charges: declares domain types (Charge, ChargeBase, Intent, State, Status, RealizationRun, DetailedLines, Expands), the Adapter interface (composed of five sub-interfaces + entutils.TxCreator), the Service interface, the Handler interface for ledger callbacks, and structured ValidationIssue error sentinels. It is the boundary consumed by usagebased/service and usagebased/adapter — no Ent or persistence imports belong here.

## Patterns

**Input.Validate() gates every cross-boundary call** — Every Input struct implements Validate() that collects errors via errors.Join and wraps with models.NewNillableGenericValidationError. Callers must call Validate() before passing to service or adapter methods. (`func (i AdvanceChargeInput) Validate() error { var errs []error; if i.CustomerOverride.Customer == nil { errs = append(errs, errors.New("expanded customer is required")) }; return models.NewNillableGenericValidationError(errors.Join(errs...)) }`)
**Adapter composed of five fine-grained sub-interfaces + TxCreator** — Adapter embeds RealizationRunAdapter, RealizationRunCreditAllocationAdapter, RealizationRunInvoiceUsageAdapter, RealizationRunPaymentAdapter, ChargeAdapter, and entutils.TxCreator. New DB operations must be added to the matching sub-interface, not directly to Adapter. (`type Adapter interface { RealizationRunAdapter; ChargeAdapter; entutils.TxCreator }`)
**Dot-separated Status sub-states with ToMetaChargeStatus()** — Status constants use dot notation for sub-states (e.g. StatusActivePartialInvoiceStarted = "active.partial_invoice.started"). ToMetaChargeStatus splits on the first dot to extract the canonical meta.ChargeStatus. New sub-states must be added to Values(). (`split := strings.SplitN(string(s), ".", 2); metaStatus := meta.ChargeStatus(split[0])`)
**Expand dependency validation enforced in validateExpands** — validateExpands (unexported) enforces that ExpandDetailedLines requires ExpandRealizations, and ExpandDeletedRealizations requires ExpandRealizations. All input types that accept Expands must call validateExpands. (`if expands.Has(meta.ExpandDetailedLines) && !expands.Has(meta.ExpandRealizations) { return fmt.Errorf("%q requires %q", meta.ExpandDetailedLines, meta.ExpandRealizations) }`)
**ValidationIssue sentinels with HTTP status codes in errors.go** — Domain-specific business-rule violations are declared as package-level models.ValidationIssue vars with commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest). Use these sentinels instead of fmt.Errorf to get correct 4xx responses at the HTTP boundary. (`var ErrChargeTotalIsNegative = models.NewValidationIssue(ErrCodeChargeTotalIsNegative, "charge total is negative", models.WithCriticalSeverity(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest))`)
**UnimplementedHandler for compile-time interface coverage** — UnimplementedHandler implements Handler returning errors.New("not implemented") for every method. Partial handler implementations embed UnimplementedHandler so new Handler methods don't break compilation of existing partial implementations. (`var _ Handler = (*UnimplementedHandler)(nil)`)
**RealizationRuns.WithoutVoidedBillingHistory() before aggregation** — Deleted runs (DeletedAt != nil) and runs with Type=RealizationRunTypeInvalidDueToUnsupportedCreditNote are voided billing history. Always call WithoutVoidedBillingHistory() before Sum(), MapToBillingMeteredQuantity(), or BisectByTimestamp() to exclude them. (`func (r RealizationRuns) Sum() totals.Totals { return totals.Sum(lo.Map(r.WithoutVoidedBillingHistory(), ...)) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Declares the Adapter interface and its five sub-interfaces. New DB operations must go into the matching sub-interface (ChargeAdapter, RealizationRunAdapter, etc.), never directly on Adapter. | Adding a method directly to Adapter instead of the appropriate sub-interface breaks callers that embed only the narrower interface. |
| `charge.go` | Core domain types: ChargeBase, Charge, Intent, State, Expands. GetFeatureKeyOrID has status-aware dispatch: created→key, deleted prefers ID then falls back to key, all other states use ID. | GetFeatureKeyOrID status dispatch must stay in sync with usagebased/service feature resolution — silent mismatch causes feature lookup failures. |
| `service.go` | Declares Service (= UsageBasedService + GetLineEngine), all Input/Output types, and validateExpands. AdvanceChargeInput requires both expanded Customer and MergedProfile. | Omitting Customer or MergedProfile in AdvanceChargeInput passes Validate() but panics at runtime inside the state machine. |
| `statemachine.go` | Status type, all status constants including dot-separated sub-states, Values(), Validate(), ToMetaChargeStatus(). IsMutableFinalRealizationStatus and IsMutableInvoiceBackedRealizationStatus classify which states own mutable invoice lines. | New sub-states must be added to Values() and the dot-split logic in ToMetaChargeStatus must remain correct. |
| `realizationrun.go` | RealizationRun, RealizationRunBase, RealizationRuns with BisectByTimestamp, MapToBillingMeteredQuantity, Sum. UpdateRealizationRunInput uses mo.Option fields. | UpdateRealizationRunInput fields must be set via mo.Some(...) — absent options are silently no-op'd by the adapter. RealizationRunTypeInvalidDueToUnsupportedCreditNote cannot be used as InitialType. |
| `errors.go` | Package-level ValidationIssue sentinels with HTTP status codes. Use these instead of fmt.Errorf for domain business-rule violations. | Generic fmt.Errorf returns for business violations produce 500 instead of 400 at the HTTP boundary. |
| `handler.go` | Handler interface for ledger callbacks (OnInvoiceUsageAccrued, OnPaymentAuthorized, OnPaymentSettled, OnCreditsOnlyUsageAccrued, OnCreditsOnlyUsageAccruedCorrection) and UnimplementedHandler base. | New Handler methods must be added to UnimplementedHandler or partial implementations will fail to compile. |
| `rating.go` | RateableIntent wraps Intent for rating.Service consumption. IsProgressivelyBilled always returns false; GetPreviouslyBilledAmount always returns zero — charges are never progressively billed. | Adding progressive-billing logic here breaks the rating invariant; charges do not support progressive billing. |

## Anti-Patterns

- Adding Ent or persistence imports to this package — it is a pure domain contract package; all DB access belongs in usagebased/adapter.
- Using fmt.Errorf for domain business-rule violations that have a ValidationIssue sentinel in errors.go — callers rely on error type assertions for HTTP status mapping.
- Setting UpdateRealizationRunInput fields directly without mo.Some() — absent options are treated as no-op updates by the adapter.
- Changing GetFeatureKeyOrID status dispatch without updating usagebased/service feature resolution to match — feature lookups will silently use the wrong ref type.
- Adding a method directly to Adapter instead of the appropriate fine-grained sub-interface (ChargeAdapter, RealizationRunAdapter, etc.) — breaks callers that embed only the narrower interface.

## Decisions

- **Dot-separated Status sub-states (e.g. active.partial_invoice.started) instead of a flat enum.** — ToMetaChargeStatus extracts the canonical top-level status by splitting on the first dot, keeping the meta layer stable while the usagebased state graph evolves independently.
- **ValidationIssue sentinels with WithHTTPStatusCodeAttribute declared in errors.go.** — Domain errors carry their HTTP status so the commonhttp encoder produces correct 4xx responses without bespoke error-type switches in HTTP handler layers.
- **RealizationRunTypeInvalidDueToUnsupportedCreditNote is a voided billing history marker retained for audit, never used as InitialType.** — Audit trail requires the run to persist after the invoice line is removed; IsVoidedBillingHistory() excludes it from future rating and balance calculations without hard-deleting it.

## Example: Adding a new service-layer input type for a usage-based charge operation

```
import (
	"errors"
	"fmt"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SnapshotUsageInput struct {
	ChargeID meta.ChargeID
	AsOf     time.Time
}

func (i SnapshotUsageInput) Validate() error {
	var errs []error
	if err := i.ChargeID.Validate(); err != nil {
// ...
```

<!-- archie:ai-end -->
