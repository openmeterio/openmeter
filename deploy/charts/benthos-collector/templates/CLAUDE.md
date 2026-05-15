# templates

<!-- archie:ai-start -->

> Helm templates rendering the complete Kubernetes resource set (StatefulSet, Service, RBAC, ServiceAccount, Secret, ConfigMap) for the benthos-collector binary; enforces naming via _helpers.tpl macros, stores Benthos config in a Secret, and wires Downward-API env vars for leader-election.

## Patterns

**All names through _helpers.tpl macros** — Every resource name must use `include "benthos-collector.fullname"` or `include "benthos-collector.componentName" (list . "component")`. Never concatenate `.Release.Name` or `.Chart.Name` directly. (`name: {{ include "benthos-collector.componentName" (list . "config") }}`)
**Common label helpers on every resource** — Every `metadata.labels` block must call `include "benthos-collector.labels" . | nindent 4`; selector labels use `include "benthos-collector.selectorLabels" . | nindent N`. (`labels:
  {{- include "benthos-collector.labels" . | nindent 4 }}`)
**Config stored in Secret, not ConfigMap** — The Benthos config.yaml is base64-encoded into a Secret named `<fullname>-config` and mounted at `/etc/benthos/config.yaml` as a read-only subPath volume. ConfigMap is only for CA certificates. (`data:
  config.yaml: {{ .Values.config | toYaml | b64enc | quote }}`)
**Container args via benthos-collector.args helper only** — Container args in statefulset.yaml must always be emitted via `include "benthos-collector.args" .`; the helper enforces mutual exclusivity of `config`, `configFile`, and `preset` and calls `fail` if none is set. (`args: {{ include "benthos-collector.args" . }}`)
**Downward-API env vars injected unconditionally** — The StatefulSet always injects K8S_POD_NAME, K8S_POD_UID, K8S_NAMESPACE, K8S_APP_INSTANCE, K8S_APP_NAME, K8S_APP_VERSION via fieldRef. New env vars must not conflict with these names. (`- name: K8S_POD_NAME
  valueFrom:
    fieldRef:
      fieldPath: metadata.name`)
**Checksum annotations force pod roll on config change** — Pod template carries `checksum/config: {{ .Values.config | toYaml | sha256sum }}` and `checksum/secret: {{ .Values.openmeter | toYaml | sha256sum }}` so StatefulSet pods roll when config or credentials change. (`annotations:
  checksum/config: {{ .Values.config | toYaml | sha256sum }}
  checksum/secret: {{ .Values.openmeter | toYaml | sha256sum }}`)
**Storage-gated volume and PVC** — The `data` volumeMount and volumeClaimTemplates are only rendered when `.Values.storage.enabled` is true. Any new persistent volume must follow the same gate. (`{{- if .Values.storage.enabled }}
  volumeClaimTemplates:
    ...
{{- end }}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `_helpers.tpl` | Single source of truth for all named template functions: name, fullname, chart, labels, selectorLabels, serviceAccountName, componentName (two-arg list), storageClass, args. | `benthos-collector.componentName` takes a list `(list . "component")` not a dict — passing a wrong arg type causes a nil-pointer render error. `benthos-collector.storageClass` delegates to inline `common.storage.class` defined in the same file. |
| `statefulset.yaml` | Core workload; StatefulSet (not Deployment) for stable pod names and optional PVC volumeClaimTemplates; mounts config Secret as subPath and injects Downward API env. | Adding a volumeMount without gating on `.Values.storage.enabled` will break renders when storage is disabled. The config volume references the Secret name from `benthos-collector.componentName (list . "config")` — if the macro output changes, the mount breaks. |
| `secret.yaml` | Always renders two Secrets unconditionally: one for OPENMETER_URL/OPENMETER_TOKEN (required field fails render if absent), one for the Benthos config YAML. | The config Secret name `benthos-collector.componentName (list . "config")` must stay in sync with the volume reference in statefulset.yaml. Renaming it in one place without the other breaks the mount. |
| `rbac.yaml` | Conditionally (`.Values.rbac.create`) creates ClusterRole+Binding for pod/node/PVC reads and a namespace Role+Binding for lease management (leader election). | Contains a duplicate `subjects:` key in the ClusterRoleBinding — an existing YAML bug some parsers accept silently. Do not replicate this pattern in new resources. |
| `configmap.yaml` | Only renders when `.Values.caRootCertificates` is non-empty; stores CA certs keyed by name with `.crt` suffix, lowercased. | If CA cert key names change, the volumeMount path `/usr/local/share/ca-certificates` in statefulset.yaml must still align. |

## Anti-Patterns

- Hardcoding release or chart name strings instead of using `benthos-collector.fullname` / `benthos-collector.componentName`
- Storing the Benthos config.yaml in a ConfigMap — it must always be a Secret to protect embedded credentials
- Adding container args directly in statefulset.yaml instead of routing through `benthos-collector.args`, which validates config source exclusivity
- Adding new env vars that conflict with the unconditionally injected Downward-API set (K8S_POD_NAME, K8S_POD_UID, K8S_NAMESPACE, etc.)
- Using a Deployment instead of StatefulSet — stable pod identity is required for leader election and PVC volumeClaimTemplates

## Decisions

- **StatefulSet over Deployment** — Stable pod names (exposed via K8S_POD_NAME Downward API) are required for the leader-election lease mechanism; PVC volumeClaimTemplates also require StatefulSet semantics.
- **Config stored in Secret, not ConfigMap** — The Benthos config can contain sensitive credentials (API keys, tokens); Secret keeps them base64-encoded and subject to RBAC, unlike a plaintext ConfigMap.
- **Checksum annotations on pod template** — StatefulSet does not automatically roll pods on Secret/ConfigMap changes; sha256sum annotations force a pod roll when config or OpenMeter credentials change.

## Example: Adding a new named template helper and using it in a new resource

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
