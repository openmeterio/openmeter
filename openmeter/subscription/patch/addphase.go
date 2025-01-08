package patch

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/datex"
)

type PatchAddPhase struct {
	PhaseKey    string
	CreateInput subscription.CreateSubscriptionPhaseInput
}

func (a PatchAddPhase) Op() subscription.PatchOperation {
	return subscription.PatchOperationAdd
}

func (a PatchAddPhase) Path() subscription.PatchPath {
	return subscription.NewPhasePath(a.PhaseKey)
}

func (a PatchAddPhase) Value() subscription.CreateSubscriptionPhaseInput {
	return a.CreateInput
}

func (a PatchAddPhase) ValueAsAny() any {
	return a.CreateInput
}

func (a PatchAddPhase) Validate() error {
	if err := a.Path().Validate(); err != nil {
		return err
	}

	if err := a.Op().Validate(); err != nil {
		return err
	}

	if err := a.CreateInput.Validate(); err != nil {
		return err
	}

	return nil
}

var _ subscription.ValuePatch[subscription.CreateSubscriptionPhaseInput] = PatchAddPhase{}

func (a PatchAddPhase) ApplyTo(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error {
	if _, exists := spec.Phases[a.PhaseKey]; exists {
		return &subscription.PatchConflictError{Msg: fmt.Sprintf("phase %s already exists", a.PhaseKey)}
	}

	// Checks we need:
	vST, _ := a.Value().StartAfter.AddTo(spec.ActiveFrom)

	// 2. You can only add a phase for future
	if !vST.After(actx.CurrentTime) {
		return &subscription.PatchForbiddenError{Msg: "cannot add phase in the past"}
	}

	// 3. You can only add a phase before the subscription ends
	if spec.ActiveTo != nil && !vST.Before(*spec.ActiveTo) {
		return &subscription.PatchForbiddenError{Msg: "cannot add phase after the subscription ends"}
	}

	// Let's apply the patch

	// Let's get all later phases & make sure their start times is aligned based on the new phase's duration:
	// 1. The very next phase should start based on the new phase's duration
	// 2. All other phases should preserve their relative start times i.e. spacing
	// To achieve this, we determine the difference between the next already scheduled phase's start and the duration, then add that difference to all later phases. Note that this difference is signed.

	sortedPhases := spec.GetSortedPhases()
	var diff datex.Period

	for i := range sortedPhases {
		p := sortedPhases[i]
		// We use !.Before() cause we might insert the phase at the same time another one starts
		if v, _ := p.StartAfter.AddTo(spec.ActiveFrom); !v.Before(vST) && diff.IsZero() {
			tillNextPhase, err := p.StartAfter.Subtract(a.Value().StartAfter)
			if err != nil {
				return fmt.Errorf("failed to calculate difference between phases: %w", err)
			}
			diff, err = a.Value().Duration.Subtract(tillNextPhase)
			if err != nil {
				return fmt.Errorf("failed to calculate difference between phases: %w", err)
			}
		}

		// Once we've reached the next phase lets increment the StartAfter by diff
		if !diff.IsZero() {
			sa, err := p.StartAfter.Add(diff)
			if err != nil {
				return fmt.Errorf("failed to adjust phase %s start time: %w", p.PhaseKey, err)
			}
			sortedPhases[i].StartAfter = sa
		}
	}

	// And then let's add the new phase
	spec.Phases[a.PhaseKey] = &subscription.SubscriptionPhaseSpec{
		CreateSubscriptionPhasePlanInput:     a.CreateInput.CreateSubscriptionPhasePlanInput,
		CreateSubscriptionPhaseCustomerInput: a.CreateInput.CreateSubscriptionPhaseCustomerInput,
		ItemsByKey:                           make(map[string][]subscription.SubscriptionItemSpec),
	}

	return nil
}
