package reconciler

import (
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
)

type CreatePatch struct {
	UniqueID string
	Target   targetstate.SubscriptionItemWithPeriods
}

func (p CreatePatch) Operation() PatchOperation {
	return PatchOperationCreate
}

func (p CreatePatch) UniqueReferenceID() string {
	return p.UniqueID
}

func (p CreatePatch) Expand(input ExpandInput) ([]invoiceupdater.Patch, error) {
	return expandCreatePatch(input, p.Target)
}
