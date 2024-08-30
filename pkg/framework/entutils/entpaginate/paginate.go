package entpaginate

import (
	_ "embed"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
)

//go:embed paginate.tpl
var tmplfile string

// Extension implements entc.Extension.
type Extension struct {
	entc.DefaultExtension
}

func (Extension) Templates() []*gen.Template {
	return []*gen.Template{
		gen.MustParse(gen.NewTemplate("entpaginate").Parse(tmplfile)),
	}
}

func New() *Extension {
	return &Extension{}
}
