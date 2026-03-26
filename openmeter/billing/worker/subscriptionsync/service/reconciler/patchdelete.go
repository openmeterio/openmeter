package reconciler

import (
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
)

type DeletePatch struct {
	UniqueID string
	Existing billing.LineOrHierarchy
}

func (p DeletePatch) Operation() PatchOperation {
	return PatchOperationDelete
}

func (p DeletePatch) UniqueReferenceID() string {
	return p.UniqueID
}

func (p DeletePatch) Expand(_ ExpandInput) ([]invoiceupdater.Patch, error) {
	return invoiceupdater.GetDeletePatchesForLine(p.Existing)
}
