# presets

<!-- archie:ai-start -->

> Self-contained YAML pipeline presets for the Benthos collector with no Go code. Each sub-folder is one deployable pipeline config (http-server, kubernetes-pod-exec-time) that composes the custom input/output plugins registered in collector/benthos. Operators deploy these directly without recompiling.

## Patterns

**Environment variable substitution for secrets** — All URLs, tokens, and credentials use ${ENV_VAR:default} substitution; never hardcode values. (`url: "${OPENMETER_URL:http://localhost:8888}"
token: "${OPENMETER_TOKEN:}"`)
**catch-log-delete for validation failures** — After json_schema validation, a catch block logs the error and deletes the message to prevent pipeline stalls; a missing catch propagates validation errors and stalls the pipeline. (`processors:
  - json_schema: ...
  - catch:
    - log: { level: ERROR, message: "${!error()}" }
    - mapping: root = deleted()`)
**switch output with DEBUG stdout branch** — When DEBUG=true the pipeline outputs to stdout; the openmeter output case must be the last switch case so the DEBUG branch short-circuits correctly. (`output:
  switch:
    cases:
      - check: env("DEBUG") == "true"
        output: { stdout: {} }
      - output: { openmeter: { ... } }`)
**duration_seconds from schedule_interval metadata** — In kubernetes-pod-exec-time, duration_seconds is derived from meta("schedule_interval") at Bloblang mapping time, not from a static env var, so it stays accurate across interval changes or leader failover. (`let duration_seconds = meta("schedule_interval").parse_duration_iso8601() / 1000000000`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `http-server/config.yaml` | Receive-validate-forward pipeline: HTTP input, CloudEvents JSON schema validation, SQLite buffer, openmeter output. | SQLite buffer requires a post-processor split; changing buffer type breaks the split; sync_response metadata must be set before buffering or the HTTP client receives no status. |
| `kubernetes-pod-exec-time/config.yaml` | Kubernetes pod billing pipeline: schedule+kubernetes_resources input, Bloblang CloudEvents mapping, openmeter output. | duration_seconds must derive from schedule_interval metadata; all resource_quantity() calls need .number(0) fallback for pods missing requests/limits. |

## Anti-Patterns

- Adding transformation or enrichment logic to http-server/config.yaml — it is receive-validate-forward only.
- Hardcoding OPENMETER_URL or OPENMETER_TOKEN instead of env-var substitution with defaults.
- Using the generic http_client output instead of the custom openmeter output plugin — the plugin handles batching and auth internally.
- Reordering switch cases so stdout appears after the openmeter case — DEBUG mode no longer short-circuits the real output.
- Omitting .number(0) on resource_quantity() calls — pods without resource requests produce nil mapping errors.

## Decisions

- **Presets are pure YAML with no Go code.** — Operators deploy these directly without recompilation; config files allow tuning schedules, selectors, and outputs without a code change.
- **kubernetes-pod-exec-time derives duration_seconds from schedule_interval metadata at mapping time.** — The actual interval may differ from the configured value due to leader failover or scheduler jitter; reading it from message metadata guarantees accuracy.

<!-- archie:ai-end -->
