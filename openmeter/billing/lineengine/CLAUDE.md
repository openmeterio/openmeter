# lineengine

<!-- archie:ai-start -->

> Implements the billing.LineEngine interface for the standard invoice line type — handles quantity snapshotting, split-line-group creation, line period truncation, and per-line rating via rating.Service; registered as LineEngineTypeInvoice in the billing.Service engine registry.

## Patterns

**Config struct with Validate() + New()** — Engine is constructed via Config.Validate() + New(Config), returning (*Engine, error). All dependencies injected through Config. (`func New(config Config) (*Engine, error) { if err := config.Validate(); err != nil { return nil, err } return &Engine{...}, nil }`)
**SplitLineGroupAdapter interface for DB isolation** — splitlinegroup.go defines SplitLineGroupAdapter with only CreateSplitLineGroup and GetSplitLineGroupHeaders; Engine never imports the full billing.Adapter. (`type SplitLineGroupAdapter interface { CreateSplitLineGroup(...) (billing.SplitLineGroup, error); GetSplitLineGroupHeaders(...) (billing.SplitLineGroupHeaders, error) }`)
**QuantitySnapshotter interface for dependency inversion** — stdinvoice.go defines QuantitySnapshotter with SnapshotLineQuantities; concrete implementation is billingservice.Service. Engine never calls the service directly. (`type QuantitySnapshotter interface { SnapshotLineQuantities(ctx context.Context, invoice billing.StandardInvoice, lines billing.StandardLines) error }`)
**billing.ValidationIssue for snapshot errors, not plain errors** — When SnapshotLineQuantities returns an ErrSnapshotInvalidDatabaseState, OnCollectionCompleted converts it to billing.ValidationIssue with ValidationIssueSeverityCritical. (`if _, isInvalidDatabaseState := lo.ErrorsAs[*billing.ErrSnapshotInvalidDatabaseState](err); isInvalidDatabaseState { return nil, billing.ValidationIssue{Severity: billing.ValidationIssueSeverityCritical, ...} }`)
**isPeriodEmptyConsideringTruncations for period validity** — Usage-based lines check if their period is empty after truncation to streaming.MinimumWindowSizeDuration; flat-fee lines always return false. (`return line.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration).IsEmpty(), nil`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/billing/lineengine/engine.go` | Implements billing.LineEngine interface: GetLineEngineType, OnCollectionCompleted, OnStandardInvoiceCreated, OnInvoiceIssued, OnPaymentAuthorized, OnPaymentSettled. OnCollectionCompleted is the primary hook for quantity snapshotting. | OnCollectionCompleted short-circuits if QuantitySnapshotedAt is already set and at or after the default collection time — do not add snapshot logic after this guard. |
| `openmeter/billing/lineengine/splitlinegroup.go` | SplitGatheringLine splits a gathering line at a point in time, creates a SplitLineGroup if needed, and produces pre/post split line pairs. ResolveSplitLineGroupHeaders batch-fetches group metadata for standard lines. | ChildUniqueReferenceID is set to nil on split lines — the SplitLineGroup owns this reference; never copy it from the source line. |
| `openmeter/billing/lineengine/stdinvoice.go` | BuildStandardInvoiceLines converts gathering lines to standard lines, snapshots quantities, and runs CalculateLines. CalculateLines calls ratingService.GenerateDetailedLines per line. | CalculateLines requires lines to be non-empty — guard callers; invoicecalc.MergeGeneratedDetailedLines is called after each GenerateDetailedLines call. |

## Anti-Patterns

- Importing billing.Adapter or openmeter/ent/db directly — use SplitLineGroupAdapter interface
- Calling SnapshotLineQuantities from outside OnCollectionCompleted/BuildStandardInvoiceLines — quantity snapshotting is lifecycle-gated
- Skipping ResolveSplitLineGroupHeaders before CalculateLines on lines that may have split groups — price calculations depend on SplitLineHierarchy being populated
- Returning plain errors from snapshot paths that should be billing.ValidationIssue — callers use type assertions on ValidationIssue
- Copying ChildUniqueReferenceID from source gathering line to split child lines

## Decisions

- **Engine registered at Service construction time via Service.RegisterLineEngine, not hardcoded** — Allows charges sub-packages to register their own engines (flatfee, usagebased, creditpurchase) without modifying the base service.
- **SplitLineGroupAdapter and QuantitySnapshotter are separate narrow interfaces rather than the full billing.Service** — Prevents circular imports; lineengine is imported by billingservice which is the QuantitySnapshotter implementation.

<!-- archie:ai-end -->
