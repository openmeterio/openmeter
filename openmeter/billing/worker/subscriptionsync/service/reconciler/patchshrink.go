package reconciler

import (
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
)

type ShrinkUsageBasedPatch struct {
	UniqueID string
	Existing billing.LineOrHierarchy
	Target   targetstate.SubscriptionItemWithPeriods
}

func (p ShrinkUsageBasedPatch) Operation() PatchOperation {
	return PatchOperationShrink
}

func (p ShrinkUsageBasedPatch) UniqueReferenceID() string {
	return p.UniqueID
}

func (p ShrinkUsageBasedPatch) Expand(input ExpandInput) ([]invoiceupdater.Patch, error) {
	return expandShrinkPatch(input, p.Existing, p.Target)
}
