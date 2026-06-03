# runai

<!-- archie:ai-start -->

> HTTP client library for the Run:ai API providing typed access to workloads, pods, and their metrics via a single resty.Client (service.go) with automatic JWT refresh on every non-token request. Resources follow a strict ListX/ListAllX pagination pair plus chunked metric fetching.

## Patterns

**Service method per API endpoint** — Every endpoint is a method on *Service: set Accept header, query params via SetQueryParams, call SetResult with a typed pointer, execute the resty verb, then type-assert resp.Result(). (`resp.Result().(*ListPodsResponse)`)
**ListX + ListAllX pagination pair** — Each resource has ListX (single page, max 500 limit) and ListAllX (loops until resp.Next == 0). ListAllX calls s.resourceTypeMetrics.Set after collecting all pages. (`ListPods / ListAllPods, ListWorkloads / ListAllWorkloads`)
**Metric chunking at max 9 per request** — GetAllWorkloadWithMetrics/GetAllPodWithMetrics chunk MetricType slices with lo.Chunk(..., 9) before GetWorkloadMetrics/GetPodMetrics to stay under the Run:ai undocumented limit. (`for _, metricTypes := range lo.Chunk(params.MetricType, 9) { ... }`)
**ResourceWithMetrics interface + MarshalJSON discriminator** — WorkloadWithMetrics and PodWithMetrics implement ResourceWithMetrics (GetType, GetMetrics, json.Marshaler); MarshalJSON injects a resourceType field via an anonymous struct embed. (`var _ ResourceWithMetrics = (*WorkloadWithMetrics)(nil)`)
**OnBeforeRequest JWT auto-refresh** — service.go registers OnBeforeRequest to RefreshToken then SetAuthToken on every non-token request; OnAfterResponse clears the token on 401 and records normalized timing metrics. (`if request.URL == "/api/v1/token" { return nil } // skip token endpoint`)
**Non-fatal per-resource metric errors** — In GetAllWorkloadWithMetrics/GetAllPodWithMetrics a per-resource metric error is logged and the loop continues (continue) rather than aborting the whole collection. (`s.logger.With(...).Errorf("failed to get workload metrics: %w", err); continue`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Constructs *Service with a resty.Client and all shared state (logger, client, token, pageSize, resourceTypeMetrics); NewService is the only entry point. | TimingMetrics path normalization uses a regex in OnAfterResponse — add new endpoint path patterns here or metrics will be untagged/high-cardinality. |
| `token.go` | JWT lifecycle: NewToken posts to /api/v1/token, RefreshToken checks expiry via verifyToken, SetToken/GetToken are field accessors. | verifyToken uses jwt.WithoutClaimsValidation() — it only checks expiry, not signature. Token is cleared on 401 by OnAfterResponse. |
| `metrics.go` | MetricType constants (workload and pod scopes), MeasurementParams/Measurement/MeasurementResponse/Metrics, and GetWorkloadMetrics/GetAllWorkloadWithMetrics/GetPodMetrics/GetAllPodWithMetrics. | PodMetricType and WorkloadMetricType share the MetricType string type; some constant names collide across groups — don't assume a constant is scoped to one resource. |
| `workloads.go` | Workload struct, ListWorkloads (filterBy excluding non-running phases), ListAllWorkloads. | ListWorkloads also filters client-side by RunningPods > 0 after the API call; including new phases requires editing both filteredOutPhases and the lo.Filter predicate. |
| `pods.go` | Pod struct, ListPods/ListAllPods; ListAllPods calls resourceTypeMetrics.Set(len, "pod") after pagination. | ListPods hardcodes completed=false — only running pods are returned; do not use it to fetch completed pods. |

## Anti-Patterns

- Adding a resource without both ListX (single page, max 500) and ListAllX (full loop calling resourceTypeMetrics.Set)
- Calling GetWorkloadMetrics/GetPodMetrics with more than 9 MetricType values — always chunk with lo.Chunk(..., 9)
- Returning an error from the per-resource metric loop instead of log-and-continue
- Adding timing metrics without normalizing the URL path via the OnAfterResponse regex
- Modifying verifyToken to verify the JWT signature — it is intentionally parsed without signature verification

## Decisions

- **Single *Service holds all HTTP state including token and resty.Client** — The OnBeforeRequest hook closes over *Service so RefreshToken runs transparently before each request without threading credentials through every method.
- **Metric fetches chunk at 9 MetricType values per request** — The Run:ai API has an undocumented max on simultaneous metric types; chunking at 9 stays safely under it while maps.Copy merges partial results.
- **ResourceWithMetrics + MarshalJSON discriminator** — Lets the Benthos input handle WorkloadWithMetrics and PodWithMetrics polymorphically while producing self-describing JSON with a resourceType field.

<!-- archie:ai-end -->
