
{{- $groupBy := list "SUBJECT" -}}
{{- $select := list "SUBJECT" -}}

{{- range .GroupBy -}}
{{- $select = printf "COALESCE(EXTRACTJSONFIELD(data, '%s'), '') AS `%s`" . . | append $select -}}
{{- $groupBy = printf "COALESCE(EXTRACTJSONFIELD(data, '%s'), '')" . | append $groupBy -}}
{{- end }}

{{- if eq .Aggregation "COUNT" }}
{{- $select = printf "COUNT(*) AS VALUE" | append $select }}
{{- else if eq .Aggregation "COUNT_DISTINCT" }}
{{- $select = printf "COUNT_DISTINCT(EXTRACTJSONFIELD(data, '%s')) AS VALUE" .ValueProperty | append $select }}
{{- else }}
{{- $select = printf "%s(CAST(EXTRACTJSONFIELD(data, '%s') AS DECIMAL(12, 4))) AS VALUE" .Aggregation .ValueProperty | append $select }}
{{- end }}
CREATE TABLE IF NOT EXISTS {{ printf "OM_METER_%s" .ID | upper | bquote  }}
WITH (
    KAFKA_TOPIC = {{ printf "OM_METER_%s" .ID | lower | squote  }},
    KEY_FORMAT = 'JSON',
    VALUE_FORMAT = 'JSON',
    PARTITIONS = {{ .Partitions }}
) AS
SELECT {{ $select | join ", " }}
FROM
    OM_DETECTED_EVENTS_STREAM
WINDOW TUMBLING (
    SIZE {{ .WindowSize }},
    RETENTION {{ .WindowRetention }}
)
WHERE
    ID_COUNT = 1 AND
    TYPE = {{ .Type | squote }}
GROUP BY
    {{ $groupBy | join ", " }}
EMIT CHANGES;
