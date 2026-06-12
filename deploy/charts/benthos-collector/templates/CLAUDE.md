# templates

<!-- archie:ai-start -->

> Helm chart templates for the benthos-collector sidecar — a Benthos-based usage collector that ships events to OpenMeter. Renders the full Kubernetes object set (StatefulSet, Service, RBAC, Secrets, ServiceAccount, ConfigMap) from values.yaml.

## Patterns

**Named-template naming via _helpers.tpl** — All names/labels are produced by `benthos-collector.*` defines (name, fullname, chart, labels, selectorLabels, serviceAccountName, componentName, storageClass, args). Object templates must call these through `include`, never hardcode names. (`name: {{ include "benthos-collector.fullname" . }}`)
**Standard label block on every metadata** — Every resource attaches `{{- include "benthos-collector.labels" . | nindent 4 }}` under metadata.labels; selectors use the narrower `benthos-collector.selectorLabels`. Adding a resource without the labels include breaks helm-managed-by/version conventions. (`labels:
  {{- include "benthos-collector.labels" . | nindent 4 }}`)
**Conditional resources gated on .Values toggles** — Resources guard their entire body with feature flags: `service.enabled`, `serviceAccount.create`, `rbac.create`, `len .Values.caRootCertificates`, `storage.enabled`. New optional resources must follow the same `{{- if ... -}} ... {{- end }}` wrapping. (`{{- if .Values.service.enabled }}
...
{{- end }}`)
**Config delivered via mutually-exclusive args resolver** — `benthos-collector.args` requires exactly one of `config`, `configFile`, or `preset` and calls `fail` otherwise. Presets are an explicit allowlist (`http-server`, `kubernetes-pod-exec-time`). New presets must be added to this template's if/else chain. (`args: {{ include "benthos-collector.args" . }}`)
**Checksum-triggered pod rollout** — statefulset.yaml stamps `checksum/config` and `checksum/secret` from `sha256sum` of the rendered values so config/secret changes force a pod restart. Preserve these annotations when editing the pod template. (`checksum/config: {{ .Values.config | toYaml | sha256sum }}`)
**Component-scoped secondary names** — The config Secret and its volume mount use `benthos-collector.componentName` (list . "config") to derive a DNS-safe suffixed name truncated to fit 63 chars. Use this helper for any per-component object, not raw printf. (`name: {{ include "benthos-collector.componentName" (list . "config") }}`)
**Secret-backed env and config volume** — OPENMETER_URL/OPENMETER_TOKEN are base64'd into the main Secret and injected via `envFrom.secretRef`; benthos config.yaml is base64'd into the config Secret and mounted read-only at /etc/benthos/config.yaml. Sensitive values flow through Secrets, never plain env. (`envFrom:
  - secretRef:
      name: {{ include "benthos-collector.fullname" . }}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `_helpers.tpl` | Defines all reusable named templates (naming, labels, serviceAccountName, componentName, storageClass, args). Single source of truth for names. | `componentName` truncates fullname to `62 - len(component)`; long release names silently lose characters. `args` and `storageClass` embed business logic (preset allowlist, storageClassName rendering) — edit here, not in object templates. |
| `statefulset.yaml` | Primary workload. Defines container command `/entrypoint.sh /usr/local/bin/benthos`, metrics port 4195, optional http port 8080, leader-election env, volumes, and optional volumeClaimTemplates. | livenessProbe/readinessProbe hit /ping and /ready on the `metrics` port (4195), not the http port. http port only exists when `service.enabled`. Removing checksum annotations disables auto-rollout on config change. |
| `secret.yaml` | Two Secrets: main (OPENMETER_URL required, OPENMETER_TOKEN optional) and a `config`-componentName Secret holding base64 benthos config.yaml. | OPENMETER_URL uses `required` and fails render if unset. config.yaml is `toYaml | b64enc`; do not double-encode. Not gated by any toggle — always rendered. |
| `rbac.yaml` | Gated on `rbac.create`. ClusterRole+Binding for pods/nodes/PVCs/PVs (watch/list/get) and namespaced Role+Binding for coordination.k8s.io leases (for leader election). | Contains a duplicated `subjects:` line in the ClusterRoleBinding block (cosmetic/redundant). Subjects bind to `benthos-collector.serviceAccountName` in `.Release.Namespace`. |
| `service.yaml` | Gated on `service.enabled`. Exposes the `http` targetPort (container 8080) on `service.port`. | Selector uses selectorLabels only. If enabled but the statefulset http port is absent (also gated on service.enabled) the service has no backend — keep both flags consistent. |
| `configmap.yaml` | Renders a `ca-certificates` ConfigMap only when `caRootCertificates` is non-empty; keys are `<name>.crt` lowercased. | ConfigMap name is the literal `ca-certificates` (not fullname-scoped) — collides across releases in the same namespace. Mounted read-only at /usr/local/share/ca-certificates in the statefulset. |
| `serviceaccount.yaml` | Gated on `serviceAccount.create`. Sets automountServiceAccountToken from `serviceAccount.automount` and supports annotations. | When create is false, statefulset still references `serviceAccountName` (default fallback) — ensure the named SA exists externally. |

## Anti-Patterns

- Hardcoding object names or label blocks instead of calling the `benthos-collector.*` helpers — breaks naming/truncation and managed-by conventions.
- Adding a benthos config source without extending `benthos-collector.args`; the template `fail`s unless exactly one of config/configFile/preset is set.
- Putting OPENMETER_TOKEN or config.yaml into plain env/ConfigMap instead of the existing Secrets.
- Removing the `checksum/config` / `checksum/secret` annotations, which silently stops pods from rolling on config changes.
- Probing the http port for liveness/readiness — health endpoints (/ping, /ready) live on the metrics port 4195.

## Decisions

- **Deployed as a StatefulSet (not Deployment) with optional volumeClaimTemplates.** — Benthos needs stable identity and optional persistent storage (e.g. buffer/state) gated on `storage.enabled`, plus leader election via coordination leases.
- **Config source is resolved by a single `args` helper enforcing exactly one of config/configFile/preset.** — Prevents ambiguous or empty config and provides a curated preset allowlist for common collection scenarios.

## Example: Adding a new optional Kubernetes resource to the chart

```
{{- if .Values.myFeature.enabled }}
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "benthos-collector.fullname" . }}
  labels:
    {{- include "benthos-collector.labels" . | nindent 4 }}
data:
  ...
{{- end }}
```

<!-- archie:ai-end -->
