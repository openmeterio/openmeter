# sync

<!-- archie:ai-start -->

> External price synchronization pipeline: fetches raw LLM prices from external sources (models.dev), normalizes provider/model IDs, deduplicates within-source collisions from normalization, then reconciles multi-source agreement into global llmcostprice rows via Adapter.UpsertGlobalPrice. Driven by cmd/jobs llm-cost sync.

## Patterns

**Fetcher interface per source in a dedicated file** — Each source implements Fetcher (Source() PriceSource, Fetch(ctx) ([]SourcePrice, error)); new sources add fetcher_<source>.go and register in DefaultFetchers. Source() returns a stable PriceSource constant used as DB key. (`func (f *modelsDevFetcher) Source() llmcost.PriceSource { return "models_dev" }`)
**Four-phase SyncJob.Run with strict ordering** — Order is mandatory: (1) fetch+normalize per source, (2) deduplicateSourcePrices, (3) optional filter, (4) Reconcile. Do not reorder. (`allPrices = deduplicateSourcePrices(allPrices); if j.filter != nil { ... }; return j.reconciler.Reconcile(ctx, allPrices)`)
**Per-million to per-token conversion in fetchers** — External APIs provide per-million-token prices; fetchers divide by 1,000,000 with alpacadecimal before returning SourcePrice so downstream works in per-token units. (`InputPerToken: alpacadecimal.NewFromFloat(*model.Cost.Input).Div(alpacadecimal.NewFromFloat(1_000_000))`)
**deduplicateSourcePrices prevents false agreement** — Provider normalization (azure_ai→azure) can collapse two raw entries from one source into the same (source, provider, modelID) key; this runs after normalize, before reconcile. (`key := sourceModelKey{Source: p.Source, Provider: string(p.Provider), ModelID: p.ModelID}; if idx, exists := seen[key]; exists { ... }`)
**minAgreement capped at fetcher count** — NewSyncJob caps minAgreement at len(fetchers) so single-source deployments work without manual override. (`if n := len(fetchers); n > 0 && minAgreement > n { minAgreement = n }`)
**Non-fatal fetcher errors** — A failing fetcher is logged and skipped (continue); a single down source must not abort the sync. (`prices, err := f.Fetch(ctx); if err != nil { j.logger.Error(...); continue }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `sync.go` | SyncJob orchestrator holding fetchers, normalizer, reconciler, optional filter; Run() is the only public entry point. | deduplicateSourcePrices must run AFTER normalization and BEFORE filter+reconcile — the order is load-bearing. |
| `fetcher_modelsdev.go` | HTTP fetcher for models.dev/api.json; strips provider/model prefix and converts per-million to per-token. | Provider IDs are only lowercased here, not aliased — aliasing happens in the normalizer; aliasing in the fetcher would bypass deduplication. |
| `reconciler.go` | Groups SourcePrice by (provider, modelID), runs pairwise findAgreement within priceTolerance, averages agreeing prices, calls repo.UpsertGlobalPrice. | findAgreement is O(n²) in sources per model (fine for <10 sources). optionalDecimalsAgree: one nil + one non-nil disagree (not equal to zero). |
| `normalizer.go` | Wraps llmcost.NormalizeModelID: lowercases, trims, strips date version suffixes, aliases provider names (gemini→google, mistralai→mistral). | Normalization must run in SyncJob.Run after fetching but before deduplication; do not call Normalize inside fetchers. |
| `fetcher.go` | Fetcher interface definition only. | Source() must return a stable PriceSource constant; it is stored in the DB SourcePricesMap, so changing a source key is a breaking migration. |

## Anti-Patterns

- Adding normalization logic inside fetchers — it belongs in the normalizer so all fetchers are handled uniformly.
- Calling reconciler.Reconcile before deduplicateSourcePrices — allows false multi-source agreement from provider aliasing.
- Making fetcher errors fatal — a single down source must not abort the sync.
- Storing per-million prices in SourcePrice instead of converting to per-token in Fetch().
- Adding a new fetcher without registering it in DefaultFetchers or SyncJobConfig.Fetchers.

## Decisions

- **Fetcher, Normalizer, Reconciler are distinct types.** — Each phase is independently testable with mocks; new sources only require a new Fetcher without touching reconciliation.
- **Require multi-source agreement before writing global prices.** — A single external source could be wrong or stale; agreement from multiple sources prevents billing on bad data during an upstream outage.

## Example: Adding a new price source fetcher

```
// fetcher_litellm.go
package sync

import (
	"net/http"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
)

type litellmFetcher struct{ client *http.Client }

func NewLiteLLMFetcher(client *http.Client) Fetcher { return &litellmFetcher{client: client} }

func (f *litellmFetcher) Source() llmcost.PriceSource { return "litellm" }
// Fetch divides external per-million costs by 1_000_000 before returning SourcePrice
```

<!-- archie:ai-end -->
