{{/*sourced from https://github.com/ent/ent/issues/1119#issuecomment-1387280573*/}}
{{ define "setorclear" }}
	{{/* Add the base header for the generated file */}}
	{{ $pkg := base $.Config.Package }}
	{{ template "header" $ }}

	{{/* Loop over all updaters and implement the "SetOrClear" method for all optional fields */}}
	{{ range $n := $.Nodes }}
		{{ $updater := $n.UpdateName }}
		{{ range $f := $n.Fields }}
			{{ if and ($f.Optional) (not $f.Immutable)}}
				{{ $set := print "Set" $f.StructField }}
				{{ $clear := print "Clear" $f.StructField }}
				func (u *{{ $updater }}) SetOrClear{{ $f.StructField }}(value *{{ $f.Type }}) *{{ $updater }} {
					if value == nil {
						return u.{{ $clear }}()
					}
					return u.{{ $set }}(*value)
				}

                {{/* ent has two update paths */}}
				func (u *{{ $updater }}One) SetOrClear{{ $f.StructField }}(value *{{ $f.Type }}) *{{ $updater }}One {
					if value == nil {
						return u.{{ $clear }}()
					}
					return u.{{ $set }}(*value)
				}
			{{ end }}
		{{ end }}
	{{ end }}
{{ end }}
