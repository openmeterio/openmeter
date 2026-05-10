// Package entsoftdelete is an entc extension that generates per-node
// soft-delete cascade walkers and registers them with
// pkg/framework/entutils/softdelete at init() time.
//
// For every Node whose schema declares a `deleted_at` field, the extension
// emits a `softdelete_<node>.go` file under the generated db/ package
// containing:
//
//   - softDeleteCascade<Node>(ctx, client, parentIDs) — walks each outgoing
//     assoc edge whose target Node also has `deleted_at`, queries currently-
//     active descendant IDs filtered by the FK column, stamps deleted_at on
//     them, and recurses through softdelete.RunCascade so init() ordering
//     between generated files is irrelevant.
//
//   - An init() block that calls softdelete.Register("<Node>", softDeleteCascade<Node>).
//
// Edges with the softdelete.NoCascade() annotation, edges to non-soft-delete
// targets, M2M edges, and inverse edges are skipped. The walker uses the
// parent mutation's *Client (passed in by the soft-delete hook) so cascade
// writes participate in the caller's transaction.
package entsoftdelete

import (
	_ "embed"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
)

//go:embed softdelete.tpl
var tmplfile string

// Extension implements entc.Extension.
type Extension struct {
	entc.DefaultExtension
}

func (Extension) Templates() []*gen.Template {
	return []*gen.Template{
		gen.MustParse(gen.NewTemplate("entsoftdelete").Parse(tmplfile)),
	}
}

func New() *Extension {
	return &Extension{}
}
