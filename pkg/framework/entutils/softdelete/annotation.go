package softdelete

import "entgo.io/ent/schema"

// noCascadeAnnotation marks an outgoing edge as exempt from soft-delete
// cascade. It is an escape hatch — the default behavior is to refuse a
// soft-delete-bearing parent that points to a child without `deleted_at`,
// because such an edge cannot be cascaded without either orphaning the
// child or hard-deleting it (which the project policy forbids).
//
// Use NoCascade only when the orphan is an acknowledged design decision.
type noCascadeAnnotation struct {
	schema.Annotation
}

// Name implements schema.Annotation.
func (noCascadeAnnotation) Name() string { return "SoftDeleteNoCascade" }

// NoCascade is an edge annotation that opts an individual outgoing edge
// out of soft-delete cascade. It is consulted by the entsoftdelete
// codegen extension when emitting per-node walkers.
func NoCascade() schema.Annotation { return noCascadeAnnotation{} }
