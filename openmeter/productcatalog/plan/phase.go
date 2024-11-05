package plan

import (
	"errors"
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
)

const DefaultStartAfter = "P0D"

type Phase struct {
	models.NamespacedID
	models.ManagedModel

	// Key is the unique key for Phase.
	Key string `json:"key"`

	// Name
	Name string `json:"name"`

	// Description
	Description *string `json:"description,omitempty"`

	// Metadata
	Metadata map[string]string `json:"metadata,omitempty"`

	// StartAfter
	StartAfter datex.Period `json:"interval"`

	// RateCards
	RateCards []RateCard `json:"rateCards"`

	// Discounts
	Discounts []Discount `json:"discounts"`

	// PlanID
	PlanID string `json:"-"`
}

func (p Phase) Validate() error {
	var errs []error

	if p.StartAfter.IsNegative() {
		errs = append(errs, fmt.Errorf("startAfter must not be negative"))
	}

	// Check for
	// * duplicated rate card keys
	// * namespace mismatch
	// * invalid RateCards
	rateCardKeys := make(map[string]RateCard)
	for _, rateCard := range p.RateCards {
		if _, ok := rateCardKeys[rateCard.Key()]; ok {
			errs = append(errs, fmt.Errorf("duplicated rate card: %s", rateCard.Key()))
		} else {
			rateCardKeys[rateCard.Key()] = rateCard
		}

		if rateCard.Namespace() != p.Namespace {
			errs = append(errs, fmt.Errorf("invalid rate card %s: namespace mismatch %s", rateCard.Key(), rateCard.Namespace()))
		}

		if err := rateCard.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid rate card %s: %w", rateCard.Key(), err))
		}
	}

	for _, discount := range p.Discounts {
		if err := discount.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid discount: %w", err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

type SortFunc func(left, right Phase) int

var SortPhasesByStartAfter SortFunc = func(left, right Phase) int {
	lt, _ := left.StartAfter.Duration()
	rt, _ := right.StartAfter.Duration()

	if lt > rt {
		return 1
	} else if lt < rt {
		return -1
	}

	return 0
}

var SortPhases = slices.SortFunc[[]Phase, Phase]
