# benthos-collector

<!-- archie:ai-start -->

> Helm chart packaging the benthos-collector binary as a Kubernetes StatefulSet. Primary constraints: stable pod identity for leader election (StatefulSet), credentials protected in Secrets not ConfigMaps, and config-source exclusivity enforced by the benthos-collector.args helper in _helpers.tpl.

## Patterns

**All resource names via helpers** — Every resource name must use `benthos-collector.fullname` or `benthos-collector.componentName` from templates/_helpers.tpl. Never string-concat `.Release.Name` directly. (`name: {{ include "benthos-collector.fullname" . }}`)
**Config stored in Secret not ConfigMap** — The Benthos config.yaml is always rendered into templates/secret.yaml as a base64-encoded Secret to protect embedded credentials. A parallel ConfigMap for the same data is forbidden. (`kind: Secret
data:
  config.yaml: {{ .Values.config | toYaml | b64enc }}`)
**Args routed through benthos-collector.args helper only** — Container startup args must come exclusively from the `benthos-collector.args` helper defined in _helpers.tpl, which validates config-source exclusivity (config vs configFile vs preset). Never add args inline to statefulset.yaml. (`args: {{- include "benthos-collector.args" . | nindent 12 }}`)
**Checksum annotation triggers pod rollout on config change** — The pod template must carry a `checksum/config` annotation derived from secret.yaml so StatefulSet pods roll when the config Secret changes. (`annotations:
  checksum/config: {{ include (print $.Template.BasePath "/secret.yaml") . | sha256sum }}`)
**Downward-API env vars injected unconditionally** — K8S_POD_NAME, K8S_POD_UID, and K8S_NAMESPACE are injected from the Downward API in statefulset.yaml unconditionally. New env vars must not conflict with these reserved names. (`- name: K8S_POD_NAME
  valueFrom:
    fieldRef:
      fieldPath: metadata.name`)
**RBAC gated on rbac.create** — Role and RoleBinding in templates/rbac.yaml are wrapped in `{{- if .Values.rbac.create }}` so operators managing RBAC externally can opt out. (`{{- if .Values.rbac.create }}
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
{{- end }}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `templates/_helpers.tpl` | Defines all naming, label, and args helpers. Source of truth for benthos-collector.fullname, benthos-collector.args, and common label blocks. | benthos-collector.args validates config-source exclusivity (config > configFile > preset); adding a new config source requires updating this helper, not statefulset.yaml. |
| `templates/statefulset.yaml` | Core workload. Mounts the Secret as a volume, injects Downward-API env vars, and references the args helper. | Must keep checksum/config annotation; must not inline args; must not reference ConfigMap for config data. |
| `templates/secret.yaml` | Renders the Benthos config.yaml as a base64-encoded Secret. Always rendered even when config is empty. | This is the only acceptable storage for Benthos config — do not create a parallel ConfigMap. |
| `templates/rbac.yaml` | Creates Role and RoleBinding for leader election; guarded by .Values.rbac.create. | Leader election requires lease/get/update permissions on coordination.k8s.io; missing rules fail silently at runtime. |
| `values.yaml` | Defines the three mutually exclusive config sources (config, configFile, preset) and leaderElection, storage, and RBAC toggles. | config takes precedence over configFile which takes precedence over preset — precedence is enforced in benthos-collector.args, not values.yaml itself. |

## Anti-Patterns

- Hardcoding release or chart name strings instead of using `benthos-collector.fullname` / `benthos-collector.componentName`
- Storing the Benthos config.yaml in a ConfigMap — it must always be a Secret to protect embedded credentials
- Adding container args directly in statefulset.yaml instead of routing through the `benthos-collector.args` helper in _helpers.tpl
- Adding new env vars without checking for conflicts with the mandatory Downward-API set (K8S_POD_NAME, K8S_NAMESPACE, K8S_POD_UID)
- Using a Deployment instead of StatefulSet — stable pod identity is required for leader election and per-pod PVC volumeClaimTemplates

## Decisions

- **StatefulSet over Deployment** — Leader election requires stable pod identity (hostname-based); volumeClaimTemplates also needs StatefulSet for per-pod persistent storage when storage.enabled=true.
- **Config stored in Secret, not ConfigMap** — The Benthos config may embed API tokens or credentials; Secret provides base64 encoding and Kubernetes RBAC access-control semantics that ConfigMap does not.
- **Checksum annotation on pod template** — StatefulSets do not automatically restart on Secret changes; the sha256sum annotation on the pod template forces a rollout whenever the config Secret content changes.

## Example: Adding a new optional feature toggle that injects env vars and CLI args

```
# In statefulset.yaml env section:
{{- if .Values.myFeature.enabled }}
- name: MY_FEATURE_URL
  value: {{ .Values.myFeature.url | quote }}
{{- end }}

# In _helpers.tpl, extend benthos-collector.args to pass --my-feature-flag
# when .Values.myFeature.enabled is true — keep all arg logic in the helper.
```

<!-- archie:ai-end -->
