package reconciler

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/samber/lo"
)

type DeletePatch struct {
	UniqueID string
	Existing persistedstate.Entity
}

func (s *Service) NewDeletePatch(existing persistedstate.Entity) (Patch, error) {
	if existing == nil {
		return nil, errors.New("new delete patch: existing entity is required")
	}

	return newFromEntity(newFromEntityInput{
		Entity: existing,
		NewInvoicePatch: func(lineOrHierarchy billing.LineOrHierarchy) (Patch, error) {
			return LineDeletePatch{
				Existing: lineOrHierarchy,
			}, nil
		},
	})
}

type LineDeletePatch struct {
	Existing billing.LineOrHierarchy
}

func (p LineDeletePatch) Operation() PatchOperation {
	return PatchOperationDelete
}

func (p LineDeletePatch) UniqueReferenceID() string {
	return lo.FromPtr(p.Existing.ChildUniqueReferenceID())
}

func (p LineDeletePatch) GetInvoicePatches(input GetInvoicePatchesInput) ([]invoiceupdater.Patch, error) {
	return invoiceupdater.GetDeletePatchesForLine(p.Existing)
}
