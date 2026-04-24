# templates

<!-- archie:ai-start -->

> Helm templates for deploying all OpenMeter binaries (api, balance-worker, billing-worker, sink-worker, notification-service) plus optional dependencies (Svix, ClickHouse) as Kubernetes workloads. Every template shares a single ConfigMap-mounted config.yaml and the same image tag, enforcing homogeneous deployment.

## Patterns

**componentName for all resource names** — Every resource name uses `include "openmeter.componentName" (list . "<component>")` — never `.Release.Name` directly. This truncates to 63 chars and ensures DNS compliance. (`name: {{ include "openmeter.componentName" (list . "billing-worker") }}`)
**componentLabels / componentSelectorLabels for label blocks** — All metadata.labels use `openmeter.componentLabels`, all selector.matchLabels use `openmeter.componentSelectorLabels` with the component string. Never write app.kubernetes.io labels inline. (`labels:
  {{- include "openmeter.componentLabels" (list . "sink-worker") | nindent 4 }}`)
**checksum/config annotation on every workload** — Every Deployment and CronJob pod template must carry `checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}` to force pod rollout on config changes. (`annotations:
  checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}`)
**initContainers gated on *.enabled flags** — Init containers are only emitted when `Values.postgresql.enabled` or `Values.clickhouse.enabled` is true, using the `openmeter.init.postgres` / `openmeter.init.clickhouse` named templates from _helpers.tpl. (`{{ if or .Values.postgresql.enabled .Values.clickhouse.enabled -}}
initContainers:
  {{- if .Values.postgresql.enabled -}}
  {{- include "openmeter.init.postgres" (list .) | nindent 8 }}
  {{- end }}`)
**Shared image tag across all Deployments** — All OpenMeter binary containers use `{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}`. Never pin a different image per component. (`image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"`)
**Config via ConfigMap volume mount at /etc/openmeter** — Every workload mounts the `openmeter.fullname` ConfigMap at `/etc/openmeter` and passes `--config /etc/openmeter/config.yaml` as an arg. CA certificates are conditionally mounted at `/usr/local/share/ca-certificates` when `Values.caRootCertificates` is non-empty. (`volumeMounts:
  - name: config
    mountPath: /etc/openmeter
volumes:
  - name: config
    configMap:
      name: {{ include "openmeter.fullname" . }}`)
