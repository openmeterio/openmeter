# templates

<!-- archie:ai-start -->

> Helm chart templates that render every Kubernetes object for an OpenMeter install: the API Deployment, four worker Deployments (balance/notification/sink/billing), three billing CronJobs, Service, Ingress, ConfigMap, ServiceAccount, and optional self-hosted ClickHouse/Svix. All naming, labeling, and dependency-wait logic flows through shared `_helpers.tpl` definitions.

## Patterns

**Component naming via openmeter.componentName** — Every workload's metadata.name is built with `include "openmeter.componentName" (list . "<component>")`, never hardcoded. This truncates the fullname to fit the 63-char DNS limit while keeping the component suffix. (`name: {{ include "openmeter.componentName" (list . "billing-worker") }}`)
**Component labels & selector labels** — Resource labels use `openmeter.componentLabels (list . "x")` and selectors use `openmeter.componentSelectorLabels (list . "x")`. The component string passed to selector and metadata labels of a workload MUST match so Pods are selected correctly. (`{{- include "openmeter.componentSelectorLabels" (list . "api") | nindent 6 }}`)
**Config-change rollout via checksum annotation** — Every Pod template adds `checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}` so config changes trigger a rolling restart. New Deployments/Jobs must replicate this annotation. (`checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}`)
**Dependency-wait initContainers** — Workloads gate on Postgres/ClickHouse readiness via `openmeter.init.postgres`/`openmeter.init.clickhouse` (svix.yaml uses init.redis), each guarded by the corresponding `.Values.*.enabled` flag. These are only added when the self-hosted dependency is enabled. (`{{- include "openmeter.init.postgres" (list .) | nindent 8 }}`)
**Shared container scaffolding** — All workload containers share the same image (`.Values.image.repository:tag|default .Chart.AppVersion`), `/entrypoint.sh` command, telemetry port 10000, /healthz/live & /healthz/ready probes, config volume at /etc/openmeter, and optional ca-certificates volume gated on `.Values.caRootCertificates`. (`command: ["/entrypoint.sh", "openmeter-balance-worker"]`)
**Self-hosted dependency config injection** — configmap.yaml defines `openmeter.helmValuesConfig` which conditionally emits kafka/clickhouse/postgres/svix blocks per `.Values.*.enabled`, then `mergeOverwrite`s them onto `.Values.config`. New self-hosted dependency wiring goes here, not into static config. (`{{- $cfg := mergeOverwrite $config $valuesConfig -}}`)
**Optional resources gated by enabled flags** — clickhouse.yaml, svix.yaml, ingress.yaml, and serviceaccount.yaml wrap their whole body in `{{- if .Values.X.enabled -}}` / `{{- if .Values.ingress.enabled -}}` so they render nothing when disabled. (`{{- if .Values.svix.enabled -}} ... {{- end }}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `_helpers.tpl` | Single source of named templates: name/fullname/chart, labels, selectorLabels, componentName/componentLabels/componentSelectorLabels, serviceAccountName, and the init.netcat/init.postgres/init.clickhouse/init.redis dependency waiters. | componentName truncates to `63 - 1 - len(component)`; a new long component name silently collides if it exceeds the budget. Add new named helpers here rather than inlining in templates. |
| `deployment.yaml` | Renders five Deployments (api, balance-worker, notification-service, sink-worker, billing-worker) separated by `---`, each differing mainly by component name, replicaCount value, and entrypoint subcommand. | Only the api Deployment exposes the http port 80 and serves traffic; workers expose only telemetry-http 10000. Copy an existing block wholesale when adding a worker; don't forget the checksum annotation and initContainers guards. |
| `jobs.yaml` | Three batch/v1 CronJobs (subscription-sync, billing-collect-invoices, billing-advance-invoices) running `openmeter-jobs` with args like `billing collect all`; all use concurrencyPolicy: Forbid and restartPolicy: Never. | Schedules come from `.Values.jobs.<name>.schedule`. The double-nested template metadata (jobTemplate.spec.template.metadata) both need labels/annotations; nindent levels differ (8 vs 12). |
| `configmap.yaml` | Builds config.yaml from `.Values.config` merged with auto-generated self-hosted dependency overrides, plus an optional ca-certificates ConfigMap from `.Values.caRootCertificates`. | helmValuesConfig uses `include` then `fromYaml`; indentation inside the conditional blocks is YAML-significant. The CA cert keys are lower-cased and suffixed `.crt`. |
| `svix.yaml` | Optional self-hosted Svix Deployment + Service, gated on `.Values.svix.enabled`, wiring SVIX_REDIS_DSN / SVIX_DB_DSN with defaults pointing at the in-chart redis-master and postgres. | Uses `.Values.svix.image.*` (not the shared `.Values.image`) and init.redis/init.postgres only. Container port is 8071. |
| `ingress.yaml` | Version-aware Ingress (networking.k8s.io/v1 vs v1beta1 vs extensions) routing to the api component Service, gated on `.Values.ingress.enabled`. | Backend service name comes from `openmeter.componentName (list . "api")` and must match service.yaml; the apiVersion/backend shape branches on `.Capabilities.KubeVersion`. |
| `clickhouse.yaml` | Optional ClickHouseInstallation CR (clickhouse.altinity.com/v1) for self-hosted analytics storage, gated on `.Values.clickhouse.enabled`. | Requires the Altinity ClickHouse operator CRD installed; service address `clickhouse-<fullname>:9000` is referenced by configmap.yaml and init.clickhouse. |
| `service.yaml` | ClusterIP/typed Service for the api component exposing `.Values.service.port` to targetPort http. | Only the api workload has a Service; selector must use componentSelectorLabels with "api". |

## Anti-Patterns

- Hardcoding a resource name instead of `openmeter.componentName` — breaks the 63-char DNS truncation and label/selector matching.
- Adding a Deployment/CronJob without the `checksum/config` annotation — config changes won't trigger a rollout.
- Adding self-hosted dependency settings as static `.Values.config` instead of conditional blocks in `helmValuesConfig` — they leak even when the dependency is disabled.
- Emitting an optional resource (svix/clickhouse/ingress) without the `{{- if .Values.X.enabled -}}` guard.
- Mismatching the component string between a workload's labels, selectorLabels, and metadata.name so Pods fail selection.

## Decisions

- **All naming and labeling are centralized in _helpers.tpl named templates rather than inlined.** — Keeps the 63-char DNS truncation logic and Kubernetes recommended labels consistent across the many Deployments/Jobs and avoids per-resource drift.
- **Self-hosted dependencies (postgres/clickhouse/kafka/redis/svix) are first-class but optional, gated by `.Values.*.enabled`, and NOTES.txt warns against using them in production.** — The chart doubles as an all-in-one demo install and a production deployment that points at managed external services.
- **Dependency readiness is enforced with netcat initContainers rather than relying on application retry.** — Ensures Postgres/ClickHouse/Redis are reachable before the OpenMeter process starts migrations or consumers, avoiding crash-loops at startup.

## Example: Standard worker Deployment block (name, labels, config checksum, dependency init, shared container)

```
metadata:
  name: {{ include "openmeter.componentName" (list . "sink-worker") }}
  labels:
    {{- include "openmeter.componentLabels" (list . "sink-worker") | nindent 4 }}
spec:
  selector:
    matchLabels:
    {{- include "openmeter.componentSelectorLabels" (list . "sink-worker") | nindent 6 }}
  template:
    metadata:
      annotations:
        checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}
    spec:
      {{ if or .Values.postgresql.enabled .Values.clickhouse.enabled -}}
      initContainers:
// ...
```

<!-- archie:ai-end -->
