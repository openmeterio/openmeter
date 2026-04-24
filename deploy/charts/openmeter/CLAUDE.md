# openmeter

<!-- archie:ai-start -->

> Helm chart deploying all OpenMeter binaries (api, balance-worker, billing-worker, sink-worker, notification-service) plus optional bundled dependencies (Kafka, ClickHouse, PostgreSQL, Redis, Svix) as Kubernetes workloads sharing a single ConfigMap-mounted config.yaml and a common image tag.

## Patterns

**Component names via openmeter.componentName helper** — Every resource name must be produced by `openmeter.componentName` (or `openmeter.fullname`) from _helpers.tpl. Hardcoding `.Release.Name` directly breaks multi-release isolation. (`name: {{ include "openmeter.componentName" (list . "api") }}`)
**Labels via componentLabels / componentSelectorLabels** — All label blocks on Deployments and Services must use `openmeter.componentLabels` and `openmeter.componentSelectorLabels` helpers so label selectors stay consistent. (`labels: {{- include "openmeter.componentLabels" (list . "sink-worker") | nindent 4 }}`)
**Checksum/config annotation on every workload pod template** — Each Deployment's pod template must carry `checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}` so config changes trigger rollouts. (`annotations:
  checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}`)
**Single shared image tag across all OpenMeter binaries** — All Deployment containers use `Values.image.repository` and `Values.image.tag` (defaulting to `.Chart.AppVersion`). Never introduce per-component image overrides — all binaries ship in the same image. (`image: {{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}`)
**Optional infrastructure wrapped in enabled guards** — Resources for bundled dependencies (Svix, ClickHouse, Kafka, Redis, PostgreSQL) must be wrapped in `{{- if .Values.<service>.enabled }}`. The configmap.yaml must also guard its override block for the same flag. (`{{- if .Values.svix.enabled }}
...
{{- end }}`)
**Config via single ConfigMap volume mounted at /etc/openmeter** — All workloads mount the same ConfigMap at /etc/openmeter/config.yaml. Per-component configuration is expressed through the shared config block; no per-Deployment ConfigMaps should be created. (`volumeMounts:
  - name: config
    mountPath: /etc/openmeter`)
**CronJobs use Forbid concurrency and Never restart policy** — All CronJob entries in jobs.yaml must set `concurrencyPolicy: Forbid` and `restartPolicy: Never` to prevent duplicate billing runs. (`concurrencyPolicy: Forbid
jobTemplate:
  spec:
    template:
      spec:
        restartPolicy: Never`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `templates/_helpers.tpl` | Defines openmeter.fullname, openmeter.componentName, openmeter.componentLabels, openmeter.componentSelectorLabels, and init-container templates. Central authority for naming and labeling. | Init container logic is defined as named templates here, not inline in deployment.yaml — extend here when adding new init containers. |
| `templates/configmap.yaml` | Merges .Values.config with auto-derived connection strings for Kafka, ClickHouse, Redis, PostgreSQL, and Svix based on enabled flags. Single source of runtime config for all workloads. | Values defined in .Values.config are overwritten by chart-derived values; document this in values.yaml comments when adding new override blocks. |
| `templates/deployment.yaml` | Renders Deployments for api, balance-worker, billing-worker, sink-worker, notification-service. Each component gets its own replicaCount from Values.<component>.replicaCount. | Each Deployment must have the checksum/config annotation; all use the same image tag; binary entrypoint differs per component. |
| `templates/jobs.yaml` | Defines CronJobs for subscriptionSync, billingCollectInvoices, and billingAdvanceInvoices. All use the shared config volume and same image. | Schedules come from .Values.jobs.<job>.schedule; must use concurrencyPolicy: Forbid to prevent parallel billing runs. |
| `templates/svix.yaml` | Renders Svix server Deployment and Service, guarded by .Values.svix.enabled. Injects signing secret and DSN from Values. | Uses a separate image (.Values.svix.image) — Svix is the only dependency allowed to have its own image field. |
| `values.yaml` | Defines replicaCount per component, all infrastructure enabled/disabled toggles, job schedules, and the config passthrough block. | config block is a passthrough — chart-derived values overwrite it at render time; warn users with a comment. |
| `Chart.yaml` | Declares Bitnami Kafka, PostgreSQL, Redis, and Altinity ClickHouse operator as conditional subchart dependencies. | All subchart conditions use `.Values.<name>.enabled`; new subcharts must follow the same pattern and update Chart.lock. |

## Anti-Patterns

- Writing app.kubernetes.io/* labels inline instead of using openmeter.componentLabels / openmeter.componentSelectorLabels helpers
- Hardcoding resource names with .Release.Name directly instead of openmeter.componentName or openmeter.fullname
- Adding a new Deployment without a checksum/config annotation — config changes won't trigger rollouts
- Adding optional infrastructure (new self-hosted dependency) without an `{{- if .Values.<service>.enabled -}}` guard in both the resource file and configmap.yaml override block
- Using different image fields per OpenMeter component — all OpenMeter binaries share Values.image.repository and Values.image.tag

## Decisions

- **Single shared ConfigMap for all workloads, merged at render time** — All OpenMeter binaries read the same config.yaml format; a single ConfigMap avoids config drift between components and lets the checksum annotation trigger all workloads on any config change.
- **CronJobs use concurrencyPolicy: Forbid** — Billing and subscription-sync jobs must not run concurrently — duplicate runs would produce double-billing or race conditions in invoice advancement.
- **All OpenMeter binaries share one image and tag** — The mono-repo produces a single Docker image containing all binaries; per-component image overrides would allow version skew between components that share domain package types.

## Example: Adding a new OpenMeter binary worker Deployment

```
# In deployment.yaml, follow existing component block:
- name: {{ include "openmeter.componentName" (list . "my-worker") }}
  replicas: {{ .Values.myWorker.replicaCount }}
  template:
    metadata:
      annotations:
        checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}
      labels: {{- include "openmeter.componentLabels" (list . "my-worker") | nindent 8 }}
    spec:
      containers:
        - image: {{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}
          args: ["my-worker"]
          volumeMounts:
            - name: config
              mountPath: /etc/openmeter
// ...
```

<!-- archie:ai-end -->
