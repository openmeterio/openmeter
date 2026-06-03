# benthos-collector

<!-- archie:ai-start -->

> Helm chart packaging the benthos-collector binary as a Kubernetes StatefulSet. Chart metadata + values live at the root; the templates/ child renders the actual K8s resources. Primary constraints: stable pod identity for leader election (StatefulSet), credentials protected in a Secret (not ConfigMap), and config-source exclusivity enforced by the benthos-collector.args helper.

## Patterns

**Three mutually-exclusive config sources with fixed precedence** — values.yaml exposes config > configFile > preset; precedence is resolved by the benthos-collector.args helper in templates/_helpers.tpl, never in values.yaml or statefulset.yaml. (`config: {} # takes precedence over configFile and preset`)
**Config rendered into a Secret, never a ConfigMap** — The Benthos config.yaml is base64-encoded into templates/secret.yaml to protect embedded OpenMeter tokens/credentials; a parallel ConfigMap for the same data is forbidden. (`kind: Secret
data:
  config.yaml: {{ .Values.config | toYaml | b64enc }}`)
**All names via helper macros** — Resource names come from benthos-collector.fullname / benthos-collector.componentName; never string-concat .Release.Name. (`name: {{ include "benthos-collector.fullname" . }}`)
**StatefulSet with leader-election + storage toggles** — leaderElection.enabled (RBAC + lease config) and storage.enabled (PVC volumeClaimTemplates) gate optional features; both rely on the StatefulSet's stable pod identity. (`leaderElection: { enabled: false, lease: { duration: 10s } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `Chart.yaml` | Chart metadata; version 0.0.0 and appVersion "latest" are placeholders rewritten at release time. | appVersion should align with the published benthos-collector image tag on release. |
| `values.yaml` | Defines the three config sources (config/configFile/preset), openmeter url/token, leaderElection, storage (PVC), rbac, and standard pod knobs. | config precedence is enforced in the args helper, not here; storage uses a per-pod PVC requiring StatefulSet. |
| `README.md / README.tmpl.md` | README.md is generated from README.tmpl.md via helm-docs; README.tmpl.md only contains {{ template "chart.base" . }}. | Edit README.tmpl.md and values.yaml comments, never README.md directly. |

## Anti-Patterns

- Storing the Benthos config.yaml in a ConfigMap instead of a Secret — leaks embedded credentials
- Hardcoding release/chart name strings instead of the benthos-collector.fullname/componentName helpers
- Bumping appVersion/version without aligning to the released image tag
- Using a Deployment instead of StatefulSet — breaks leader election and per-pod PVCs

## Decisions

- **StatefulSet over Deployment** — Leader election needs stable hostname-based pod identity, and volumeClaimTemplates needs StatefulSet for per-pod persistent storage when storage.enabled.
- **Config stored in a Secret, not ConfigMap** — The Benthos config can embed API tokens; Secret provides base64 encoding and RBAC access-control semantics ConfigMap lacks.

<!-- archie:ai-end -->
