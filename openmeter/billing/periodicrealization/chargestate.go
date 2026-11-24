package periodicrealization

import (
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/pkg/models"
)

// ChargeStatus is the outcome of a charge for the run it is currently stamped
// for. Stamping the charge for a later run overwrites it.
type ChargeStatus string

const (
	// ChargeStatusPending means the run collected the charge and has not
	// attempted it yet.
	ChargeStatusPending ChargeStatus = "pending"
	// ChargeStatusDone means the charge has a checkpoint for the run.
	ChargeStatusDone ChargeStatus = "done"
	// ChargeStatusFailed means the attempt failed. It is not retried within the
	// run; the next run covers the usage.
	ChargeStatusFailed ChargeStatus = "failed"
	// ChargeStatusSkipped means no checkpoint was created for a reason that is
	// not an error, for example the charge left the run's scope or already has a
	// run past the period end.
	ChargeStatusSkipped ChargeStatus = "skipped"
)

func (ChargeStatus) Values() []string {
	return []string{
		string(ChargeStatusPending),
		string(ChargeStatusDone),
		string(ChargeStatusFailed),
		string(ChargeStatusSkipped),
	}
}

func (s ChargeStatus) Validate() error {
	if !slices.Contains(s.Values(), string(s)) {
		return models.NewGenericValidationError(fmt.Errorf("invalid periodic realization charge status: %q", s))
	}

	return nil
}
