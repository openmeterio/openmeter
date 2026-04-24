# templates

<!-- archie:ai-start -->

> Helm templates that render all Kubernetes resources for the benthos-collector StatefulSet; owns the full resource set (StatefulSet, Service, RBAC, ServiceAccount, Secret, ConfigMap) and enforces the naming, config-injection, and leader-election wiring conventions for the collector binary.

## Patterns

**All names go through _helpers.tpl macros** — Every resource name uses `include "benthos-collector.fullname"` or `include "benthos-collector.componentName"` — never raw `.Release.Name` or `.Chart.Name` concatenation. (`name: {{ include "benthos-collector.componentName" (list . "config") }}`)
**All labels use the common label helpers** — Every `metadata.labels` block calls `include "benthos-collector.labels" . | nindent 4`; selector labels use `include "benthos-collector.selectorLabels" . | nindent N`. (`labels:
  {{- include "benthos-collector.labels" . | nindent 4 }}`)
**Config is stored in a Secret, not a ConfigMap** — The Benthos config.yaml is base64-encoded into a Secret (name: `<fullname>-config`) and mounted at `/etc/benthos/config.yaml` as a read-only subPath volume; never stored in a ConfigMap. (`data:
  config.yaml: {{ .Values.config | toYaml | b64enc | quote }}`)
**benthos-collector.args helper resolves startup mode** — Container args are always emitted via `include "benthos-collector.args" .`; the helper validates that exactly one of `config`, `configFile`, or `preset` is set and fails the render otherwise. (`args: {{ include "benthos-collector.args" . }}`)
**Downward-API K8s metadata injected as env vars** — The StatefulSet always injects K8S_POD_NAME, K8S_POD_UID, K8S_NAMESPACE, K8S_APP_INSTANCE, K8S_APP_NAME, K8S_APP_VERSION via fieldRef so the benthos binary can self-identify for leader election. (`- name: K8S_POD_NAME
  valueFrom:
    fieldRef:
      fieldPath: metadata.name`)
**Credentials live in a mandatory Secret rendered unconditionally** — secret.yaml always renders two Secrets: one for OPENMETER_URL/OPENMETER_TOKEN (required .Values.openmeter.url fails if absent) and one for the Benthos config; both are referenced by the StatefulSet via `envFrom.secretRef` and volume. (`OPENMETER_URL: {{ required "OpenMeter URL is required" .Values.openmeter.url | b64enc | quote }}`)
**Config change triggers pod roll via checksum annotation** — The StatefulSet pod template carries `checksum/config` and `checksum/secret` annotations computed from `.Values.config | toYaml | sha256sum` and `.Values.openmeter | toYaml | sha256sum` so pods roll on config changes. (`checksum/config: {{ .Values.config | toYaml | sha256sum }}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `_helpers.tpl` | Defines all named template functions: name, fullname, chart, labels, selectorLabels, serviceAccountName, componentName (two-arg), storageClass, args. Single source of truth for naming logic. | `benthos-collector.componentName` takes a list `(list . "component")` not a dict — calling it with wrong arg type causes a cryptic nil-pointer in render. |
| `statefulset.yaml` | Core workload; uses StatefulSet (not Deployment) to support persistent storage and stable pod names for leader election; mounts config Secret as subPath, injects Downward API env, and conditionally adds PVC volumeClaimTemplates. | Storage is controlled by `.Values.storage.enabled`; if omitted, the `data` volumeMount and volumeClaimTemplates are skipped entirely — adding a volume without gating on this flag will break renders when storage is disabled. |
| `secret.yaml` | Always renders two Secrets unconditionally; the config Secret holds the full Benthos YAML. Changing config Secret name breaks the volume reference in statefulset.yaml. | The config Secret name is computed via `benthos-collector.componentName (list . "config")` — if this macro changes, the volumeMount in statefulset.yaml must be updated in sync. |
| `rbac.yaml` | Conditionally (`.Values.rbac.create`) creates ClusterRole+Binding (for pod/node/PVC reads) plus a namespace-scoped Role+Binding for lease management (leader election). | Contains a duplicate `subjects:` key in the ClusterRoleBinding block — this is an existing YAML bug that some parsers accept; do not replicate the pattern in new resources. |
| `configmap.yaml` | Only renders when `.Values.caRootCertificates` is non-empty; stores CA certs with names derived from the map key + `.crt` suffix, lowercased. | If CA cert names in the ConfigMap change, the corresponding volumeMount path in statefulset.yaml (`/usr/local/share/ca-certificates`) must still match. |

## Anti-Patterns

- Hardcoding release or chart name strings instead of using `benthos-collector.fullname` / `benthos-collector.componentName`
- Storing the Benthos config.yaml in a ConfigMap — it is always a Secret to protect potential credentials embedded in the config
- Adding container args directly to statefulset.yaml instead of routing them through the `benthos-collector.args` helper, which validates config source exclusivity
- Adding new env vars without checking whether they conflict with the mandatory Downward-API set (K8S_POD_NAME, K8S_NAMESPACE, etc.) injected unconditionally
- Using a Deployment instead of StatefulSet — the workload requires stable pod identity for leader election and optionally persistent storage via volumeClaimTemplates

## Decisions

- **StatefulSet over Deployment** — Stable pod names (K8S_POD_NAME from Downward API) are required for the leader-election lease mechanism; PVC volumeClaimTemplates also require StatefulSet.
- **Config stored in Secret, not ConfigMap** — The Benthos config can contain sensitive credentials (API keys, tokens); Secret keeps them base64-encoded and subject to RBAC, unlike a ConfigMap.
- **Checksum annotations on pod template** — StatefulSet does not automatically roll pods on Secret/ConfigMap changes; sha256sum annotations force a pod roll when config or openmeter credentials change.

## Example: Adding a new named template helper and using it in a resource

```
{{/* In _helpers.tpl */}}
{{- define "benthos-collector.myHelper" -}}
{{- printf "%s-%s" (include "benthos-collector.fullname" .) "mycomponent" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* In a new resource template */}}
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "benthos-collector.myHelper" . }}
  labels:
    {{- include "benthos-collector.labels" . | nindent 4 }}
```

<!-- archie:ai-end -->
