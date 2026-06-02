package usagebased

import (
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/pkg/models"
)

type RatingEngine string

const (
	// RatingEngineDelta rates the current cumulative meter snapshot, subtracts
	// already booked detailed lines, and books the remainder on the current run
	// service period.
	RatingEngineDelta RatingEngine = "delta"
	// RatingEnginePeriodPreserving rates cumulative snapshots per service
	// period and preserves correction lines on their original service periods.
	RatingEnginePeriodPreserving RatingEngine = "period_preserving"
)

func (e RatingEngine) Values() []string {
	return []string{
		string(RatingEngineDelta),
		string(RatingEnginePeriodPreserving),
	}
}

func (e RatingEngine) Validate() error {
	if !slices.Contains(e.Values(), string(e)) {
		return models.NewGenericValidationError(fmt.Errorf("invalid rating engine: %s", e))
	}

	return nil
}
