package rte

import (
	"errors"
	"time"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ComparisonOperation string

const (
	ComparisonOperationRead      ComparisonOperation = "read"
	ComparisonOperationCalculate ComparisonOperation = "calculate"

	// Discard old observations after an outage instead of replaying a backlog
	// against a different workload and warming the cache for today's requests.
	ComparisonObservationTTL = time.Minute
)

// BalanceComparisonEvent pairs one successful production operation with the
// same balance operation in RTE. It carries timing and identity only.
type BalanceComparisonEvent struct {
	CalculationTarget
	Operation      ComparisonOperation `json:"operation"`
	DirectDuration time.Duration       `json:"direct_duration_ns"`
	ObservedAt     time.Time           `json:"observed_at"`
}

func (e BalanceComparisonEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{Subsystem: "rte", Name: "balanceComparisonRequested", Version: "v1"})
}

func (e BalanceComparisonEvent) Validate() error {
	errs := []error{e.CalculationTarget.Validate()}
	if e.Operation != ComparisonOperationRead && e.Operation != ComparisonOperationCalculate {
		errs = append(errs, errors.New("unknown comparison operation"))
	}

	if e.ObservedAt.IsZero() || e.DirectDuration < 0 {
		errs = append(errs, errors.New("observation requires a completion time and non-negative duration"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
