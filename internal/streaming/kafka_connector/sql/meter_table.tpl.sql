
{{- $groupBy := list "SUBJECT" -}}
{{- $select := list "SUBJECT AS KEY1, AS_VALUE(SUBJECT) AS SUBJECT, WINDOWSTART AS WINDOWSTART_TS, WINDOWEND AS WINDOWEND_TS" -}}

{{- range $groupByKey, $groupByValue := .GroupBy -}}
{{- $select = printf "COALESCE(EXTRACTJSONFIELD(data, '%s'), '') AS `%s_KEY`" $groupByValue $groupByKey | append $select -}}
{{- $select = printf "AS_VALUE(COALESCE(EXTRACTJSONFIELD(data, '%s'), '')) AS `%s`" $groupByValue $groupByKey | append $select -}}
{{- $groupBy = printf "COALESCE(EXTRACTJSONFIELD(data, '%s'), '')" $groupByValue | append $groupBy -}}
{{- end -}}

{{- if eq .Aggregation "COUNT" }}
    {{- if .ValueProperty }}
    {{- $select = printf "COUNT(EXTRACTJSONFIELD(data, '%s')) AS VALUE" .ValueProperty | append $select }}
    {{- else }}
    {{- $select = printf "COUNT(*) AS VALUE" | append $select }}
    {{- end }}
{{- else }}
{{- $select = printf "%s(CAST(EXTRACTJSONFIELD(data, '%s') AS DECIMAL(12, 4))) AS VALUE" .Aggregation .ValueProperty | append $select }}
{{- end }}
CREATE TABLE IF NOT EXISTS {{ printf "OM_%s_METER_%s" .Namespace .Slug | upper | bquote  }}
WITH (
    KAFKA_TOPIC = {{ printf "OM_%s_METER_%s" .Namespace .Slug | lower | squote  }},
    KEY_FORMAT = 'JSON_SR',
    VALUE_FORMAT = 'JSON_SR',
    PARTITIONS = {{ .Partitions }}
) AS
SELECT {{ $select | join ", " }}
FROM
    OM_{{ .Namespace | upper }}_DETECTED_EVENTS_STREAM
WINDOW TUMBLING (
    SIZE 1 {{ .WindowSize }},
    RETENTION {{ .WindowRetention }}
)
WHERE
    ID_COUNT = 1 AND
    TYPE = {{ .EventType | squote }}
GROUP BY
    {{ $groupBy | join ", " }}
EMIT CHANGES;
