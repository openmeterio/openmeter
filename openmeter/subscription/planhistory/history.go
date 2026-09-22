// Package planhistory records the effective catalog plan versions of a subscription.
package planhistory

import (
	"errors"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

// PlanVersion is the concrete catalog version effective for a subscription.
type PlanVersion struct {
	Key     string `json:"key"`
	Version int    `json:"version"`
}

func (p PlanVersion) Validate() error {
	var errs []error
	if p.Key == "" {
		errs = append(errs, errors.New("plan key is required"))
	}
	if p.Version < 1 {
		errs = append(errs, errors.New("plan version must be positive"))
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type Change struct {
	EffectiveAt time.Time   `json:"effective_at"`
	Plan        PlanVersion `json:"plan"`
}

type History []Change

// At returns a snapshot of the last recorded plan effective at the given time.
// Missing history stays unattributed; the current plan reference cannot reconstruct it.
func (h History) At(at time.Time) *PlanVersion {
	var matched *Change
	for i := range h {
		change := &h[i]
		if !change.EffectiveAt.After(at) && (matched == nil || !change.EffectiveAt.Before(matched.EffectiveAt)) {
			matched = change
		}
	}
	if matched == nil {
		return nil
	}
	plan := matched.Plan
	return &plan
}
