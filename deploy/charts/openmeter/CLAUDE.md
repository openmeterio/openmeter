# openmeter

<!-- archie:ai-start -->

> Helm chart deploying all OpenMeter binaries (api, balance-worker, billing-worker, sink-worker, notification-service) plus optional bundled dependencies (Kafka, ClickHouse, PostgreSQL, Redis, Svix) as Kubernetes workloads. Chart metadata/values live at the root; templates/ renders the workloads. Primary constraint: all binaries share one ConfigMap-mounted config.yaml and a single image tag to prevent version skew across the mono-repo image.

## Patterns

**Single shared image + tag for all OpenMeter binaries** — Every workload uses Values.image.repository and Values.image.tag (default .Chart.AppVersion); only Svix has its own Values.svix.image. Per-component image overrides are forbidden. (`image: {{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}`)
**Optional dependencies gated by conditional subcharts + .enabled flags** — Chart.yaml declares Bitnami Kafka/PostgreSQL/Redis and the Altinity ClickHouse operator as conditional dependencies (condition: <name>.enabled / clickhouse.operator.install); each must also be guarded in resource templates and configmap.yaml. (`dependencies: - name: kafka, condition: kafka.enabled`)
**Single config passthrough overwritten by chart-derived values** — Values.config is a passthrough merged in configmap.yaml; chart-derived connection strings (from enabled flags) overwrite matching keys at render time — documented in values.yaml comments. (`# Values defined in `config` will get overwritten by the values calculated from chart values!`)
**Per-component replicaCount** — Each binary has its own <component>.replicaCount (api, balanceWorker, billingWorker, sinkWorker, notificationService); workloads otherwise stay homogeneous. (`billingWorker: { replicaCount: 1 }`)
**CronJobs for billing/sync with fixed schedules** — values.yaml jobs.* (subscriptionSync, billingCollectInvoices, billingAdvanceInvoices) define cron schedules; the templates use concurrencyPolicy: Forbid to prevent duplicate billing runs. (`jobs: { subscriptionSync: { schedule: "0 * * * *" } }`)
**Bitnami legacy image-repo overrides** — Bitnami moved free images to bitnamilegacy/*; the kafka/redis/postgresql subchart blocks override every nested image.repository (kafka, kubectl, jmx-exporter, redis-sentinel, os-shell, etc.) to bitnamilegacy/*. (`kafka: { image: { repository: bitnamilegacy/kafka } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `Chart.yaml` | Declares altinity-clickhouse-operator, Bitnami kafka/postgresql/redis as conditional subchart dependencies; version is the chart release version. | New subcharts must follow the .Values.<name>.enabled condition pattern and update Chart.lock. |
| `values.yaml` | Per-component replicaCount, all dependency enable/disable toggles, job cron schedules, Svix config, bitnamilegacy image overrides, and the config passthrough block. | config is a passthrough — chart-derived values overwrite matching keys at render; warn users in comments when adding new overridable keys. |
| `values.example.yaml` | Reference config showing meters, events/notification/entitlements feature flags. | Keep in sync with the config schema in app/config; it is documentation, not rendered. |
| `README.md / README.tmpl.md` | README.md is generated from README.tmpl.md via helm-docs (chart.baseHead + chart.valuesSection). | Edit README.tmpl.md and values.yaml comments, never README.md. |

## Anti-Patterns

- Using different image fields per OpenMeter binary — all share Values.image; only Svix is exempt with Values.svix.image
- Adding optional infra without both a {{- if .Values.<service>.enabled }} guard in the resource AND an override block in configmap.yaml
- Hardcoding resource names with .Release.Name instead of openmeter.componentName/fullname helpers
- Adding a Deployment/CronJob without a checksum/config annotation — config changes won't trigger rollouts
- Pinning Bitnami images to docker.io/bitnami/* instead of bitnamilegacy/* — those tags are no longer published

## Decisions

- **Single shared ConfigMap and image/tag for all workloads** — All binaries ship in one mono-repo Docker image and read the same config.yaml; a shared ConfigMap + checksum annotation rolls all workloads on any config change and prevents version skew between components sharing compiled domain types.
- **Bundled dependencies are conditional subcharts disabled for production** — Kafka/ClickHouse/PostgreSQL/Redis/Svix are bundled for dev convenience but documented as not-for-production; operators disable them and provide connection details via config.
- **Billing/sync CronJobs use concurrencyPolicy: Forbid** — Duplicate concurrent billing or subscription-sync runs would cause double-billing or invoice-advancement races.

<!-- archie:ai-end -->
