{{/*
    This package exposes the generated ScanValues and AssignValues methods for each resource.
*/}}
{{ define "entscan" }}

{{/* Add the base header for the generated file */}}
{{ $pkg := base $.Config.Package }}
{{ template "header" $ }}

{{ range $n := $.Nodes }}
// {{ $n.Name }}
func (e *{{ $n.Name }}) ScanValues(columns []string) ([]any, error) {
	return e.scanValues(columns)
}

func (e *{{ $n.Name }}) AssignValues(columns []string, values []any) error {
	return e.assignValues(columns, values)
}

{{ end }}

{{ end }}
