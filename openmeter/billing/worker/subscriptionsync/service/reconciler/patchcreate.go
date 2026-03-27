package reconciler

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
)

type NewCreatePatchInput struct {
	UniqueID string
	Target   targetstate.SubscriptionItemWithPeriods
}

func (i NewCreatePatchInput) Validate() error {
	var errs []error
	if i.UniqueID == "" {
		errs = append(errs, errors.New("unique id is required"))
	}

	if err := i.Target.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("target: %w", err))
	}

	return errors.Join(errs...)
}

func (s *Service) NewCreatePatch(input NewCreatePatchInput) (Patch, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("new create patch: %w", err)
	}

	// TODO: use the service's field to decide if it should create a line or charge
	return LineCreatePatch{
		UniqueID: input.UniqueID,
		Target:   input.Target,
	}, nil
}

type LineCreatePatch struct {
	UniqueID string
	Target   targetstate.SubscriptionItemWithPeriods
}

func (p LineCreatePatch) Operation() PatchOperation {
	return PatchOperationCreate
}

func (p LineCreatePatch) UniqueReferenceID() string {
	return p.UniqueID
}

func (p LineCreatePatch) GetInvoicePatches(input GetInvoicePatchesInput) ([]invoiceupdater.Patch, error) {
	line, err := p.Target.GetExpectedLine(input.Subscription, input.Currency)
	if err != nil {
		return nil, fmt.Errorf("generating line from subscription item [%s]: %w", p.Target.SubscriptionItem.ID, err)
	}

	if line == nil {
		return nil, nil
	}

	return []invoiceupdater.Patch{invoiceupdater.NewCreateLinePatch(*line)}, nil
}
