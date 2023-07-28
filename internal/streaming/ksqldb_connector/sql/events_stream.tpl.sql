{{- $with := list -}}
{{- $with = printf "KAFKA_TOPIC = '%s'" .Topic | append $with -}}
{{- $with = printf "VALUE_FORMAT = '%s'" .Format | append $with -}}

{{if eq .Format "JSON" }}
    {{- $with = printf "KEY_FORMAT = 'NONE'" | append $with -}}
{{end}}
{{if eq .Format "JSON_SR" }}
    {{- $with = printf "KEY_FORMAT = '%s'" | .Format  | append $with -}}
    {{- $with = printf "KEY_SCHEMA_ID = %d" .KeySchemaId | append $with -}}
    {{- $with = printf "VALUE_SCHEMA_ID = %d" .ValueSchemaId | append $with -}}
{{end}}

CREATE STREAM IF NOT EXISTS OM_{{ .Namespace | upper }}_EVENTS
{{if eq .Format "JSON" }}
(
    id STRING,
    type STRING,
    source STRING,
    subject STRING,
    time STRING,
    data STRING
)
{{end}}
WITH (
    {{ $with | join ", " }}
);
