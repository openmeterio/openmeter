{{- $select := list "windowstart" "windowend" "subject" "sumMerge(value) AS value" -}}
{{- $groupBy := list "windowstart" "windowend" "subject" -}}
{{- $where := list -}}

{{- range .GroupBy -}}
{{- $select = printf "%s" . | append $select -}}
{{- $groupBy = printf "%s" . | append $groupBy -}}
{{- end }}

{{- if .Subject }}
{{- $where = (printf "subject = %s" (.Subject | derefstr | squote)) | append $where }}
{{- end -}}
{{- if .From }}
{{- $where = (printf "windowstart >= toDateTime(%s)" (.From | dereftime | unixEpochMs)) | append $where }}
{{- end -}}
{{- if .To }}
{{- $where = (printf "windowend <= toDateTime(%s)" (.To | dereftime | unixEpochMs)) | append $where }}
{{- end -}}

SELECT
    {{ $select | join ", " }}
FROM
    {{ .Database }}.{{ .MeterViewName }}
{{ if gt (len $where) 0 }}
WHERE
    {{ $where | join " AND " }}
{{- end -}}
GROUP BY
    {{ $groupBy | join ", " }}
ORDER BY windowstart;
