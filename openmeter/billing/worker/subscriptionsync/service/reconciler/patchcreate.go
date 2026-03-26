package reconciler

import (
	"fmt"

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

func (p CreatePatch) GetInvoicePatches(input GetInvoicePatchesInput) ([]invoiceupdater.Patch, error) {
	line, err := p.Target.GetExpectedLine(input.Subscription, input.Currency)
	if err != nil {
		return nil, fmt.Errorf("generating line from subscription item [%s]: %w", p.Target.SubscriptionItem.ID, err)
	}

	if line == nil {
		return nil, nil
	}

	return []invoiceupdater.Patch{invoiceupdater.NewCreateLinePatch(*line)}, nil
}
