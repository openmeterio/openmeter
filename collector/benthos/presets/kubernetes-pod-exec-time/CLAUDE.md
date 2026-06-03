# kubernetes-pod-exec-time

<!-- archie:ai-start -->

> Benthos pipeline preset that periodically scrapes Kubernetes pod resources via kubernetes_resources, maps each pod into a CloudEvents-shaped usage event with CPU/memory/GPU metrics from pod spec and annotations, validates it, and forwards to OpenMeter. Designed for billing pod execution time.

## Patterns

**schedule + kubernetes_resources input** — Input combines schedule (wrapping kubernetes_resources) to control poll interval via SCRAPE_INTERVAL; label_selector and namespaces filter scraped pods. meta("schedule_interval") carries the interval into the pipeline for per-second normalisation. (`input:
  schedule:
    input:
      kubernetes_resources:
        namespaces: ["${SCRAPE_NAMESPACE:}"]
        label_selector: "app=seed"
    interval: "${SCRAPE_INTERVAL:15s}"`)
**Bloblang mapping to CloudEvents structure** — The mapping must produce a CloudEvents object with id (uuid_v4()), specversion: "1.0", type, source, time, subject, and data. Subject reads from openmeter.io/subject annotation, falling back to pod name. (`root = {"id": uuid_v4(), "specversion": "1.0", "subject": this.metadata.annotations."openmeter.io/subject".or(this.metadata.name)}`)
**data.openmeter.io/ annotation passthrough** — Pod annotations prefixed with data.openmeter.io/ are stripped of the prefix and merged into the CloudEvent data map, letting operators inject custom dimensions without touching the pipeline. (`this.metadata.annotations.filter(item -> item.key.has_prefix("data.openmeter.io/")).map_each_key(key -> key.trim_prefix("data.openmeter.io/"))`)
**resource_quantity Bloblang function with .number(0) default** — CPU/memory/GPU values from pod spec are converted via the custom resource_quantity(...) Bloblang function (registered by the benthos-collector binary). Always call .number(0) to default missing/nil fields to zero. (`"cpu_request_millicores": this.spec.containers.map_each(c -> resource_quantity(c.resources.requests.cpu).number(0)).sum()`)
**$duration_seconds from schedule interval metadata** — $duration_seconds is computed from meta("schedule_interval") parsed to seconds; all per-second metric fields divide by it. Must derive from metadata, not a static env var, so values stay correct when SCRAPE_INTERVAL changes. (`let duration_seconds = (meta("schedule_interval").parse_duration() / 1000 / 1000 / 1000).round().int64()`)
**switch output: openmeter first with continue:true, stdout second for DEBUG** — Output uses switch with two cases: first unconditionally sends to the openmeter output plugin with continue: true; second sends to stdout only when DEBUG=true. The openmeter case MUST be first with continue: true. (`output:
  switch:
    cases:
      - check: ""
        continue: true
        output: {openmeter: ...}
      - check: '"${DEBUG:false}" == "true"'
        output: {stdout: {}}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `config.yaml` | Full Benthos pipeline: scheduled Kubernetes scrape, pod-to-CloudEvent Bloblang mapping with resource_quantity functions, CloudEvents schema validation, custom openmeter output plugin with optional DEBUG stdout. | The openmeter output type is a custom Benthos plugin registered by the benthos-collector binary (not standard) and handles batching/auth internally. $duration_seconds is the divisor for all per-second fields; changing SCRAPE_INTERVAL changes these proportionally. |

## Anti-Patterns

- Using the standard http_client output instead of the custom openmeter output plugin — the plugin handles batching and auth internally
- Reordering switch cases so stdout appears before the openmeter case — real output events are dropped when DEBUG is true
- Omitting .number(0) on resource_quantity() calls — nil fields cause mapping errors for pods without resource requests/limits
- Computing duration_seconds as a static env var instead of deriving from meta("schedule_interval") — per-second normalisation breaks when SCRAPE_INTERVAL changes
- Adding route or path bindings to the kubernetes_resources input — it is a pull-based scraper, not an HTTP server

## Decisions

- **Derive duration_seconds from schedule interval metadata.** — Makes per-second normalisation automatic and correct when operators change SCRAPE_INTERVAL without touching the pipeline mapping.
- **Annotation-driven subject and custom data dimensions.** — Avoids hardcoding tenant/subject identity; operators annotate pods with openmeter.io/subject and data.openmeter.io/* to control metering attribution at the Kubernetes level without pipeline changes.

<!-- archie:ai-end -->
