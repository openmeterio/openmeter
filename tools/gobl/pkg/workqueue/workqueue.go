package workqueue

import (
	"slices"

	"github.com/openmeterio/openmeter/tools/gobl/pkg/schematype"
	"github.com/samber/lo"
)

type Queue struct {
	pendingTypes map[string]*schematype.TypeRef
	emittedTypes map[string]struct{}
}

func New() *Queue {
	return &Queue{
		pendingTypes: make(map[string]*schematype.TypeRef),
		emittedTypes: make(map[string]struct{}),
	}
}

func (q *Queue) Add(schema *schematype.TypeRef) {
	if _, ok := q.emittedTypes[schema.GetTargetRef().Location]; ok {
		return
	}
	q.pendingTypes[schema.GetTargetRef().Location] = schema
}

func (q *Queue) Next() *schematype.TypeRef {
	pendingItems := lo.Keys(q.pendingTypes)

	if len(pendingItems) == 0 {
		return nil
	}

	slices.Sort(pendingItems)

	return q.pendingTypes[pendingItems[0]]
}

func (q *Queue) Emitted(schema *schematype.TypeRef) {
	q.emittedTypes[schema.GetTargetRef().Location] = struct{}{}
	delete(q.pendingTypes, schema.GetTargetRef().Location)
}
