package reconciler

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
)

type ExtendPatch struct {
	UniqueID string
	Existing billing.LineOrHierarchy
	Target   targetstate.SubscriptionItemWithPeriods
}

func (ExtendPatch) semanticPatch() {}

func (p ExtendPatch) Operation() SemanticPatchOperation {
	return SemanticPatchOperationExtend
}

func (p ExtendPatch) UniqueReferenceID() string {
	return p.UniqueID
}

func (p ExtendPatch) Expand(_ context.Context, input ExpandInput) ([]invoiceupdater.Patch, error) {
	return expandExistingPatch(input, p.Existing, p.Target, SemanticPatchOperationExtend)
}