**optional-dependency resources wrapped in enabled guards** — Files for optional services (clickhouse.yaml, svix.yaml) wrap their entire content in `{{- if .Values.<service>.enabled -}}`. New optional dependencies must follow this pattern. (`{{- if .Values.svix.enabled -}}
apiVersion: apps/v1
kind: Deployment
...
{{- end }}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `_helpers.tpl` | Defines all named templates: openmeter.fullname, openmeter.componentName, openmeter.labels, openmeter.selectorLabels, openmeter.componentLabels, openmeter.componentSelectorLabels, openmeter.serviceAccountName, and init-container helpers (openmeter.init.netcat, openmeter.init.postgres, openmeter.init.clickhouse, openmeter.init.redis). All templates in this folder depend on these. | openmeter.componentName uses list syntax `(list . "component")` — passing a plain string will break. The name is truncated at 62-len(component) chars; long component names silently truncate. |
| `configmap.yaml` | Produces the config.yaml ConfigMap by deep-merging `Values.config` with auto-generated overrides for self-hosted Kafka/ClickHouse/Postgres/Svix via `mergeOverwrite`. Also emits a separate ca-certificates ConfigMap when `Values.caRootCertificates` is set. | The `helmValuesConfig` template uses `fromYaml`/`mergeOverwrite` — whitespace inside the define block matters for YAML validity. Self-hosted service addresses are derived from `openmeter.componentName` / `openmeter.fullname`, so adding a new self-hosted dependency must follow the same pattern here. |
| `deployment.yaml` | Defines five Deployments: api, balance-worker, notification-service, sink-worker, billing-worker. Each uses its own `.Values.<component>.replicaCount` and passes a binary-specific entrypoint command to `/entrypoint.sh`. | Health probes hit `/healthz/live` and `/healthz/ready` on port `telemetry-http` (10000), not the API port (80). Adding a new worker Deployment must replicate this probe pattern. The api Deployment additionally exposes port 80 (http). |
| `jobs.yaml` | Defines three CronJobs: subscription-sync, billing-collect-invoices, billing-advance-invoices. All use `concurrencyPolicy: Forbid` and `restartPolicy: Never`. Schedules come from `Values.jobs.<job>.schedule`. | New CronJobs must set `concurrencyPolicy: Forbid` to prevent overlapping runs. The job command invokes `openmeter-jobs` binary with subcommand args, not a long-running server. |
| `svix.yaml` | Conditionally deploys Svix as a Deployment + Service. Svix config is injected via environment variables (SVIX_REDIS_DSN, SVIX_DB_DSN, SVIX_JWT_SECRET), not the shared ConfigMap. | Svix uses its own image (`Values.svix.image`) and its own resources field (`Values.svix.resources`), not the shared `Values.image` or `Values.resources`. Init containers check postgres and redis readiness, not clickhouse. |
| `clickhouse.yaml` | Deploys a ClickHouseInstallation CRD (Altinity operator) only when `Values.clickhouse.enabled`. Requires the clickhouse-operator to be installed in the cluster. | This is a custom CRD (`clickhouse.altinity.com/v1`), not a standard Deployment — linting will fail without the CRD present. Storage sizes (3Gi data, 1Gi log) are hardcoded, not values-driven. |
| `ingress.yaml` | Standard Kubernetes Ingress for the api Service (`openmeter.componentName` with "api") only. Supports multi-Kube-version API version selection via `semverCompare`. | Ingress always targets the api component Service, not workers. Only emitted when `Values.ingress.enabled`. |

## Anti-Patterns

- Writing app.kubernetes.io/* labels inline instead of using openmeter.componentLabels / openmeter.componentSelectorLabels helpers
- Hardcoding resource names with .Release.Name directly instead of openmeter.componentName or openmeter.fullname
- Adding a new Deployment without a checksum/config annotation on the pod template (config changes won't trigger rollouts)
- Adding optional infrastructure (new self-hosted dependency) without an `{{- if .Values.<service>.enabled -}}` guard in both the resource file and configmap.yaml override block
- Using different image fields per component — all OpenMeter binaries must share Values.image.repository and Values.image.tag

## Decisions

- **Single shared ConfigMap for all workloads, merged at render time from Values.config plus per-flag overrides** — All OpenMeter binaries share the same config.yaml schema. Merging at Helm render time via mergeOverwrite avoids per-binary ConfigMaps while still allowing self-hosted dependency addresses to be injected automatically when *.enabled flags are set.
- **CronJobs use concurrencyPolicy: Forbid and restartPolicy: Never** — Billing advance, collect, and subscription-sync jobs are not idempotent under concurrent execution; Forbid prevents overlap. Never restart policy means the next scheduled run handles recovery rather than restarting a failed pod indefinitely.
- **Init containers generated from named templates in _helpers.tpl rather than inline YAML** — The same readiness-check logic (nc -z for postgres/clickhouse, PING loop for redis) is reused across all five Deployments and three CronJobs. Centralizing in _helpers.tpl ensures consistent busybox image tag and address derivation from openmeter.fullname.

## Example: Adding a new worker Deployment (e.g. charges-worker)

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "openmeter.componentName" (list . "charges-worker") }}
  labels:
    {{- include "openmeter.componentLabels" (list . "charges-worker") | nindent 4 }}
spec:
  replicas: {{ .Values.chargesWorker.replicaCount }}
  selector:
    matchLabels:
    {{- include "openmeter.componentSelectorLabels" (list . "charges-worker") | nindent 6 }}
  template:
    metadata:
      annotations:
        checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}
// ...
```

<!-- archie:ai-end -->
