# usagebased

<!-- archie:ai-start -->

> Public domain package for usage-based charges: declares domain types (Charge, ChargeBase, Intent, State, Status, RealizationRun, RealizationRuns, DetailedLines, Expands), the Adapter interface, the Service interface, the Handler interface, rating helpers (RateableIntent), and structured error sentinels. It is the contract boundary consumed by usagebased/service and usagebased/adapter.

## Patterns

**Input.Validate() gates all cross-boundary calls** — Every Input struct — CreateInput, AdvanceChargeInput, GetByIDsInput, CreateRealizationRunInput, UpdateRealizationRunInput — implements Validate() collecting errors via errors.Join and wrapping with models.NewNillableGenericValidationError. (`if err := input.Validate(); err != nil { return nil, err }`)
**Adapter composed of fine-grained sub-interfaces + entutils.TxCreator** — Adapter embeds RealizationRunAdapter, RealizationRunCreditAllocationAdapter, RealizationRunInvoiceUsageAdapter, RealizationRunPaymentAdapter, ChargeAdapter, and entutils.TxCreator. (`type Adapter interface { RealizationRunAdapter; ChargeAdapter; ...; entutils.TxCreator }`)
**Hierarchical status with dot-separated sub-states** — Status constants use dot notation for sub-states (e.g. StatusActivePartialInvoiceStarted = "active.partial_invoice.started"); ToMetaChargeStatus splits on the first dot to extract the canonical meta.ChargeStatus. (`split := strings.SplitN(string(s), ".", 2); metaStatus := meta.ChargeStatus(split[0])`)
**Expand dependency validation** — validateExpands (local, unexported) enforces that ExpandDetailedLines requires ExpandRealizations; unit-tested in service_test.go. (`require.Error(t, validateExpands(meta.Expands{meta.ExpandDetailedLines}))`)
**RateableIntent implements rating.StandardLineAccessor** — RateableIntent wraps Intent + MeterValue + CreditsApplied and satisfies rating.StandardLineAccessor; IsProgressivelyBilled and GetPreviouslyBilledAmount always return false/zero for charges. (`var _ rating.StandardLineAccessor = (*RateableIntent)(nil)`)
**Structured ValidationIssue errors with HTTP status codes** — errors.go declares domain-specific errors as models.ValidationIssue sentinels (ErrChargeTotalIsNegative, ErrCreditAllocationsDoNotMatchTotal, ErrActiveRealizationRunAlreadyExists) with commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest). (`var ErrChargeTotalIsNegative = models.NewValidationIssue(ErrCodeChargeTotalIsNegative, "charge total is negative", models.WithCriticalSeverity(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest))`)
**UnimplementedHandler for compile-time interface coverage** — UnimplementedHandler implements Handler returning errors.New("not implemented") for every method; used as embedded base in partial handler implementations. (`var _ Handler = (*UnimplementedHandler)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Declares Adapter interface and its five sub-interfaces. New DB operations for realization runs, credit allocations, invoiced usage, or payments must be added to the matching sub-interface. | Adding a method directly to Adapter instead of the appropriate sub-interface breaks callers that embed only the narrower interface. |
| `charge.go` | Core domain types: ChargeBase, Charge, Intent, State, Expands, Charges. GetFeatureKeyOrID has status-aware logic (created→key, deleted prefers ID, others use ID). | GetFeatureKeyOrID status dispatch must stay in sync with usagebased/service feature resolution; incorrect logic causes feature lookup misses. |
| `service.go` | Declares Service (= UsageBasedService + GetLineEngine), all service-layer Input/Output types, and validateExpands. | AdvanceChargeInput requires both expanded Customer and MergedProfile — constructors that omit them will panic at runtime inside the state machine. |
| `handler.go` | Handler interface for ledger callbacks; UnimplementedHandler provides a safe base for partial implementations. | New Handler methods must be added to UnimplementedHandler or embedded structs will fail to compile. |
| `realizationrun.go` | RealizationRun, RealizationRunBase, RealizationRuns, CreateRealizationRunInput, UpdateRealizationRunInput. RealizationRuns.Sum() aggregates totals across all runs. | UpdateRealizationRunInput uses mo.Option fields — callers must use mo.Some(...) not direct assignment; absent options are silently ignored by the adapter. |
| `statemachine.go` | Status type, all status constants including dot-separated sub-states, Values(), Validate(), ToMetaChargeStatus(). | New sub-states must be added to Values() and the dot-split logic in ToMetaChargeStatus must remain correct. |
| `errors.go` | Package-level ValidationIssue sentinels with HTTP status codes. Use these rather than fmt.Errorf for domain-level business rule violations. | Generic fmt.Errorf returns instead of these sentinels will produce 500s instead of 400s at the HTTP boundary. |
| `rating.go` | RateableIntent wraps Intent for rating.Service consumption. IsProgressivelyBilled always returns false; GetPreviouslyBilledAmount always returns zero. | Any progressive-billing logic added here would break the rating invariant that charges are never progressively billed. |

## Anti-Patterns

- Using fmt.Errorf for business rule violations that have a ValidationIssue sentinel — callers rely on error type assertions for HTTP status mapping.
- Accessing RealizationRun.DetailedLines.OrEmpty() without checking IsPresent() first — returns a zero DetailedLines slice silently.
- Setting UpdateRealizationRunInput fields without mo.Some() — absent options are treated as no-op updates by the adapter.
- Changing GetFeatureKeyOrID without updating usagebased/service feature resolution to match — feature lookups will silently use the wrong ref type.
- Adding Ent or persistence imports to this package — it is a pure domain contract package; all DB access belongs in usagebased/adapter.

## Decisions

- **Dot-separated Status sub-states (e.g. active.partial_invoice.started) instead of a flat enum.** — Sub-states allow ToMetaChargeStatus to extract the canonical top-level status by splitting on the first dot, keeping the meta layer stable while the usagebased state graph evolves independently.
- **ValidationIssue sentinels with WithHTTPStatusCodeAttribute declared in errors.go.** — Domain errors carry their HTTP status so the commonhttp encoder can produce correct 4xx responses without bespoke error-type switches in the HTTP handler layer.
- **RateableIntent implements rating.StandardLineAccessor rather than converting Intent at the service layer.** — Wrapping Intent in a thin accessor keeps the rating.Service dependency out of the service constructor and makes the accessor testable in isolation.

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
