{{/*
Expand the name of the chart.
*/}}
{{- define "openmeter.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "openmeter.fullname" -}}
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
{{- define "openmeter.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "openmeter.labels" -}}
helm.sh/chart: {{ include "openmeter.chart" . }}
{{ include "openmeter.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "openmeter.selectorLabels" -}}
app.kubernetes.io/name: {{ include "openmeter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "openmeter.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "openmeter.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create a default fully qualified component name from the full app name and a component name.
We truncate the full name at 63 - 1 (last dash) - len(component name) chars because some Kubernetes name fields are limited to this (by the DNS naming spec)
and we want to make sure that the component is included in the name.

Usage: {{ include "openmeter.componentName" (list . "component") }}
*/}}
{{- define "openmeter.componentName" -}}
{{- $global := index . 0 -}}
{{- $component := index . 1 | trimPrefix "-" -}}
{{- printf "%s-%s" (include "openmeter.fullname" $global | trunc (sub 62 (len $component) | int) | trimSuffix "-" ) $component | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels with component name

Usage: {{ include "openmeter.componentLabels" (list . "component") }}
*/}}
{{- define "openmeter.componentLabels" -}}
{{- $global := index . 0 -}}
{{- $component := index . 1 | trimPrefix "-" -}}
{{ include "openmeter.labels" $global }}
app.kubernetes.io/component: {{ $component }}
{{- end }}

{{/*
Selector labels with component name

Usage: {{ include "openmeter.componentSelectorLabels" (list . "component") }}
*/}}
{{- define "openmeter.componentSelectorLabels" -}}
{{- $global := index . 0 -}}
{{- $component := index . 1 | trimPrefix "-" -}}
{{ include "openmeter.selectorLabels" $global }}
app.kubernetes.io/component: {{ $component }}
{{- end }}

{{/*
Generic init container with netcat to check if a dependency is ready
*/}}
{{- define "openmeter.init.netcat" -}}
{{- $global := index . 0 -}}
{{- $name := index . 1 -}}
{{- $address := index . 2 -}}
{{- $port := index . 3 -}}
- name: init-{{ $name }}
  image: "busybox:{{ $global.Values.init.busybox.tag }}"
  imagePullPolicy: IfNotPresent
  command:
    - sh
    - -c
    - |
        echo "Waiting for {{ lower $name }}...";
        while ! nc -z {{ $address }} {{ $port }}; do
          sleep 1;
        done;
        echo "{{ title $name }} is ready!";
{{- end }}

{{/*
Checks if the postres port is ready
*/}}
{{- define "openmeter.init.postgres" -}}
{{- $global := index . 0 -}}
{{- $address := printf "%v-postgres" (include "openmeter.fullname" $global) -}}
{{ include "openmeter.init.netcat" (list $global "postgres" $address "5432") }}
{{- end }}

{{/*
Checks if the clickhouse port is ready
*/}}
{{- define "openmeter.init.clickhouse" -}}
{{- $global := index . 0 -}}
{{- $address := printf "clickhouse-%v" (include "openmeter.fullname" $global) -}}
{{ include "openmeter.init.netcat" (list $global "clickhouse" $address "9000") }}
{{- end }}

{{/*
Checks if redis is ready
*/}}
{{- define "openmeter.init.redis" -}}
{{- $global := index . 0 -}}
{{- $address := printf "%v-redis-master" (include "openmeter.fullname" $global) -}}
- name: init-redis
  image: "busybox:{{ $global.Values.init.busybox.tag }}"
  imagePullPolicy: IfNotPresent
  command:
    - sh
    - -c
    - |
        echo "Waiting for redis...";
        while true; do
            response=$(echo -en "PING\r\n" | nc -w 1 {{ $address }} 6379)

            if [[ "$response" == "+PONG*" ]]; then
                echo "Redis is ready!"
                exit 0
            else
                echo "Redis is not ready: $response"
                sleep 1
            fi
        done
{{- end }}
