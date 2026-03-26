package reconciler

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
)

type ShrinkPatch struct {
	UniqueID string
	Existing billing.LineOrHierarchy
	Target   targetstate.SubscriptionItemWithPeriods
}

func (ShrinkPatch) semanticPatch() {}

func (p ShrinkPatch) Operation() SemanticPatchOperation {
	return SemanticPatchOperationShrink
}

func (p ShrinkPatch) UniqueReferenceID() string {
	return p.UniqueID
}

func (p ShrinkPatch) Expand(_ context.Context, input ExpandInput) ([]invoiceupdater.Patch, error) {
	return expandExistingPatch(input, p.Existing, p.Target, SemanticPatchOperationShrink)
}
