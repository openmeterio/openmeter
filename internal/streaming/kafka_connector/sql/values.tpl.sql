{{- $clauses := list -}}
{{- if .Subject }}
{{- $clauses = append $clauses (printf "SUBJECT = %s" (.Subject | derefstr | squote)) }}
{{- end -}}
{{- if .From }}
{{- $clauses = append $clauses (printf "WINDOWSTART >= %s" (.From | dereftime | unixEpochMs)) }}
{{- end -}}
{{- if .To }}
{{- $clauses = append $clauses (printf "WINDOWEND < %s" (.To | dereftime | unixEpochMs)) }}
{{- end -}}

SELECT * FROM {{ printf "OM_METER_%s" .ID | upper | bquote }}
{{- if len $clauses }}
WHERE {{ $clauses | join " AND " }}
{{- end -}}
;
