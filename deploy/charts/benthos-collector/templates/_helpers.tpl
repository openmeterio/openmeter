{{/*
Expand the name of the chart.
*/}}
{{- define "benthos-collector.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "benthos-collector.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "benthos-collector.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "benthos-collector.labels" -}}
helm.sh/chart: {{ include "benthos-collector.chart" . }}
{{ include "benthos-collector.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "benthos-collector.selectorLabels" -}}
app.kubernetes.io/name: {{ include "benthos-collector.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "benthos-collector.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "benthos-collector.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create a default fully qualified component name from the full app name and a component name.
We truncate the full name at 63 - 1 (last dash) - len(component name) chars because some Kubernetes name fields are limited to this (by the DNS naming spec)
and we want to make sure that the component is included in the name.

Usage: {{ include "benthos-collector.componentName" (list . "component") }}
*/}}
{{- define "benthos-collector.componentName" -}}
{{- $global := index . 0 -}}
{{- $component := index . 1 | trimPrefix "-" -}}
{{- printf "%s-%s" (include "benthos-collector.fullname" $global | trunc (sub 62 (len $component) | int) | trimSuffix "-" ) $component | trimSuffix "-" -}}
{{- end -}}

{{/*
Create args for the deployment
*/}}
{{- define "benthos-collector.args" -}}
{{- if .Values.config -}}
["benthos", "-c", "/etc/benthos/config.yaml"]
{{- else if .Values.configFile -}}
["benthos", "-c", "{{ .Values.configFile }}"]
{{- else if .Values.preset }}
{{- if eq .Values.preset "http-server" -}}
["benthos", "streams", "--no-api", "/etc/benthos/presets/http-server/input.yaml", "/etc/benthos/presets/http-server/output.yaml"]
{{- else if eq .Values.preset "kubernetes-pod-exec-time" -}}
["benthos", "-c", "/etc/benthos/presets/kubernetes-pod-exec-time/config.yaml"]
{{- else }}
{{- fail (printf "Invalid example '%s" .Values.preset) }}
{{- end }}
{{- else }}
{{- fail "One of 'config', 'configFile' or 'preset' is required" }}
{{- end }}
{{- end }}
