{{- $select := list "SUBJECT" "VALUE" "WINDOWSTART" "WINDOWEND" -}}

{{- range $groupByKey, $groupByValue := .GroupBy -}}
{{- $select = printf "`%s`" $groupByKey | append $select -}}
{{- end }}

{{- $clauses := list -}}
{{- if .Subject }}
{{- $clauses = (printf "SUBJECT = %s" (.Subject | derefstr | squote)) | append $clauses }}
{{- end -}}
{{- if .From }}
{{- $clauses = (printf "WINDOWSTART >= %s" (.From | dereftime | unixEpochMs)) | append $clauses }}
{{- end -}}
{{- if .To }}
{{- $clauses = (printf "WINDOWEND <= %s" (.To | dereftime | unixEpochMs)) | append $clauses }}
{{- end -}}

SELECT {{ $select | join ", " }} FROM {{ printf "OM_METER_%s" .Slug | upper | bquote }}
{{- if len $clauses }}
WHERE {{ $clauses | join " AND " }}
{{- end -}}
;
