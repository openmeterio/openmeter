{{- $select := list "`subject`" "`value`" "windowstart as `windowstart`" "windowend as `windowend`" -}}

{{- range $groupByKey := .GroupBy -}}
{{- $select = printf "`%s`" $groupByKey | append $select -}}
{{- end -}}

{{- $clauses := list -}}
{{- if .Subject }}
{{- $clauses = (printf "subject = %s" (.Subject | derefstr | squote)) | append $clauses }}
{{- end -}}
{{- if .From }}
{{- $clauses = (printf "windowstart >= %s" (.From | dereftime | unixEpochMs)) | append $clauses }}
{{- end -}}
{{- if .To }}
{{- $clauses = (printf "windowend <= %s" (.To | dereftime | unixEpochMs)) | append $clauses }}
{{- end -}}

SELECT {{ $select | join ", " }} FROM {{ printf "OM_%s_METER_%s" .Namespace .Slug | upper | bquote }}
{{- if len $clauses }}
WHERE {{ $clauses | join " AND " }}
{{- end -}}
;
