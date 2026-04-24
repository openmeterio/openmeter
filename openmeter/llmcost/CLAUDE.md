# llmcost

<!-- archie:ai-start -->

> LLM cost price management domain: persists global (synced) prices and per-namespace manual overrides in llmcostprice, resolves effective prices with namespace-override precedence, and synchronises prices from external sources (models.dev). All monetary values use alpacadecimal.Decimal for precision.

## Patterns

**Validate() on every input and domain type** — All input types implement models.Validator (compile-time asserted with `var _ models.Validator = (*XInput)(nil)`). Call input.Validate() at the service boundary before delegating to the adapter. (`var _ models.Validator = (*ListPricesInput)(nil)
func (i ListPricesInput) Validate() error { ... return models.NewNillableGenericValidationError(errors.Join(errs...)) }`)
**models.NewNillableGenericValidationError wrapping** — All Validate() methods return models.NewNillableGenericValidationError(errors.Join(errs...)) — returns nil when errs is empty, ensuring consistent error typing for HTTP encoding. (`return models.NewNillableGenericValidationError(errors.Join(errs...))`)
**ValidationIssue sentinel errors for field-level errors** — Each error condition has a named sentinel models.NewValidationIssue with ErrCodeXxx constant, field path, severity, and HTTP status attribute. New error conditions follow this pattern in errors.go. (`var ErrProviderEmpty = models.NewValidationIssue(ErrCodeProviderEmpty, "provider must not be empty", models.WithFieldString("provider"), models.WithCriticalSeverity(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest))`)
**PriceSource discrimination for global vs override rows** — Rows with namespace IS NULL and source='system' are global prices; rows with namespace IS NOT NULL and source='manual' are overrides. Never mix these in a single query path. (`PriceSourceManual PriceSource = "manual"; PriceSourceSystem PriceSource = "system"`)
**alpacadecimal.Decimal for all price fields** — All cost-per-token fields use alpacadecimal.Decimal (never float64 or string). Optional token dimensions (CacheRead, CacheWrite, Reasoning) are *alpacadecimal.Decimal. (`InputPerToken alpacadecimal.Decimal; CacheReadPerToken *alpacadecimal.Decimal`)
**NormalizeModelID before any price lookup or insert** — Call llmcost.NormalizeModelID(provider, modelID) before storing or resolving prices to canonicalise casing, version suffixes, region prefixes, and provider aliases. (`canonicalProvider, canonicalModelID := llmcost.NormalizeModelID(provider, modelID)`)
**TransactingRepo wrapping in adapter** — Every adapter method that writes must be wrapped with entutils.TransactingRepo / TransactingRepoWithNoValue so ctx-carried transactions are honored. (`return entutils.TransactingRepo(ctx, a.db, func(tx *entdb.Tx) (Price, error) { ... })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/llmcost/llmcost.go` | Core domain types: Price, ModelPricing, PriceSource, SourcePrice, SourcePricesMap. All fields validated in Validate() methods. | Adding new token-cost dimensions requires updating ModelPricing, Validate(), and all mapping/adapter code. |
| `openmeter/llmcost/service.go` | Service interface definition plus all input types with their Validate() methods. Compile-time Validator assertions guard every input type. | New Service methods require a matching XInput type with Validate() and a compile-time assertion here. |
| `openmeter/llmcost/adapter.go` | Adapter interface declaration (extends entutils.TxCreator). Source of truth for which persistence operations exist. | Any new query or write must be added to Adapter before implementing in openmeter/llmcost/adapter/. |
| `openmeter/llmcost/errors.go` | All ValidationIssue sentinels and error constructors for the llmcost domain. | New field-level errors must follow the ErrCodeXxx constant + models.NewValidationIssue pattern with explicit HTTP status. |
| `openmeter/llmcost/normalize.go` | NormalizeModelID and NormalizeProvider: strips version/region suffixes, normalises provider aliases. Must be called before any price store or resolve. | Adding new provider aliases or version-suffix patterns requires updating this file and its test table. |
| `openmeter/llmcost/adapter/adapter.go` | Ent/PostgreSQL adapter; all writes use TransactingRepo; ResolvePrice uses ORDER BY namespace DESC to prefer overrides. | Never call a.db directly outside TransactingRepo in write paths — falls off ctx transaction. |
| `openmeter/llmcost/service/service.go` | Business-logic layer; applies namespace-override overlay in ListPrices at service layer, not adapter layer; wraps mutations in transaction.Run. | Do not add DB queries inside the overlay loop — batch-fetch overrides once before iterating. |
| `openmeter/llmcost/sync/sync.go` | Four-phase sync pipeline: Fetch → Normalize → Deduplicate → Reconcile. Writes via Adapter.UpsertGlobalPrice only after multi-source agreement. | Fetcher errors must not abort the sync for all models; reconciler.Reconcile must follow deduplicateSourcePrices. |

## Anti-Patterns

- Using float64 or string for price fields — must use alpacadecimal.Decimal to preserve billing precision.
- Skipping NormalizeModelID before storing or resolving prices — causes provider/model ID mismatches across sources.
- Calling a.db directly in adapter helpers without TransactingRepo — bypasses ctx-carried Ent transaction.
- Adding source='manual' rows via UpsertGlobalPrice (reserved for source='system' reconciled prices).
- Calling reconciler.Reconcile before deduplicateSourcePrices — allows false multi-source agreement from provider aliasing.

## Decisions

- **Namespace-override overlay applied at service layer, not adapter layer** — Keeps the adapter queries simple (global vs override are separate query paths); the service batches both and merges them in memory.
- **Multi-source agreement threshold before writing global prices** — A single down source should not pollute global prices; requiring minAgreement sources within priceTolerance prevents rogue data from one fetcher.
- **Normalisation as a pure package-level function (NormalizeModelID)** — Applied uniformly by both the sync pipeline and the adapter/service, eliminating per-caller inconsistency in canonical key construction.

## Example: Adding a new service method with input validation

```
// In service.go — declare input type with compile-time Validator assertion
var _ models.Validator = (*GetLatestPriceInput)(nil)

type GetLatestPriceInput struct {
    Namespace string
    Provider  Provider
    ModelID   string
}

func (i GetLatestPriceInput) Validate() error {
    var errs []error
    if i.Provider == "" { errs = append(errs, ErrProviderEmpty) }
    if i.ModelID == "" { errs = append(errs, ErrModelIDEmpty) }
    return models.NewNillableGenericValidationError(errors.Join(errs...))
}
// ...
```

<!-- archie:ai-end -->
