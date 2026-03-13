{{/*
    Generates getters for fields that were added to schemas via mixins.

    For each node and each mixed-in field, it generates:
      func (e *<Node>) Get<Field>() <Type> { return e.<Field> }

    For nillable fields, it returns a pointer type (matching the generated entity field type).
*/}}
{{ define "entmixinaccessor" }}

{{ $pkg := base $.Config.Package }}
{{ template "header" $ }}

{{ range $n := $.Nodes }}
	{{ range $f := $n.Fields }}
		{{- if and $f.Position $f.Position.MixedIn }}

func (e *{{ $n.Name }}) Get{{ $f.StructField }}() {{ if $f.Nillable }}*{{ end }}{{ $f.Type }} {
	return e.{{ $f.StructField }}
}

		{{- end }}
	{{ end }}
{{ end }}

{{ end }}

