# runai

<!-- archie:ai-start -->

> HTTP client library for the Run:ai API, providing typed access to workloads, pods, and their metrics. All network calls go through a single resty.Client configured in service.go with automatic JWT token refresh on every non-token request.

## Patterns

**Service method per API endpoint** — Every API endpoint is a method on *Service (e.g. ListPods, ListWorkloads, GetWorkloadMetrics). Methods set Accept header, query params via SetQueryParams, and call SetResult with a typed pointer before executing the resty verb. (`resp.Result().(*ListPodsResponse)`)
**Paginated list helpers: ListX + ListAllX** — Each resource has a ListX (single page, max 500) and a ListAllX (loops until resp.Next == 0) pair. ListAllX calls s.resourceTypeMetrics.Set after collecting all pages. (`ListPods / ListAllPods, ListWorkloads / ListAllWorkloads`)
**Metric chunking (max 9 per request)** — GetAllWorkloadWithMetrics and GetAllPodWithMetrics chunk MetricType slices with lo.Chunk(params.MetricType, 9) before calling GetWorkloadMetrics/GetPodMetrics to avoid exceeding API limits. (`for _, metricTypes := range lo.Chunk(params.MetricType, 9) { ... }`)
**ResourceWithMetrics interface for polymorphic metric resources** — WorkloadWithMetrics and PodWithMetrics both implement ResourceWithMetrics (GetType, GetMetrics, json.Marshaler). MarshalJSON injects a resourceType discriminator field via an anonymous struct embed. (`var _ ResourceWithMetrics = (*WorkloadWithMetrics)(nil)`)
**OnBeforeRequest JWT auto-refresh** — service.go registers an OnBeforeRequest hook that calls service.RefreshToken then SetAuthToken on every non-token request. OnAfterResponse clears token on 401 and records timing metrics with path normalization. (`if request.URL == "/api/v1/token" { return nil }`)
**Non-fatal per-resource metric errors** — In GetAllWorkloadWithMetrics and GetAllPodWithMetrics, a per-resource metric error is logged (s.logger.Errorf) and the loop continues with continue rather than returning the error. (`s.logger.With(...).Errorf("failed to get workload metrics: %w", err); continue`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Constructs *Service with a resty.Client; holds all shared state (logger, client, token, pageSize, resourceTypeMetrics). The single constructor NewService is the only entry point. | TimingMetrics path normalization uses regex in OnAfterResponse — add new endpoints here when adding new API paths, otherwise metrics will be untagged. |
| `token.go` | JWT lifecycle: NewToken posts to /api/v1/token, RefreshToken checks expiry via verifyToken (parses without signature verification), SetToken/GetToken are plain field accessors. | verifyToken uses jwt.WithoutClaimsValidation() — it only checks expiry, not signature. Token is cleared on 401 by OnAfterResponse in service.go. |
| `metrics.go` | Defines MetricType constants for both workload and pod scopes, MeasurementParams/Measurement/MeasurementResponse/Metrics types, and the four public methods: GetWorkloadMetrics, GetAllWorkloadWithMetrics, GetPodMetrics, GetAllPodWithMetrics. | PodMetricType and WorkloadMetricType share the MetricType string type — some constant names collide (e.g. CPU_MEMORY_USAGE_BYTES, GPU_UTILIZATION appear in both groups). Don't assume a constant is scoped. |
| `workloads.go` | Defines Workload struct, ListWorkloads (with hardcoded phase filter excluding non-running workloads), and ListAllWorkloads. | ListWorkloads filters workloads client-side by RunningPods > 0 after the API call; the phase filter is also applied as a filterBy query param. Adding new phases to include requires editing both. |
| `pods.go` | Defines Pod struct and ListPods/ListAllPods. ListAllPods calls s.resourceTypeMetrics.Set(int64(len(pods)), "pod") after pagination completes. | ListPods hardcodes completed=false query param; running pod listing only. |

## Anti-Patterns

- Adding a new API resource without providing both a ListX (single page) and ListAllX (pagination loop) method.
- Calling GetWorkloadMetrics or GetPodMetrics with more than 9 MetricType values in one request — always chunk with lo.Chunk(..., 9).
- Returning an error from the per-resource metric loop instead of logging and continuing — this would abort collection for all remaining resources.
- Editing token verification to check signature — the Run:ai JWT is not verified by signature here by design.
- Adding timing metrics without normalizing the URL path in the OnAfterResponse regex block in service.go.

## Decisions

- **Single *Service struct holds all HTTP state including token and resty.Client** — Simplifies token refresh: the OnBeforeRequest hook has a closure over *Service so it can call RefreshToken without passing credentials through every method.
- **Metric fetches chunk at 9 MetricType values per request** — The Run:ai API has an undocumented max on simultaneous metric types; chunking at 9 stays safely under that limit while maps.Copy merges results into a single Metrics struct.
- **ResourceWithMetrics interface + MarshalJSON discriminator** — Allows the Benthos input plugin to handle both WorkloadWithMetrics and PodWithMetrics polymorphically while producing self-describing JSON with a resourceType field for downstream consumers.

## Example: Adding a new resource type (e.g. nodes) with pagination and metric support

```
// nodes.go
package runai

import (
	"context"
	"fmt"
	"net/http"
)

type Node struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ListNodesResponse struct {
// ...
```

<!-- archie:ai-end -->
