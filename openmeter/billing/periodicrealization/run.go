// Package periodicrealization schedules and tracks the periodic revenue
// recognition runs of a namespace and the per-charge work they cover.
package periodicrealization

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

// RunStatus is the lifecycle state of a namespace's run for one period.
type RunStatus string

const (
	// RunStatusScheduled means the run exists but has not collected its charges
	// yet. Charge runs can already reference it.
	RunStatusScheduled RunStatus = "scheduled"
	// RunStatusRunning means the in-scope charges were collected and some of
	// them may still be pending.
	RunStatusRunning RunStatus = "running"
	// RunStatusCompleted means every collected charge was done or skipped.
	RunStatusCompleted RunStatus = "completed"
	// RunStatusCompletedWithErrors means every collected charge was attempted
	// and at least one failed, or the run was closed while work was still pending.
	RunStatusCompletedWithErrors RunStatus = "completed_with_errors"
	// RunStatusFailed means the run itself could not be executed, for example
	// because its charges could not be collected.
	RunStatusFailed RunStatus = "failed"
)

func (RunStatus) Values() []string {
	return []string{
		string(RunStatusScheduled),
		string(RunStatusRunning),
		string(RunStatusCompleted),
		string(RunStatusCompletedWithErrors),
		string(RunStatusFailed),
	}
}

func (s RunStatus) Validate() error {
	if !slices.Contains(s.Values(), string(s)) {
		return models.NewGenericValidationError(fmt.Errorf("invalid periodic realization run status: %q", s))
	}

	return nil
}

var _ models.Validator = (*Run)(nil)

// Run is a namespace's realization run for one period.
type Run struct {
	models.NamespacedID
	models.ManagedModel

	// PreviousRunID is the namespace's latest run when this one was scheduled,
	// or nil for the first run. Periods without a run are skipped, so the
	// previous run is not always one period back.
	PreviousRunID *string
	// PeriodTo is the end of the period the run realizes usage in. The period
	// excludes it.
	PeriodTo time.Time
	// RealizeAfter is the earliest time the run may realize its period: the
	// period end plus the collection delay that applied when the run was created.
	RealizeAfter time.Time
	Status       RunStatus
}

func (r Run) Validate() error {
	var errs []error

	if err := r.NamespacedID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if r.PreviousRunID != nil && *r.PreviousRunID == "" {
		errs = append(errs, errors.New("previous run id must not be empty when set"))
	}

	if r.PeriodTo.IsZero() {
		errs = append(errs, errors.New("period to is required"))
	}

	if r.RealizeAfter.Before(r.PeriodTo) {
		errs = append(errs, errors.New("realize after must not be before period to"))
	}

	if err := r.Status.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("status: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
