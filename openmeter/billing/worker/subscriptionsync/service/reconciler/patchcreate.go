package reconciler

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
)

type CreatePatch struct {
	Target targetstate.StateItem
}

func (p CreatePatch) Operation() PatchOperation {
	return PatchOperationCreate
}

func (p CreatePatch) UniqueReferenceID() string {
	return p.Target.UniqueID
}

func (p CreatePatch) GetInvoicePatches(input GetInvoicePatchesInput) ([]invoiceupdater.Patch, error) {
	line, err := p.Target.GetExpectedLine()
	if err != nil {
		return nil, fmt.Errorf("generating line from subscription item [%s]: %w", p.Target.SubscriptionItem.ID, err)
	}

	if line == nil {
		return nil, nil
	}

	return []invoiceupdater.Patch{invoiceupdater.NewCreateLinePatch(*line)}, nil
}
