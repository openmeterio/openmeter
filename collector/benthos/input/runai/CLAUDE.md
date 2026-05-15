# runai

<!-- archie:ai-start -->

> HTTP client library for the Run:ai API providing typed access to workloads, pods, and their metrics via a single resty.Client in service.go with automatic JWT token refresh on every non-token request. All API resource types follow a strict ListX/ListAllX pagination pair plus chunked metric fetching.

## Patterns

**Service method per API endpoint** — Every API endpoint is a method on *Service. Methods set Accept header, query params via SetQueryParams, and call SetResult with a typed pointer before executing the resty verb, then type-assert resp.Result(). (`resp.Result().(*ListPodsResponse)`)
**ListX + ListAllX pagination pair** — Each resource has a ListX (single page, max 500 limit enforced) and a ListAllX (loops until resp.Next == 0) pair. ListAllX calls s.resourceTypeMetrics.Set after collecting all pages. (`ListPods / ListAllPods, ListWorkloads / ListAllWorkloads`)
**Metric chunking at max 9 per request** — GetAllWorkloadWithMetrics and GetAllPodWithMetrics chunk MetricType slices with lo.Chunk(params.MetricType, 9) before calling GetWorkloadMetrics/GetPodMetrics to avoid exceeding the Run:ai API undocumented limit. (`for _, metricTypes := range lo.Chunk(params.MetricType, 9) { m, err := s.GetWorkloadMetrics(ctx, workload.ID, MeasurementParams{MetricType: metricTypes, ...}) }`)
**ResourceWithMetrics interface with MarshalJSON discriminator** — WorkloadWithMetrics and PodWithMetrics both implement ResourceWithMetrics (GetType, GetMetrics, json.Marshaler). MarshalJSON injects a resourceType discriminator field via an anonymous struct embed so downstream consumers can distinguish types. (`var _ ResourceWithMetrics = (*WorkloadWithMetrics)(nil)`)
**OnBeforeRequest JWT auto-refresh** — service.go registers an OnBeforeRequest hook that calls service.RefreshToken then SetAuthToken on every non-token request. OnAfterResponse clears token on 401 and records timing metrics with path normalization. (`if request.URL == "/api/v1/token" { return nil } // skip token endpoint`)
**Non-fatal per-resource metric errors** — In GetAllWorkloadWithMetrics and GetAllPodWithMetrics, a per-resource metric error is logged (s.logger.Errorf) and the loop continues with continue rather than returning the error — collection must not abort for all remaining resources. (`s.logger.With(...).Errorf("failed to get workload metrics: %w", err); continue`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Constructs *Service with a resty.Client; holds all shared state (logger, client, token, pageSize, resourceTypeMetrics). The single constructor NewService is the only entry point. | TimingMetrics path normalization uses regex in OnAfterResponse — add new endpoint path patterns here when adding new API paths, otherwise metrics will be untagged. |
| `token.go` | JWT lifecycle: NewToken posts to /api/v1/token, RefreshToken checks expiry via verifyToken (parses without signature verification), SetToken/GetToken are plain field accessors. | verifyToken uses jwt.WithoutClaimsValidation() — it only checks expiry, not signature. Token is cleared on 401 by OnAfterResponse in service.go. |
| `metrics.go` | Defines MetricType constants for both workload and pod scopes, MeasurementParams/Measurement/MeasurementResponse/Metrics types, and the four public methods: GetWorkloadMetrics, GetAllWorkloadWithMetrics, GetPodMetrics, GetAllPodWithMetrics. | PodMetricType and WorkloadMetricType share the MetricType string type — some constant names collide (e.g. CPU_MEMORY_USAGE_BYTES, GPU_UTILIZATION appear in both groups). Don't assume a constant is scoped to one resource type. |
| `workloads.go` | Defines Workload struct, ListWorkloads (with filterBy query param excluding non-running phases), and ListAllWorkloads. | ListWorkloads filters workloads client-side by RunningPods > 0 AFTER the API call in addition to the filterBy query param. Adding new phases to include requires editing both the filteredOutPhases slice and the lo.Filter predicate. |
| `pods.go` | Defines Pod struct and ListPods/ListAllPods. ListAllPods calls s.resourceTypeMetrics.Set(int64(len(pods)), "pod") after pagination completes. | ListPods hardcodes completed=false query param — only running pods are returned. Do not use this to fetch completed pods. |

## Anti-Patterns

- Adding a new API resource without providing both a ListX (single page, max 500) and ListAllX (full pagination loop calling resourceTypeMetrics.Set) method pair.
- Calling GetWorkloadMetrics or GetPodMetrics with more than 9 MetricType values in one request — always chunk with lo.Chunk(..., 9).
- Returning an error from the per-resource metric loop in GetAllWorkloadWithMetrics/GetAllPodWithMetrics — log and continue instead to avoid aborting collection for all remaining resources.
- Adding timing metrics without normalizing the URL path via the regex block in OnAfterResponse in service.go — untagged paths produce high-cardinality metric series.
- Modifying verifyToken to verify JWT signature — the Run:ai JWT is intentionally parsed without signature verification here by design.

## Decisions

- **Single *Service struct holds all HTTP state including token and resty.Client** — The OnBeforeRequest hook has a closure over *Service, allowing RefreshToken to be called transparently before every request without threading credentials through each method signature.
- **Metric fetches chunk at 9 MetricType values per request** — The Run:ai API has an undocumented maximum on simultaneous metric types; chunking at 9 stays safely under that limit while maps.Copy merges partial results into a single Metrics.Values map.
- **ResourceWithMetrics interface + MarshalJSON discriminator field** — Allows the Benthos input plugin to handle both WorkloadWithMetrics and PodWithMetrics polymorphically while producing self-describing JSON with a resourceType field for downstream consumers to distinguish types without schema inspection.

## Example: Adding a new resource type (e.g. nodes) with pagination, metric support, and ResourceWithMetrics implementation

```
// nodes.go
package runai

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"time"

	"github.com/samber/lo"
)

type Node struct {
// ...
```

<!-- archie:ai-end -->
