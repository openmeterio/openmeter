package patch

import (
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/datex"
)

type timeRelation int

const (
	isCurrentPhase timeRelation = iota
	isFuturePhase
)

// Helper utility to get a given phase and determine its relation to the current phase
type phaseContentHelper struct {
	spec subscription.SubscriptionSpec
	actx subscription.ApplyContext
}

func (p phaseContentHelper) GetPhaseForEdit(phaseKey string) (*subscription.SubscriptionPhaseSpec, timeRelation, error) {
	phase, ok := p.spec.Phases[phaseKey]
	if !ok {
		return nil, 0, &subscription.PatchConflictError{Msg: fmt.Sprintf("phase %s not found", phaseKey)}
	}

	phaseStartTime, _ := phase.StartAfter.AddTo(p.spec.ActiveFrom)

	// 1. You cannot add items to previous phases
	currentPhase, exists := p.spec.GetCurrentPhaseAt(p.actx.CurrentTime)
	if !exists {
		// If the current phase doesn't exist then either all phases are in the past or in the future
		// If all phases are in the past then no addition is possible
		// If all phases are in the past then the selected one is also in the past
		if st, _ := phase.StartAfter.AddTo(p.spec.ActiveFrom); st.Before(p.actx.CurrentTime) {
			return nil, 0, &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot change contents of phase %s which starts before current phase", phaseKey)}
		} else {
			return phase, isFuturePhase, nil
		}
	} else {
		currentPhaseStartTime, _ := currentPhase.StartAfter.AddTo(p.spec.ActiveFrom)

		// If the selected phase is before the current phase, it's forbidden
		if phaseStartTime.Before(currentPhaseStartTime) {
			return nil, 0, &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot change contents of phase %s which starts before current phase", phaseKey)}
		} else if phase.PhaseKey == currentPhase.PhaseKey {
			return phase, isCurrentPhase, nil
		} else if phaseStartTime.After(currentPhaseStartTime) {
			return phase, isFuturePhase, nil
		} else {
			return nil, 0, fmt.Errorf("didn't enter any logical branch")
		}
	}
}

type relativeCadenceHelper struct {
	contentType    string
	phaseStartTime time.Time
	phaseKey       string
	rel            timeRelation
	actx           subscription.ApplyContext
}

func (r relativeCadenceHelper) ValidateRelativeCadence(c *subscription.CadenceOverrideRelativeToPhaseStart) error {
	if r.rel == isCurrentPhase {
		// 2. If it's added to the current phase, the specified start time cannot point to the past
		if c.ActiveFromOverride != nil {
			iST, _ := c.ActiveFromOverride.AddTo(r.phaseStartTime)
			if iST.Before(r.actx.CurrentTime) {
				return &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot add %s to phase %s which would become active in the past at %s", r.contentType, r.phaseKey, iST)}
			}
		} else {
			// 3. If it's added to the current phase, and start time is not specified, it will be set for the current time, as you cannot change the past
			diff := datex.Between(r.phaseStartTime, r.actx.CurrentTime)
			c.ActiveFromOverride = &diff
		}
	}

	return nil
}
