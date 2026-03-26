package reconciler

import (
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
)

type ExtendUsageBasedPatch struct {
	UniqueID string
	Existing billing.LineOrHierarchy
	Target   targetstate.SubscriptionItemWithPeriods
}

func (p ExtendUsageBasedPatch) Operation() PatchOperation {
	return PatchOperationExtend
}

func (p ExtendUsageBasedPatch) UniqueReferenceID() string {
	return p.UniqueID
}

func (p ExtendUsageBasedPatch) Expand(input ExpandInput) ([]invoiceupdater.Patch, error) {
	return expandExtendPatch(input, p.Existing, p.Target)
}
