# templates

<!-- archie:ai-start -->

> Helm templates deploying all OpenMeter binaries (api, balance-worker, billing-worker, sink-worker, notification-service) plus optional dependencies (Svix, ClickHouse) as Kubernetes workloads. Every workload shares one ConfigMap-mounted config.yaml and the same image tag, enforcing homogeneous deployment across binaries.

## Patterns

**componentName for all resource names** — Resource names use include "openmeter.componentName" (list . "<component>"), never .Release.Name directly; the helper truncates to 63 chars for DNS compliance. (`name: {{ include "openmeter.componentName" (list . "billing-worker") }}`)
**componentLabels / componentSelectorLabels** — metadata.labels use openmeter.componentLabels and selector.matchLabels use openmeter.componentSelectorLabels; never write app.kubernetes.io labels inline. (`labels:
  {{- include "openmeter.componentLabels" (list . "sink-worker") | nindent 4 }}`)
**checksum/config annotation on every pod template** — Every Deployment and CronJob pod template carries checksum/config to force rollout on config changes. (`checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}`)
**initContainers gated on *.enabled via named templates** — Init containers emit only when postgresql.enabled/clickhouse.enabled is true, using openmeter.init.postgres/clickhouse helpers — never inline readiness checks. (`{{- if .Values.postgresql.enabled -}}
{{- include "openmeter.init.postgres" (list .) | nindent 8 }}
{{- end }}`)
**Shared image tag across all OpenMeter Deployments** — All binary containers use Values.image.repository:Values.image.tag (default Chart.AppVersion); never pin a different image per component. Svix is the only exception. (`image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"`)
**Config via ConfigMap volume at /etc/openmeter** — Every workload mounts the openmeter.fullname ConfigMap at /etc/openmeter and passes --config /etc/openmeter/config.yaml; CA certs mount conditionally when caRootCertificates is set. (`volumes:
  - name: config
    configMap:
      name: {{ include "openmeter.fullname" . }}`)
**Optional-dependency resources wrapped in enabled guards** — Optional services (clickhouse.yaml, svix.yaml) wrap their entire content in {{- if .Values.<service>.enabled -}} and also add an override block in configmap.yaml's helmValuesConfig. (`{{- if .Values.svix.enabled -}}
kind: Deployment
...
{{- end }}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `_helpers.tpl` | All named templates: fullname, componentName, labels/selectorLabels, componentLabels/SelectorLabels, serviceAccountName, init-container helpers (netcat, postgres, clickhouse, redis). | componentName uses (list . "component") syntax — a plain string breaks it; name truncates at 62-len(component); init.redis uses a PING loop unlike netcat-based postgres/clickhouse. |
| `configmap.yaml` | config.yaml ConfigMap deep-merging Values.config with auto-overrides via mergeOverwrite; emits ca-certificates ConfigMap when set. | helmValuesConfig uses fromYaml/mergeOverwrite — whitespace matters; a new self-hosted dependency needs an if .Values.<dep>.enabled override block mirroring config.example.yaml. |
| `deployment.yaml` | Five Deployments (api, balance-worker, notification-service, sink-worker, billing-worker), each with its own replicaCount and entrypoint command. | Health probes hit /healthz/live and /healthz/ready on telemetry-http (10000), not the API port (80); all five need checksum/config. |
| `jobs.yaml` | Three CronJobs (subscription-sync, billing-collect-invoices, billing-advance-invoices) with concurrencyPolicy Forbid, restartPolicy Never. | New CronJobs must set concurrencyPolicy Forbid; checksum/config must appear at both jobTemplate.metadata and template.metadata. |
| `svix.yaml` | Svix Deployment + Service gated on svix.enabled; config via env vars, not the shared ConfigMap. | Uses Values.svix.image and Values.svix.resources; init containers check postgres+redis, not clickhouse. |
| `clickhouse.yaml` | ClickHouseInstallation CRD (Altinity operator) only when clickhouse.enabled. | Custom CRD (clickhouse.altinity.com/v1) — helm lint fails without the CRD; storage sizes are hardcoded, not values-driven. |
| `ingress.yaml` | Ingress for the api Service only, with semverCompare-based API version selection. | Always targets the api component Service; emitted only when ingress.enabled; workers have no Ingress. |

## Anti-Patterns

- Writing app.kubernetes.io/* labels inline instead of componentLabels/componentSelectorLabels.
- Hardcoding names with .Release.Name instead of componentName/fullname.
- Adding a Deployment/CronJob without checksum/config — config changes won't trigger rollouts.
- Adding optional infra without both an enabled guard in the resource AND an override block in configmap.yaml.
- Using different image fields per binary — all share Values.image; only Svix is exempt.

## Decisions

- **Single shared ConfigMap merged at render time from Values.config plus per-flag overrides.** — All binaries share one config.yaml schema; mergeOverwrite avoids per-binary ConfigMaps while auto-injecting self-hosted dependency addresses when *.enabled is set.
- **CronJobs use concurrencyPolicy Forbid and restartPolicy Never.** — Billing/sync jobs are not idempotent under concurrency; Forbid prevents overlap, and the next scheduled run handles recovery.
- **Init containers generated from named templates, not inline YAML.** — The same readiness-check logic is reused across all Deployments and CronJobs; centralizing ensures a consistent busybox tag and address derivation.

## Example: Add a new worker Deployment (e.g. charges-worker)

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
```

<!-- archie:ai-end -->
