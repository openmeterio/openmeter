{{- define "chart.versionBadge" -}}
![version: {{ .Version }}](https://img.shields.io/badge/version-{{ .Version | replace "-" "--" }}-informational?style=flat-square)
{{- end -}}

{{- define "chart.typeBadge" -}}
{{- if .Type -}}![type: {{ .Type }}](https://img.shields.io/badge/type-{{ .Type }}-informational?style=flat-square){{- end -}}
{{- end -}}

{{- define "chart.appVersionBadge" -}}
{{- if .AppVersion -}}![app version: {{ .AppVersion }}](https://img.shields.io/badge/app%20version-{{ .AppVersion | replace "-" "--" }}-informational?style=flat-square){{- end -}}
{{- end -}}

{{- define "chart.kubeVersionBadge" -}}
{{- if .KubeVersion -}}![kube version: {{ .KubeVersion }}](https://img.shields.io/badge/kube%20version-{{ .KubeVersion | replace "-" "--" }}-informational?style=flat-square){{- end -}}
{{- end -}}

{{- define "chart.artifactHubBadge" -}}
[![artifact hub](https://img.shields.io/badge/artifact%20hub-{{ .Name | replace "-" "--" }}-informational?style=flat-square)](https://artifacthub.io/packages/helm/openmeter/{{ .Name }})
{{- end -}}

{{- define "tldr" -}}
## TL;DR;

```bash
helm install --generate-name --wait oci://ghcr.io/openmeterio/helm-charts/{{ .Name }}
```

to install a specific version:

```bash
helm install --generate-name --wait oci://ghcr.io/openmeterio/helm-charts/{{ .Name }} --version $VERSION
```
{{- end -}}

{{- define "chart.badges" -}}
{{ template "chart.typeBadge" . }} {{ template "chart.kubeVersionBadge" . }} {{ template "chart.artifactHubBadge" . }}
{{- end -}}

{{- define "chart.baseHead" -}}
{{ template "chart.header" . }}

{{ template "chart.badges" . }}

{{ template "chart.description" . }}

{{ template "chart.homepageLine" . }}

{{ template "chart.requirementsSection" . }}

{{ template "tldr" . }}
{{- end -}}

{{- define "chart.base" -}}
{{ template "chart.baseHead" . }}

{{ template "chart.valuesSection" . }}
{{- end -}}
