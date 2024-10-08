{{- if eq .Type.Type "integer" }}
/**
 *
 * Location: {{ .Type.Location }}
 */
scalar {{ .Type.Name }} extends uint64;
{{- else if eq .Type.Type "float64" }}
/**
 *
 * Location: {{ .Type.Location }}
 */
scalar {{ .Type.Name }} extends float64;
{{- else if eq .Type.Type "string" }}
/**
 *
 * Location: {{ .Type.Location }}
 */
scalar {{ .Type.Name }} extends string;
{{- else if eq .Type.Type "boolean" }}
/**
 *
 * Location: {{ .Type.Location }}
 */
scalar {{ .Type.Name }} extends boolean;
{{- else if eq .Type.Type "genericObject" }}
/**
 *
 * Location: {{ .Type.Location }}
 */
scalar {{ .Type.Name }} extends Record<unknown>;
{{- else }}
/**
 *
 * Location: {{ .Type.Location }}
 */
model {{ .Type.Name }} {
{{ range .Properties }}
  {{ .Name }}{{ if not .Required }}?{{ end }}: {{ .TypeString }};

{{ end }}
{{ if empty .Properties }}
{{ fail "no properties set"}}
{{ end }}
}

{{- end }}
