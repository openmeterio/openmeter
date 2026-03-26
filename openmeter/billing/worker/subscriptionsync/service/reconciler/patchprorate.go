package reconciler

import (
	"context"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type ProratePatch struct {
	UniqueID string
	Existing billing.LineOrHierarchy
	Target   targetstate.SubscriptionItemWithPeriods

	OriginalPeriod timeutil.ClosedPeriod
	TargetPeriod   timeutil.ClosedPeriod

	OriginalAmount alpacadecimal.Decimal
	TargetAmount   alpacadecimal.Decimal
}

func (ProratePatch) semanticPatch() {}

func (p ProratePatch) Operation() SemanticPatchOperation {
	return SemanticPatchOperationProrate
}

func (p ProratePatch) UniqueReferenceID() string {
	return p.UniqueID
}

func (p ProratePatch) Expand(_ context.Context, input ExpandInput) ([]invoiceupdater.Patch, error) {
	return expandExistingPatch(input, p.Existing, p.Target, SemanticPatchOperationProrate)
}
