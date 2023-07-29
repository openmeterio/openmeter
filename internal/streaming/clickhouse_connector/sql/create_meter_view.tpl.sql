{{- $columns := list "subject String" "windowstart DateTime" "windowend DateTime" "value AggregateFunction(sum, Float64)" -}}
{{- $select := list "subject" "tumbleStart(time, toIntervalMinute(1)) AS windowstart" "tumbleEnd(time, toIntervalMinute(1)) AS windowend" "sumState(value) AS value" -}}
{{- $groupBy := list "windowstart" "windowend" "subject" -}}

{{- range $key, $value := .GroupBy -}}
{{- $columns = printf "%s String" $key | append $columns -}}
{{- $select = printf " JSON_VALUE(DATA, %s) as %s" ($value | squote) $key | append $select -}}
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
    meterSlug = {{ .MeterSlug | squote }}
GROUP BY
    {{ $groupBy | join ", " }};
