package meta

import (
	"errors"
	"fmt"

	"github.com/qmuntal/stateless"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	_                                  Patch = (*PatchUpdateSubscriptionReference)(nil)
	TriggerUpdateSubscriptionReference       = stateless.Trigger("update_subscription_reference")
)

type PatchUpdateSubscriptionReference struct {
	phaseID            *string
	subscriptionItemID *string
}

type NewPatchUpdateSubscriptionReferenceInput struct {
	PhaseID            *string
	SubscriptionItemID *string
}

// AsPatchUpdateSubscriptionReference expresses reassignment within a subscription.
// A charge may follow a replacement phase or item, but changing the root subscription
// would move the charge between independently reconciled ownership boundaries.
func (r SubscriptionReference) AsPatchUpdateSubscriptionReference(existing SubscriptionReference) (PatchUpdateSubscriptionReference, error) {
	if err := r.Validate(); err != nil {
		return PatchUpdateSubscriptionReference{}, fmt.Errorf("target subscription reference: %w", err)
	}
	if err := existing.Validate(); err != nil {
		return PatchUpdateSubscriptionReference{}, fmt.Errorf("existing subscription reference: %w", err)
	}
	if r.SubscriptionID != existing.SubscriptionID {
		return PatchUpdateSubscriptionReference{}, errors.New("subscription ID cannot be updated")
	}

	input := NewPatchUpdateSubscriptionReferenceInput{}
	if r.PhaseID != existing.PhaseID {
		input.PhaseID = lo.ToPtr(r.PhaseID)
	}
	if r.ItemID != existing.ItemID {
		input.SubscriptionItemID = lo.ToPtr(r.ItemID)
	}

	return NewPatchUpdateSubscriptionReference(input)
}

func NewPatchUpdateSubscriptionReference(input NewPatchUpdateSubscriptionReferenceInput) (PatchUpdateSubscriptionReference, error) {
	patch := PatchUpdateSubscriptionReference{}
	if input.PhaseID != nil {
		phaseID := *input.PhaseID
		patch.phaseID = &phaseID
	}
	if input.SubscriptionItemID != nil {
		subscriptionItemID := *input.SubscriptionItemID
		patch.subscriptionItemID = &subscriptionItemID
	}

	if err := patch.Validate(); err != nil {
		return PatchUpdateSubscriptionReference{}, err
	}

	return patch, nil
}

func (p PatchUpdateSubscriptionReference) Apply(reference SubscriptionReference) (SubscriptionReference, error) {
	if err := p.Validate(); err != nil {
		return SubscriptionReference{}, err
	}

	reference.PhaseID = lo.FromPtrOr(p.phaseID, reference.PhaseID)
	reference.ItemID = lo.FromPtrOr(p.subscriptionItemID, reference.ItemID)

	return reference, nil
}

func (p PatchUpdateSubscriptionReference) GetTargetLayer(LayeredIntentReader) (ChangeTarget, error) {
	return ChangeTargetBase, nil
}

func (p PatchUpdateSubscriptionReference) Op() PatchType {
	return PatchTypeUpdateSubscriptionReference
}

func (p PatchUpdateSubscriptionReference) Trigger() stateless.Trigger {
	return TriggerUpdateSubscriptionReference
}

func (p PatchUpdateSubscriptionReference) Validate() error {
	var errs []error

	if p.phaseID == nil && p.subscriptionItemID == nil {
		errs = append(errs, errors.New("at least one subscription reference update is required"))
	}

	if p.phaseID != nil && *p.phaseID == "" {
		errs = append(errs, errors.New("phase ID is required"))
	}

	if p.subscriptionItemID != nil && *p.subscriptionItemID == "" {
		errs = append(errs, errors.New("subscription item ID is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
