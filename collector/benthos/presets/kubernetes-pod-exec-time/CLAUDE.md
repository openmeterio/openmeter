# kubernetes-pod-exec-time

<!-- archie:ai-start -->

> Benthos pipeline preset that periodically scrapes Kubernetes pod resources, maps each pod into a CloudEvents-shaped usage event (with CPU/memory/GPU metrics derived from pod spec), validates against CloudEvents schema, and forwards to OpenMeter. Designed for billing pod execution time.

## Patterns

**schedule + kubernetes_resources input** — Input combines `schedule` (wrapping `kubernetes_resources`) so the poll interval is controlled by `SCRAPE_INTERVAL`. The `label_selector` and `namespaces` fields filter which pods are scraped. (`input:
  schedule:
    input:
      kubernetes_resources:
        namespaces:
          - ${SCRAPE_NAMESPACE:}
        label_selector: "app=seed"
    interval: "${SCRAPE_INTERVAL:15s}"`)
**Bloblang mapping to CloudEvents structure** — The pipeline mapping must produce a CloudEvents-compliant object with `id`, `specversion`, `type`, `source`, `time`, `subject`, and `data`. Subject is read from the `openmeter.io/subject` annotation, falling back to pod name. (`root = {
  "id": uuid_v4(),
  "specversion": "1.0",
  "subject": this.metadata.annotations."openmeter.io/subject".or(this.metadata.name),
  ...
}`)
**data.openmeter.io/ annotation passthrough** — Arbitrary pod annotations prefixed with `data.openmeter.io/` are stripped of the prefix and merged into the CloudEvent `data` map. This lets operators inject custom dimensions without changing the pipeline. (`this.metadata.annotations.filter(item -> item.key.has_prefix("data.openmeter.io/")).map_each_key(key -> key.trim_prefix("data.openmeter.io/"))`)
**resource_quantity Bloblang function** — CPU/memory/GPU values from pod spec are converted via the custom `resource_quantity(...).number(0)` Bloblang function (provided by the benthos-collector binary). Always call `.number(0)` to default missing fields to zero. (`"cpu_request_millicores": this.spec.containers.map_each(container -> resource_quantity(container.resources.requests.cpu).number(0)).sum()`)
**switch output with DEBUG stdout branch** — Output uses `switch` with two cases: first unconditionally sends to the `openmeter` output (continue: true), second sends to stdout when `DEBUG=true`. The openmeter case must always be first and have `continue: true`. (`output:
  switch:
    cases:
      - check: ""
        continue: true
        output:
          openmeter: ...
      - check: '"${DEBUG:false}" == "true"'
        output:
          stdout: ...`)
**per-second rate metrics** — Each raw resource count is accompanied by a `*_per_second` field computed by dividing by `$duration_seconds` (the schedule interval in seconds). Both the raw and per-second fields must be emitted together. (`"cpu_request_millicores_per_second": .../  $duration_seconds`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `config.yaml` | Full Benthos pipeline: scheduled Kubernetes scrape, pod-to-CloudEvent mapping with resource_quantity functions, CloudEvents schema validation, openmeter output with optional DEBUG stdout. | `$duration_seconds` is derived from `meta("schedule_interval")` and is used as the divisor for per-second metrics — if `SCRAPE_INTERVAL` is changed, per-second values change proportionally. The `openmeter` output type is a custom Benthos plugin registered by the benthos-collector binary; it is not a standard Benthos output. |

## Anti-Patterns

- Using the standard `http_client` output instead of the custom `openmeter` output plugin — the plugin handles batching and auth internally
- Reordering the switch cases so stdout appears before the openmeter case — events will be dropped from the real output when DEBUG is true
- Omitting `.number(0)` on `resource_quantity()` calls — nil fields will cause mapping errors for pods without resource requests/limits
- Computing duration_seconds outside Bloblang (e.g., as a static env var) — it must be derived from the actual schedule_interval metadata so interval changes are reflected automatically

## Decisions

- **Derive duration_seconds from schedule interval metadata** — Makes per-second normalisation automatic and correct when operators change SCRAPE_INTERVAL without touching the pipeline mapping.
- **Annotation-driven subject and custom data dimensions** — Avoids hardcoding tenant/subject identity in the pipeline; operators annotate pods with openmeter.io/subject and data.openmeter.io/* to control metering attribution at the Kubernetes level.

<!-- archie:ai-end -->
