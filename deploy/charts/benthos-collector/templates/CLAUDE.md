# templates

<!-- archie:ai-start -->

> Helm templates rendering the complete Kubernetes resource set (StatefulSet, Service, RBAC, ServiceAccount, Secret, ConfigMap) for the benthos-collector binary; enforces naming via _helpers.tpl macros, stores Benthos config in a Secret, and wires Downward-API env vars for leader-election.

## Patterns

**All names through _helpers.tpl macros** — Resource names use include "benthos-collector.fullname" or componentName (list . "component"); never concatenate .Release.Name/.Chart.Name directly. (`name: {{ include "benthos-collector.componentName" (list . "config") }}`)
**Common label helpers on every resource** — Every metadata.labels calls include "benthos-collector.labels" . | nindent 4; selectors use selectorLabels. (`labels:
  {{- include "benthos-collector.labels" . | nindent 4 }}`)
**Config stored in Secret, not ConfigMap** — config.yaml is base64-encoded into a Secret <fullname>-config and mounted read-only at /etc/benthos/config.yaml; ConfigMap is only for CA certs. (`data:
  config.yaml: {{ .Values.config | toYaml | b64enc | quote }}`)
**Container args via benthos-collector.args helper only** — Args in statefulset.yaml are emitted via include "benthos-collector.args" .; the helper enforces mutual exclusivity of config/configFile/preset and fails if none is set. (`args: {{ include "benthos-collector.args" . }}`)
**Downward-API env vars injected unconditionally** — The StatefulSet always injects K8S_POD_NAME, K8S_POD_UID, K8S_NAMESPACE, K8S_APP_INSTANCE, K8S_APP_NAME, K8S_APP_VERSION via fieldRef; new env vars must not conflict. (`- name: K8S_POD_NAME
  valueFrom:
    fieldRef:
      fieldPath: metadata.name`)
**Checksum annotations force pod roll on config change** — Pod template carries checksum/config and checksum/secret sha256sum annotations so pods roll when config or credentials change. (`annotations:
  checksum/config: {{ .Values.config | toYaml | sha256sum }}`)
**Storage-gated volume and PVC** — The data volumeMount and volumeClaimTemplates render only when .Values.storage.enabled is true; any new persistent volume follows the same gate. (`{{- if .Values.storage.enabled }}
  volumeClaimTemplates:
    ...
{{- end }}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `_helpers.tpl` | Single source of truth for named template functions: name, fullname, chart, labels, selectorLabels, serviceAccountName, componentName, storageClass, args. | componentName takes a list (list . "component") not a dict — wrong arg type causes nil-pointer render errors; storageClass delegates to common.storage.class. |
| `statefulset.yaml` | Core workload — StatefulSet (not Deployment) for stable pod names and optional PVC volumeClaimTemplates; mounts config Secret as subPath and injects Downward API env. | Adding a volumeMount without gating on .Values.storage.enabled breaks renders when storage is disabled; the config volume references componentName (list . "config"). |
| `secret.yaml` | Always renders two Secrets: one for OPENMETER_URL/OPENMETER_TOKEN (required, fails render if absent), one for the Benthos config YAML. | The config Secret name must stay in sync with the volume reference in statefulset.yaml. |
| `rbac.yaml` | Conditionally (.Values.rbac.create) creates ClusterRole+Binding for pod/node/PVC reads and a namespace Role+Binding for lease management. | Contains a duplicate subjects: key in the ClusterRoleBinding — an existing bug; do not replicate this pattern. |
| `configmap.yaml` | Renders only when .Values.caRootCertificates is non-empty; stores CA certs keyed by name with .crt suffix, lowercased. | If CA cert key names change, the volumeMount path /usr/local/share/ca-certificates in statefulset.yaml must still align. |

## Anti-Patterns

- Hardcoding release or chart name strings instead of fullname/componentName helpers.
- Storing config.yaml in a ConfigMap — it must be a Secret to protect embedded credentials.
- Adding container args directly in statefulset.yaml instead of routing through benthos-collector.args.
- Adding env vars that conflict with the injected Downward-API set (K8S_POD_NAME, etc.).
- Using a Deployment instead of StatefulSet — stable pod identity is required for leader election and PVCs.

## Decisions

- **StatefulSet over Deployment.** — Stable pod names (via K8S_POD_NAME Downward API) are required for the leader-election lease; PVC volumeClaimTemplates also require StatefulSet semantics.
- **Config stored in Secret, not ConfigMap.** — Benthos config can contain credentials; a Secret keeps them base64-encoded and RBAC-scoped, unlike a plaintext ConfigMap.
- **Checksum annotations on the pod template.** — StatefulSet does not auto-roll pods on Secret/ConfigMap changes; sha256sum annotations force a roll when config or credentials change.

## Example: Adding a new named helper and using it in a new resource

```
{{- define "benthos-collector.myHelper" -}}
{{- printf "%s-%s" (include "benthos-collector.fullname" .) "mycomponent" | trunc 63 | trimSuffix "-" }}
{{- end }}

apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "benthos-collector.myHelper" . }}
  labels:
    {{- include "benthos-collector.labels" . | nindent 4 }}
```

<!-- archie:ai-end -->
