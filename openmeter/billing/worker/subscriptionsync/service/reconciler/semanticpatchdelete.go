package reconciler

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type DeletePatch struct {
	UniqueID string
	Existing billing.LineOrHierarchy
}

func (DeletePatch) semanticPatch() {}

func (p DeletePatch) Operation() SemanticPatchOperation {
	return SemanticPatchOperationDelete
}

func (p DeletePatch) UniqueReferenceID() string {
	return p.UniqueID
}

func (p DeletePatch) Expand(_ context.Context, _ ExpandInput) ([]Patch, error) {
	return GetDeletePatchesForLine(p.Existing)
}
