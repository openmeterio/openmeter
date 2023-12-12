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
[![artifact hub](https://img.shields.io/badge/artifact%20hub-{{ .Name | replace "-" "--" }}-informational?style=flat-square)](https://artifacthub.io/packages/helm/sagikazarmark/{{ .Name }})
{{- end -}}

{{- define "tldr" -}}
## TL;DR;

```bash
helm repo add skm https://charts.sagikazarmark.dev
helm install --generate-name --wait skm/{{ .Name }}
```
{{- end -}}

{{- define "chart.badges" -}}
{{ template "chart.versionBadge" . }} {{ template "chart.typeBadge" . }} {{ template "chart.appVersionBadge" . }} {{ template "chart.kubeVersionBadge" . }} {{ template "chart.artifactHubBadge" . }}
{{- end -}}

{{- define "chart.baseHead" -}}
{{ template "chart.header" . }}

{{ template "chart.badges" . }}

{{ template "chart.description" . }}

{{ template "chart.homepageLine" . }}

{{ template "tldr" . }}
{{- end -}}

{{- define "chart.base" -}}
{{ template "chart.baseHead" . }}

{{ template "chart.valuesSection" . }}
{{- end -}}
