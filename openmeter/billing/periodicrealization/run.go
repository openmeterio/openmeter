// Package periodicrealization schedules and tracks the periodic revenue
// recognition runs of a namespace and the per-charge work they cover.
package periodicrealization

import (
	"fmt"
	"slices"

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
