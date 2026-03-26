package reconciler

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
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

func (p DeletePatch) Expand(_ context.Context, _ ExpandInput) ([]invoiceupdater.Patch, error) {
	return invoiceupdater.GetDeletePatchesForLine(p.Existing)
}
