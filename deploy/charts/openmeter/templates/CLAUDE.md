# templates

<!-- archie:ai-start -->

> Helm templates for deploying all OpenMeter binaries (api, balance-worker, billing-worker, sink-worker, notification-service) plus optional dependencies (Svix, ClickHouse) as Kubernetes workloads. Every template shares a single ConfigMap-mounted config.yaml and the same image tag, enforcing homogeneous deployment across all binaries.

## Patterns

**componentName for all resource names** — Every resource name must use `include "openmeter.componentName" (list . "<component>")` — never `.Release.Name` directly. The helper truncates to 63 chars minus component length for DNS compliance. (`name: {{ include "openmeter.componentName" (list . "billing-worker") }}`)
**componentLabels / componentSelectorLabels for all label blocks** — All metadata.labels use `openmeter.componentLabels`, all selector.matchLabels use `openmeter.componentSelectorLabels` with the component string. Never write app.kubernetes.io labels inline. (`labels:
  {{- include "openmeter.componentLabels" (list . "sink-worker") | nindent 4 }}`)
**checksum/config annotation on every workload pod template** — Every Deployment and CronJob pod template must carry `checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}` to force pod rollout on config changes. (`annotations:
  checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}`)
**initContainers gated on *.enabled flags via named templates** — Init containers are only emitted when `Values.postgresql.enabled` or `Values.clickhouse.enabled` is true, using `openmeter.init.postgres` / `openmeter.init.clickhouse` named templates from _helpers.tpl. Never inline the readiness-check logic. (`{{ if or .Values.postgresql.enabled .Values.clickhouse.enabled -}}
initContainers:
  {{- if .Values.postgresql.enabled -}}
  {{- include "openmeter.init.postgres" (list .) | nindent 8 }}
  {{- end }}`)
**Shared image tag across all OpenMeter Deployments** — All OpenMeter binary containers use `{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}`. Never pin a different image per component. Svix is the only exception — it uses `Values.svix.image`. (`image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"`)
**Config via ConfigMap volume mount at /etc/openmeter** — Every workload mounts the `openmeter.fullname` ConfigMap at `/etc/openmeter` and passes `--config /etc/openmeter/config.yaml` as an arg. CA certificates are conditionally mounted at `/usr/local/share/ca-certificates` when `Values.caRootCertificates` is non-empty. (`volumeMounts:
  - name: config
    mountPath: /etc/openmeter
volumes:
  - name: config
    configMap:
      name: {{ include "openmeter.fullname" . }}`)
**Optional-dependency resources wrapped in enabled guards** — Files for optional services (clickhouse.yaml, svix.yaml) wrap their entire content in `{{- if .Values.<service>.enabled -}}`. New optional dependencies must follow this pattern and also add an override block in configmap.yaml's helmValuesConfig template. (`{{- if .Values.svix.enabled -}}
apiVersion: apps/v1
kind: Deployment
...
{{- end }}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `_helpers.tpl` | Defines all named templates: openmeter.fullname, openmeter.componentName, openmeter.labels, openmeter.selectorLabels, openmeter.componentLabels, openmeter.componentSelectorLabels, openmeter.serviceAccountName, and init-container helpers (openmeter.init.netcat, openmeter.init.postgres, openmeter.init.clickhouse, openmeter.init.redis). All other templates depend on these. | openmeter.componentName uses list syntax `(list . "component")` — passing a plain string will break. The name is truncated at `62 - len(component)` chars; long component names silently truncate. openmeter.init.redis uses a PING loop unlike netcat-based postgres/clickhouse init containers. |
| `configmap.yaml` | Produces the config.yaml ConfigMap by deep-merging `Values.config` with auto-generated overrides for self-hosted Kafka/ClickHouse/Postgres/Svix via `mergeOverwrite`. Also emits a separate ca-certificates ConfigMap when `Values.caRootCertificates` is set. | The `helmValuesConfig` define block uses `fromYaml`/`mergeOverwrite` — whitespace inside the define block matters for YAML validity. Adding a new self-hosted dependency requires a new `if .Values.<dep>.enabled` block here mirroring the override keys from config.example.yaml. |
| `deployment.yaml` | Defines five Deployments: api, balance-worker, notification-service, sink-worker, billing-worker. Each uses its own `.Values.<component>.replicaCount` and passes a binary-specific entrypoint command to `/entrypoint.sh`. | Health probes hit `/healthz/live` and `/healthz/ready` on port `telemetry-http` (10000), not the API port (80). The api Deployment additionally exposes port 80 (http). All five Deployments must carry the checksum/config annotation. |
| `jobs.yaml` | Defines three CronJobs: subscription-sync, billing-collect-invoices, billing-advance-invoices. All use `concurrencyPolicy: Forbid` and `restartPolicy: Never`. Schedules come from `Values.jobs.<job>.schedule`. | New CronJobs must set `concurrencyPolicy: Forbid` to prevent overlapping runs. The job command invokes `openmeter-jobs` binary with subcommand args (e.g. `billing advance all`), not a long-running server. checksum/config must appear at both jobTemplate.metadata and template.metadata levels. |
| `svix.yaml` | Conditionally deploys Svix as a Deployment + Service, gated on `Values.svix.enabled`. Svix config is injected via environment variables (SVIX_REDIS_DSN, SVIX_DB_DSN, SVIX_JWT_SECRET), not the shared ConfigMap. | Svix uses its own image (`Values.svix.image`) and its own resources field (`Values.svix.resources`), not the shared `Values.image` or `Values.resources`. Init containers check postgres and redis readiness, not clickhouse. |
| `clickhouse.yaml` | Deploys a ClickHouseInstallation CRD (Altinity operator) only when `Values.clickhouse.enabled`. Requires the clickhouse-operator to be installed in the cluster. | This is a custom CRD (`clickhouse.altinity.com/v1`), not a standard Deployment — helm lint will fail without the CRD present. Storage sizes (3Gi data, 1Gi log) are hardcoded in volumeClaimTemplates, not values-driven. |
| `ingress.yaml` | Standard Kubernetes Ingress for the api Service only. Supports multi-Kube-version API version selection via `semverCompare` (>=1.19-0 uses networking.k8s.io/v1, >=1.14-0 uses v1beta1, otherwise extensions/v1beta1). | Ingress always targets the api component Service via `openmeter.componentName` with "api". Only workers have no Ingress. Only emitted when `Values.ingress.enabled`. |

## Anti-Patterns

- Writing app.kubernetes.io/* labels inline instead of using openmeter.componentLabels / openmeter.componentSelectorLabels helpers
- Hardcoding resource names with .Release.Name directly instead of openmeter.componentName or openmeter.fullname
- Adding a new Deployment or CronJob without a checksum/config annotation on the pod template — config changes won't trigger rollouts
- Adding optional infrastructure without both an `{{- if .Values.<service>.enabled -}}` guard in the resource file AND a corresponding override block in configmap.yaml's helmValuesConfig
- Using different image fields per OpenMeter binary — all binaries share Values.image.repository and Values.image.tag; only Svix is exempt with Values.svix.image

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
