# llmcost

<!-- archie:ai-start -->

> LLM cost-price domain: persists global (synced, namespace IS NULL, source='system') prices and per-namespace manual overrides (source='manual') in the llmcostprice table, resolves effective prices with namespace-override precedence, and syncs prices from external sources (models.dev). All monetary values use alpacadecimal.Decimal.

## Patterns

**Validate() + Nillable wrapping on every input** — Each input type asserts `var _ models.Validator = (*XInput)(nil)` and its Validate() returns models.NewNillableGenericValidationError(errors.Join(errs...)); call input.Validate() at the service boundary. (`return models.NewNillableGenericValidationError(errors.Join(errs...))`)
**ValidationIssue sentinels for field errors** — Each field-level error is a named sentinel in errors.go combining an ErrCodeXxx, field path, severity, and HTTP-status attribute. (`var ErrProviderEmpty = models.NewValidationIssue(ErrCodeProviderEmpty, "provider must not be empty", models.WithFieldString("provider"), models.WithCriticalSeverity(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest))`)
**PriceSource discrimination (system vs manual)** — Global rows are namespace IS NULL + source='system'; overrides are namespace IS NOT NULL + source='manual'. Never mix these in one query path; UpsertGlobalPrice is reserved for system rows. (`PriceSourceManual PriceSource = "manual"; PriceSourceSystem PriceSource = "system"`)
**alpacadecimal.Decimal for all price fields** — Cost-per-token fields use alpacadecimal.Decimal (never float64/string); optional dimensions (CacheRead/CacheWrite/Reasoning) are *alpacadecimal.Decimal. (`InputPerToken alpacadecimal.Decimal; CacheReadPerToken *alpacadecimal.Decimal`)
**NormalizeModelID before lookup/insert** — Call llmcost.NormalizeModelID(provider, modelID) before storing or resolving to canonicalise casing, version/region suffixes, and provider aliases uniformly. (`canonicalProvider, canonicalModelID := llmcost.NormalizeModelID(provider, modelID)`)
**Override overlay at service layer; sync requires multi-source agreement** — Adapter keeps global/override as separate query paths; the service batch-fetches overrides once and overlays them in memory. The sync pipeline (Fetch→Normalize→Deduplicate→Reconcile) writes globals only after minAgreement sources agree within tolerance. (`// service overlays overrides; sync calls Adapter.UpsertGlobalPrice after reconciler.Reconcile`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `llmcost.go` | Core domain types: Price, ModelPricing, PriceSource, SourcePrice, SourcePricesMap with Validate(). | Adding a token-cost dimension requires updating ModelPricing, Validate(), and all mapping/adapter code. |
| `service.go` | Service interface plus input types with Validate() and compile-time Validator assertions. | Every new Service method needs a matching XInput with Validate() and an assertion. |
| `adapter.go` | Adapter interface (extends entutils.TxCreator) — source of truth for persistence operations. | New queries/writes must be declared here before implementing in adapter/. |
| `errors.go` | All ValidationIssue sentinels and error constructors for the domain. | New field errors follow the ErrCodeXxx + NewValidationIssue + explicit HTTP status pattern. |
| `normalize.go` | NormalizeModelID/NormalizeProvider: strip version/region suffixes, normalise aliases. Must precede any store/resolve. | New aliases or suffix patterns require updating this file and its test table. |
| `adapter/adapter.go` | Ent adapter; all writes wrapped in TransactingRepo; ResolvePrice uses ORDER BY namespace DESC to prefer overrides; soft-delete via SetDeletedAt(clock.Now()). | Never call a.db directly in a write helper outside TransactingRepo; never hard-delete. |
| `sync/sync.go` | Four-phase pipeline writing globals only after multi-source agreement. | Fetcher errors must be non-fatal; reconciler.Reconcile must run after deduplicateSourcePrices. |

## Anti-Patterns

- Using float64/string for price fields instead of alpacadecimal.Decimal.
- Skipping NormalizeModelID before storing/resolving prices.
- Calling a.db directly in adapter write helpers without TransactingRepo.
- Adding source='manual' rows via UpsertGlobalPrice (reserved for reconciled system prices).
- Calling reconciler.Reconcile before deduplicateSourcePrices — allows false multi-source agreement from aliasing.

## Decisions

- **Namespace-override overlay applied at the service layer, not the adapter.** — Keeps adapter queries simple (separate global/override paths); the service batches and merges in memory.
- **Require multi-source agreement before writing global prices.** — A single down or rogue source must not pollute globals; minAgreement within priceTolerance guards this.
- **Normalisation is a pure package-level function (NormalizeModelID).** — Applied uniformly by both sync and adapter/service, eliminating per-caller key-construction drift.

## Example: Adding a new service method with input validation

```
// In service.go
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
```

<!-- archie:ai-end -->
