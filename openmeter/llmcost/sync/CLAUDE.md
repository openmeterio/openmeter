# sync

<!-- archie:ai-start -->

> External price synchronization pipeline: fetches raw LLM model prices from external sources (models.dev), normalizes provider and model IDs, deduplicates within-source collisions introduced by normalization, then reconciles multi-source agreement into global llmcostprice rows via llmcost.Adapter.UpsertGlobalPrice. Used by cmd/jobs llm-cost sync command.

## Patterns

**Fetcher interface per source in dedicated file** — Each external source implements Fetcher (Source() PriceSource, Fetch(ctx) ([]SourcePrice, error)). New sources add fetcher_<source>.go and register in DefaultFetchers. Source() must return a stable PriceSource constant — used as DB key. (`type modelsDevFetcher struct{ client *http.Client }; func (f *modelsDevFetcher) Source() llmcost.PriceSource { return "models_dev" }`)
**Four-phase SyncJob.Run pipeline with strict ordering** — Phase order is mandatory: (1) fetch+normalize per source, (2) deduplicateSourcePrices (must run after normalize, before reconcile), (3) optional filter, (4) Reconcile. Do not reorder phases. (`allPrices = deduplicateSourcePrices(allPrices); if j.filter != nil { ... }; return j.reconciler.Reconcile(ctx, allPrices)`)
**Per-million to per-token conversion in fetchers** — External APIs provide prices per million tokens. Fetchers must divide by 1,000,000 using alpacadecimal before returning SourcePrice so all downstream code works in per-token units. (`perMillion := alpacadecimal.NewFromFloat(1_000_000); InputPerToken: alpacadecimal.NewFromFloat(*model.Cost.Input).Div(perMillion)`)
**deduplicateSourcePrices prevents false multi-source agreement** — Provider normalization (azure_ai → azure) can collapse two raw entries from the same source into the same (source, provider, modelID) key. deduplicateSourcePrices removes within-source duplicates after normalization but before reconciliation. (`key := sourceModelKey{Source: p.Source, Provider: string(p.Provider), ModelID: p.ModelID}; if idx, exists := seen[key]; exists { ... keep preferred }`)
**minAgreement capped at fetcher count** — NewSyncJob caps minAgreement at len(fetchers) so a single-source deployment works without requiring manual config override. (`if numFetchers := len(fetchers); numFetchers > 0 && minAgreement > numFetchers { minAgreement = numFetchers }`)
**Non-fatal fetcher errors — continue with remaining sources** — If a fetcher returns an error, log it and continue. A single down source must not abort the sync for all models. (`prices, err := f.Fetch(ctx); if err != nil { j.logger.Error(...); continue }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `sync.go` | SyncJob orchestrator: holds fetchers, normalizer, reconciler, optional filter. Run() is the only public entry point. | deduplicateSourcePrices must run AFTER normalization and BEFORE filter+reconcile — the order is load-bearing to prevent false source agreement. |
| `fetcher_modelsdev.go` | HTTP fetcher for models.dev/api.json. Strips provider/model prefix from model IDs and converts per-million to per-token. | Provider IDs are only lowercased here, not aliased (azure_ai → azure happens in normalizer). Aliasing in the fetcher would bypass deduplication logic. |
| `reconciler.go` | Groups SourcePrice by (provider, modelID), runs pairwise findAgreement within priceTolerance, averages agreeing prices, calls repo.UpsertGlobalPrice. averagePrices uses prices[0] as metadata base (model name from first agreeing source). | findAgreement is O(n²) in sources per model — acceptable for <10 sources. optionalDecimalsAgree: one nil + one non-nil = disagree (not equivalent to zero). |
| `normalizer.go` | Wraps llmcost.NormalizeModelID: lowercases, trims whitespace, strips date version suffixes, aliases provider names (gemini→google, mistralai→mistral, etc.). | Normalization must be called in SyncJob.Run AFTER fetching but BEFORE deduplication. Do not call Normalize inside individual fetchers. |
| `fetcher.go` | Fetcher interface definition only. Source() must return a stable PriceSource constant. | PriceSource is stored in the DB in SourcePricesMap — changing a source key is a breaking migration. |

## Anti-Patterns

- Adding normalization logic inside fetchers — normalization belongs in the normalizer so all fetchers are handled uniformly
- Calling reconciler.Reconcile before deduplicateSourcePrices — allows false multi-source agreement from provider aliasing
- Making fetcher errors fatal — a single down source must not abort the sync for remaining models
- Storing per-million prices in SourcePrice — always convert to per-token before returning from Fetch()
- Adding a new fetcher without registering it in DefaultFetchers (or the caller's SyncJobConfig.Fetchers list)

## Decisions

- **Separate Fetcher, Normalizer, Reconciler into distinct types** — Allows testing each phase independently with mocks; new sources only require a new Fetcher without touching reconciliation logic.
- **Multi-source agreement threshold before writing global prices** — A single external source could be wrong or stale; requiring agreement from multiple sources prevents billing using bad data from a single upstream outage.

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
