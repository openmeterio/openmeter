# presets

<!-- archie:ai-start -->

> Self-contained YAML pipeline presets for the Benthos collector — no Go code. Each sub-folder is one deployable pipeline config (http-server for CloudEvent ingestion, kubernetes-pod-exec-time for Kubernetes billing). They compose the custom input/output plugins registered in collector/benthos/input and collector/benthos/output.

## Patterns

**Environment variable substitution for secrets** — All URLs, tokens, and credentials use ${ENV_VAR:default} substitution. Never hardcode values. (`url: "${OPENMETER_URL:http://localhost:8888}"
token: "${OPENMETER_TOKEN:}"`)
**catch-log-delete for validation failures** — After json_schema validation, a catch block logs the error and deletes the message to prevent pipeline stalls. Never let a validation error propagate uncaught. (`processors:
  - json_schema: ...
  - catch:
    - log: { level: ERROR, message: ... }
    - mapping: root = deleted()`)
**switch output with DEBUG stdout branch** — When DEBUG=true the pipeline outputs to stdout instead of the real output. The openmeter case must always come last in the switch so the debug branch short-circuits correctly. (`output:
  switch:
    cases:
      - check: env("DEBUG") == "true"
        output: { stdout: {} }
      - output: { openmeter: { ... } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `presets/http-server/config.yaml` | Receive-validate-forward pipeline. HTTP input → CloudEvents JSON schema validation → SQLite buffer → openmeter output. | SQLite buffer requires a post-processor split; changing the buffer type will break the split step. sync_response metadata must be set before buffering or the HTTP client receives no response. |
| `presets/kubernetes-pod-exec-time/config.yaml` | Kubernetes pod billing pipeline. schedule+kubernetes_resources input → Bloblang CloudEvents mapping → openmeter output. | duration_seconds is derived from schedule_interval metadata — do not compute it as a static env var. resource_quantity() calls must use .number(0) fallback for pods missing resource limits. |

## Anti-Patterns

- Adding transformation or enrichment logic to http-server/config.yaml — it is a receive-validate-forward pipeline only.
- Hardcoding OPENMETER_URL or OPENMETER_TOKEN instead of env var substitution.
- Using the generic http_client output instead of the custom openmeter output plugin — the plugin handles batching and auth.
- Reordering switch cases so stdout appears after openmeter — DEBUG mode will no longer short-circuit the real output.
- Omitting .number(0) on resource_quantity() calls — pods without resource requests will produce nil mapping errors.

## Decisions

- **Presets are pure YAML with no Go code.** — Operators deploy these directly without recompilation; keeping them as config files allows tuning schedules, selectors, and outputs without a code change.
- **kubernetes-pod-exec-time derives duration_seconds from schedule_interval metadata at mapping time.** — The actual interval may differ from the configured value due to leader failover or scheduler jitter; reading it from message metadata guarantees accuracy.

<!-- archie:ai-end -->
