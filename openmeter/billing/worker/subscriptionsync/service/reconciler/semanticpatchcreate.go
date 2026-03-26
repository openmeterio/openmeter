package reconciler

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
)

type CreatePatch struct {
	UniqueID string
	Target   targetstate.SubscriptionItemWithPeriods
}

func (CreatePatch) semanticPatch() {}

func (p CreatePatch) Operation() SemanticPatchOperation {
	return SemanticPatchOperationCreate
}

func (p CreatePatch) UniqueReferenceID() string {
	return p.UniqueID
}

func (p CreatePatch) Expand(_ context.Context, input ExpandInput) ([]Patch, error) {
	return expandCreatePatch(input, p.Target)
}
