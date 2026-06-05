# persistedstate

<!-- archie:ai-start -->

> Loads the current persisted billing state for a subscription (invoice lines, split-line hierarchies, and charges) and wraps each as a uniform Item keyed by ChildUniqueReferenceID, so the subscription-sync reconciler can diff persisted state against target state. Primary constraint: every persisted entity is normalized to meter (MinimumWindowSizeDuration) precision on read.

## Patterns

**Uniform Item interface over heterogeneous backing types** — Invoice lines, split-line hierarchies, usage-based charges, and flat-fee charges all implement the Item interface (ID, Type, ChildUniqueReferenceID, ServicePeriod, IsSubscriptionManaged, HasLastLineAnnotation). New persisted-state kinds must add an ItemType constant and a private struct implementing Item, asserted via a `var _ Item = ...` block. (`var (\n\t_ Item       = persistedLine{}\n\t_ LineGetter = persistedLine{}\n)`)
**Private constructors validate, public Item accessors never error** — Construction funcs (newPersistedLine, newPersistedUsageBasedCharge, etc.) are private and perform nil/Validate() checks so the Item methods can expose non-erroring accessors. Never expose a constructor publicly or skip the validation it performs. (`func newPersistedUsageBasedCharge(charge usagebased.Charge) (persistedUsageBasedCharge, error) {\n\tif err := charge.Validate(); err != nil { return persistedUsageBasedCharge{}, fmt.Errorf("usage based charge is invalid: %w", err) }\n\treturn persistedUsageBasedCharge{charge: charge}, nil\n}`)
**Typed downcast getters via ItemAs* helpers** — To recover a backing type from an Item, use the ItemAs* helpers (ItemAsLine, ItemAsSplitLineHierarchy, ItemAsUsageBasedCharge, ItemAsFlatFeeCharge), which type-assert against the matching *Getter interface and return a descriptive error from getErrorDetails(in). Never type-assert against the private structs directly. (`lineGetter, ok := in.(LineGetter)\nif !ok { return nil, fmt.Errorf("persisted item does not implement line getter: %s", getErrorDetails(in)) }`)
**Read-time normalization to meter precision** — normalizePersistedLineOrHierarchy and normalizeSubscriptionReference Truncate service periods, invoice-at, and subscription BillingPeriod to streaming.MinimumWindowSizeDuration on load, so reconciliation does not propose no-op repairs against legacy sub-second timestamps. Any new persisted field carrying a time MUST be normalized here. (`cloned.UpdateServicePeriod(func(period *timeutil.ClosedPeriod) {\n\t*period = period.Truncate(streaming.MinimumWindowSizeDuration)\n})`)
**Charge-managed lines loaded as charges, not lines** — LoadForSubscription requests lines with IncludeChargeManaged: false and separately calls chargeService.ListCharges, because charge-managed invoice lines are edited via charge patches. The two maps are merged with a duplicate-unique-id guard across lines and charges. (`GetLinesForSubscriptionInput{... IncludeChargeManaged: false}`)
**State keyed by ChildUniqueReferenceID with global uniqueness** — State.ByUniqueID maps the deterministic ChildUniqueReferenceID to its Item; items lacking a unique id are skipped, and duplicates (within lines, within charges, or across the two) are hard errors. State.Validate requires both ByUniqueID and Invoices to be non-nil. (`if _, ok := byUniqueID[*uniqueID]; ok { return State{}, fmt.Errorf("duplicate unique ids in the existing lines") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `item.go` | Defines ItemType constants, the Item interface plus per-kind getter interfaces (LineGetter, SplitLineHierarchyGetter, UsageBasedChargeGetter, FlatFeeChargeGetter), the four private backing structs, their private constructors, the ItemAs* downcasters, and the NewItemFromLineOrHierarchy / NewChargeItemFromChargeType factories. | SplitLineHierarchy delegates IsSubscriptionManaged/HasLastLineAnnotation to getLastLineForAnnotations (the child whose service-period end matches the group end and is not deleted); charge items read annotations directly off Intent.Annotations, not a child line. |
| `loader.go` | Loader struct over billingService (GetLinesForSubscription, ListInvoices) and chargeService (ListCharges) interfaces; LoadForSubscription assembles State, loadChargesForSubscription/loadInvoicesForSubscriptionLines fan out, normalizePersistedLineOrHierarchy normalizes precision on read. | chargeService may be nil (returns empty map). Credit-purchase charges tied to a subscription are an explicit hard error. Invoices are loaded with IncludeDeleted: true and every referenced invoice id must resolve or it errors. |
| `state.go` | State struct (ByUniqueID map[string]Item + Invoices) and Invoices map type with IsGatheringInvoice helper and Validate. | Invoices.IsGatheringInvoice errors when the invoice id is absent from state rather than returning false. |

## Anti-Patterns

- Exposing a public constructor for an Item backing type or skipping the nil/Validate() check it performs
- Type-asserting an Item directly to a private struct instead of using the ItemAs* getter helpers
- Reading a persisted time field without Truncate(streaming.MinimumWindowSizeDuration) normalization, causing endless no-op reconciliation
- Loading charge-managed lines as plain lines instead of via ListCharges (set IncludeChargeManaged: false)
- Reusing a ChildUniqueReferenceID across lines and charges (collisions are hard errors by design)

## Decisions

- **Wrap lines, hierarchies, and charges behind one Item interface keyed by unique reference id** — The reconciler diffs target vs persisted state purely by unique id and a small set of accessors, so it must not care whether the backing object is a line, a split-line group, or a charge.
- **Normalize timestamps to meter precision on read with a TODO to migrate the data** — Legacy rows persist ns precision the meter engine cannot query; normalizing on read prevents sync from perpetually proposing no-op timestamp repairs until a backfill migration lands.

<!-- archie:ai-end -->
