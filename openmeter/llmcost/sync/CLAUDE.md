# sync

<!-- archie:ai-start -->

> External price synchronization pipeline: fetches raw prices from external sources (models.dev), normalizes provider/model IDs, deduplicates within-source collisions from normalization, and reconciles multi-source agreement into global llmcostprice rows via llmcost.Adapter.UpsertGlobalPrice.

## Patterns

**Fetcher interface per source** — Each external source implements the Fetcher interface (Source() PriceSource, Fetch(ctx) ([]SourcePrice, error)). New sources add a new file (fetcher_<source>.go) and register in DefaultFetchers. (`type modelsDevFetcher struct { client *http.Client }; func (f *modelsDevFetcher) Source() llmcost.PriceSource { return "models_dev" }`)
**Four-phase SyncJob.Run pipeline** — Run executes: (1) fetch+normalize, (2) deduplicateSourcePrices, (3) optional filter, (4) Reconcile. Each phase is independent and logged. Fetcher errors are non-fatal — the job continues with remaining sources. (`allPrices = deduplicateSourcePrices(allPrices); if j.filter != nil { ... }; return j.reconciler.Reconcile(ctx, allPrices)`)
**Deduplication before reconciliation** — deduplicateSourcePrices removes within-source (source, provider, modelID) collisions that arise after normalization (e.g., azure_ai and azure both normalize to azure). Without this step, provider normalization creates false multi-source agreement in the reconciler. (`key := sourceModelKey{Source: p.Source, Provider: string(p.Provider), ModelID: p.ModelID}; seen[key] = idx`)
**Reconciler requires minAgreement sources within priceTolerance** — Reconciler.Reconcile groups prices by (provider, modelID). A global price is only upserted if at least minAgreement sources agree within priceTolerance (default 2 sources, 1% tolerance). Agreeing prices are averaged. (`NewReconciler(repo, logger, DefaultMinSourceAgreement, DefaultPriceTolerance)`)
**Per-million token conversion in fetchers** — External APIs (models.dev) provide prices per million tokens. Fetchers divide by 1,000,000 using alpacadecimal before returning SourcePrice so all downstream code works in per-token units. (`InputPerToken: alpacadecimal.NewFromFloat(*model.Cost.Input).Div(perMillion)`)
**minAgreement capped at fetcher count** — NewSyncJob caps minAgreement at len(fetchers) so a single-source deployment still works without manual config override. (`if numFetchers := len(fetchers); numFetchers > 0 && minAgreement > numFetchers { minAgreement = numFetchers }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `sync.go` | SyncJob orchestrator: holds fetchers, normalizer, reconciler, optional filter. Run() is the only public entry point. | deduplicateSourcePrices must run AFTER normalization and BEFORE filter+reconcile — the order is load-bearing |
| `fetcher_modelsdev.go` | HTTP fetcher for models.dev/api.json; strips provider prefix from model IDs and names, converts per-million to per-token | Provider IDs are lowercased but not aliased here — aliasing (azure_ai→azure) happens in normalizer. The fetcher strips 'provider/model' ID prefix but keeps the raw provider key as-is for the normalizer to canonicalize. |
| `reconciler.go` | Groups SourcePrice by (provider, modelID), runs findAgreement pairwise comparison, averages agreeing prices, calls repo.UpsertGlobalPrice | findAgreement is O(n²) in number of sources per model — acceptable for current source count (<10) but not for large fan-out. averagePrices uses prices[0] as metadata base; model name comes from the first agreeing source. |
| `normalizer.go` | Wraps llmcost.NormalizeModelID — lowercases, trims whitespace, strips date version suffixes, and aliases provider names (gemini→google, mistralai→mistral, etc.) | Normalization is applied in SyncJob.Run AFTER fetching but BEFORE deduplication. Do not call Normalize inside fetchers. |
| `fetcher.go` | Fetcher interface definition only | Source() must return a stable PriceSource constant — it is used as the key in SourcePricesMap stored in the DB |

## Anti-Patterns

- Adding normalization logic inside fetchers — normalization belongs in the normalizer so all fetchers are handled uniformly
- Calling reconciler.Reconcile before deduplicateSourcePrices — allows false multi-source agreement from provider aliasing
- Making fetcher errors fatal — a single down source should not abort the sync for all models
- Storing per-million prices in SourcePrice — always convert to per-token before returning from Fetch()

## Decisions

- **Separate Fetcher, Normalizer, Reconciler into distinct types** — Allows testing each phase independently with mocks; new sources only require a new Fetcher implementation without touching reconciliation logic
- **Multi-source agreement threshold before writing global prices** — A single external source could be wrong or stale; requiring agreement from multiple sources prevents billing using bad data from a single upstream outage or API change

## Example: Add a new price source (e.g., litellm)

```
// fetcher_litellm.go
package sync

import (
	"context"
	"net/http"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
)

type litellmFetcher struct{ client *http.Client }

func NewLiteLLMFetcher(client *http.Client) Fetcher {
	return &litellmFetcher{client: client}
}

// ...
```

<!-- archie:ai-end -->
