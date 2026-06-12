# sync

<!-- archie:ai-start -->

> Background sync pipeline that fetches LLM model prices from external sources (e.g. models.dev), normalizes provider/model IDs, deduplicates, and reconciles multi-source agreement into canonical system global prices via llmcost.Adapter.UpsertGlobalPrice. Run by app/common and cmd/jobs.

## Patterns

**Fetcher interface per source** — Each external source implements Fetcher{ Source() llmcost.PriceSource; Fetch(ctx) ([]llmcost.SourcePrice, error) }. NewModelsDevFetcher is the built-in; DefaultFetchers(client) returns the registered list. (`type Fetcher interface { Source() llmcost.PriceSource; Fetch(ctx context.Context) ([]llmcost.SourcePrice, error) }`)
**SyncJob orchestration phases** — SyncJob.Run executes fetch+normalize -> deduplicate -> optional filter -> reconcile. A failing fetcher logs and is skipped (continue), never failing the whole job. (`continue // Don't fail entire sync if one source is down`)
**Normalizer canonicalizes provider+model** — ModelIDNormalizer.Normalize delegates to llmcost.NormalizeModelID, lowercasing, trimming, mapping provider aliases (gemini->google, aws->amazon), and stripping date version suffixes (gpt-4o-2024-08-06 -> gpt-4o). (`func (n *defaultNormalizer) Normalize(modelID, provider string) (string, string) { return llmcost.NormalizeModelID(provider, modelID) }`)
**Per-source deduplication before reconcile** — deduplicateSourcePrices collapses duplicate (source, provider, model_id) keys that normalization created, preferring model names without a '/' provider prefix and using lexicographic tie-break, to prevent false multi-source agreement. (`if existingHasPrefix && !newHasPrefix || existingHasPrefix == newHasPrefix && p.ModelName < existing.ModelName { result[idx] = p }`)
**Tolerance-based multi-source agreement** — Reconciler groups by (provider, model_id), requires >= minAgreement sources agreeing within priceTolerance (pricesAgree on all token dimensions incl. optional ones), averages the agreeing prices, and upserts a PriceSourceSystem global price with a SourcePricesMap. (`agreeing := r.findAgreement(sourcePrices); if agreeing == nil { skipped++; continue }`)
**Optional decimal agreement semantics** — optionalDecimalsAgree: both nil agree, exactly one nil disagrees, both set delegate to decimalsAgree (zero-vs-nonzero disagree, else within ratio tolerance). averageOptionalDecimal returns nil if the first price's field is unset. (`if a == nil && b == nil { return true }; if a == nil || b == nil { return false }`)
**Config caps minAgreement at fetcher count** — NewSyncJob defaults MinSourceAgreement<=0 to DefaultMinSourceAgreement(2) but caps it at len(fetchers) so a single-fetcher setup can still reconcile. (`if numFetchers := len(fetchers); numFetchers > 0 && minAgreement > numFetchers { minAgreement = numFetchers }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `sync.go` | SyncJob, SyncJobConfig, NewSyncJob, DefaultFetchers, deduplicateSourcePrices, and the Run pipeline. | Filter reuses allPrices[:0] in place; dedup must run BEFORE reconcile or provider-alias collapse creates false agreement. minAgreement capping logic is load-bearing for single-source setups. |
| `fetcher_modelsdev.go` | modelsDevFetcher fetching https://models.dev/api.json, converting per-million prices to per-token (Div(perMillion)) and stripping provider prefixes from model IDs/names. | Skips models lacking cost.Input or cost.Output. Provider key is lowercased; empty provider entries are skipped. Optional cache/reasoning fields are pointer-set only when present. |
| `reconciler.go` | Reconciler, NewReconciler (clamps minAgreement<=0 and priceTolerance<0 to defaults), Reconcile, findAgreement, pricesAgree, averagePrices and decimal helpers. | DefaultMinSourceAgreement=2, DefaultPriceTolerance=0.01. decimalsAgree treats zero-vs-nonzero as disagreement; ratio is diff/max. Upsert failures are collected into errors.Join, not returned eagerly. |
| `normalizer.go` | ModelIDNormalizer interface, defaultNormalizer wrapping llmcost.NormalizeModelID. | Generic transforms only; source-specific cleanup belongs in each Fetcher. Provider alias and version-suffix logic lives in llmcost.NormalizeModelID, not here. |
| `fetcher.go` | Fetcher interface definition. | Source() must return a stable PriceSource string used as the SourcePricesMap key in reconciliation. |

## Anti-Patterns

- Failing the whole SyncJob.Run when a single Fetcher errors instead of logging and continuing.
- Reconciling before deduplicateSourcePrices runs, letting provider-alias collapse fake multi-source agreement.
- Requiring more agreeing sources than fetchers exist (must cap minAgreement at len(fetchers)).
- Doing source-specific cleanup inside the normalizer instead of in the Fetcher before returning prices.
- Treating a missing optional pricing field as zero in agreement checks (one nil must disagree with a set value).

## Decisions

- **Canonical global prices require multi-source agreement within tolerance, then average.** — Single-source external data is unreliable; requiring >=2 sources within 1% and averaging produces a defensible PriceSourceSystem price with a full SourcePricesMap audit trail.
- **Deduplicate per source after normalization, before reconciliation.** — Provider aliases (azure_ai and azure) collapse to one key; without dedup a single source would appear as two agreeing sources.
- **Prices are fetched per-million then divided to per-token at the fetcher boundary.** — Keeps the internal llmcost.ModelPricing in a single per-token unit so the reconciler and adapter never deal with source-specific scaling.

## Example: Full sync cycle: fetch (skip on error) -> normalize -> deduplicate -> filter -> reconcile

```
func (j *SyncJob) Run(ctx context.Context) error {
	var allPrices []llmcost.SourcePrice
	for _, f := range j.fetchers {
		prices, err := f.Fetch(ctx)
		if err != nil {
			j.logger.Error("failed to fetch prices", "source", f.Source(), "error", err)
			continue
		}
		for _, p := range prices {
			provider, modelID := j.normalizer.Normalize(p.ModelID, string(p.Provider))
			p.Provider = llmcost.Provider(provider)
			p.ModelID = modelID
			allPrices = append(allPrices, p)
		}
	}
// ...
```

<!-- archie:ai-end -->
