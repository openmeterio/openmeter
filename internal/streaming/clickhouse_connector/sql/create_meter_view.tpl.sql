{{- $columns := list "subject String" "windowstart DateTime" "windowend DateTime" "value AggregateFunction(sum, Float64)" -}}
{{- $select := list "subject" "tumbleStart(time, toIntervalMinute(1)) AS windowstart" "tumbleEnd(time, toIntervalMinute(1)) AS windowend" -}}
{{- $groupBy := list "windowstart" "windowend" "subject" -}}

{{- $select = printf "sumState(cast(JSON_VALUE(data, %s), 'Float64')) AS value" (.ValueProperty | squote) | append $select -}}

{{- range $key, $value := .GroupBy -}}
{{- $columns = printf "%s String" $key | append $columns -}}
{{- $select = printf " JSON_VALUE(data, %s) as %s" ($value | squote) $key | append $select -}}
{{- $groupBy = printf "%s" $key | append $groupBy -}}
{{- end }}

CREATE MATERIALIZED VIEW IF NOT EXISTS {{ .Database }}.{{ .MeterViewName }} (
    {{ $columns | join ", " }}
) ENGINE = AggregatingMergeTree()
ORDER BY
    ({{ $groupBy | join ", " }}) AS
SELECT
    {{ $select | join ", " }}
FROM
    {{ .Database }}.{{ .EventsTableName }}
WHERE
    type = {{ .EventType | squote }}
GROUP BY
    {{ $groupBy | join ", " }};
