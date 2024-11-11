package model

import (
	"errors"
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/pkg/datex"
)

type PhaseGeneric struct {
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

	// PlanID
	PlanID string `json:"-"`
}

type Phase struct {
	PhaseGeneric

	// RateCards
	RateCards []RateCard `json:"rateCards"`

	// Discounts
	Discounts []Discount `json:"discounts"`
}

func (p Phase) Validate() error {
	var errs []error

	if p.StartAfter.IsNegative() {
		errs = append(errs, fmt.Errorf("the StartAfter period must not be negative"))
	}

	// Check for
	// * duplicated rate card keys
	// * namespace mismatch
	// * invalid RateCards
	rateCardKeys := make(map[string]RateCard)
	for _, rateCard := range p.RateCards {
		if _, ok := rateCardKeys[rateCard.Key()]; ok {
			errs = append(errs, fmt.Errorf("duplicated RateCard: %s", rateCard.Key()))
		} else {
			rateCardKeys[rateCard.Key()] = rateCard
		}

		if err := rateCard.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid RateCard: %w", err))
		}
	}

	for _, discount := range p.Discounts {
		if err := discount.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid Discount: %w", err))
		}

		for _, key := range discount.RateCardKeys() {
			if _, ok := rateCardKeys[key]; !ok {
				errs = append(errs, fmt.Errorf("invalid Discount: unknown RateCard: %s", key))
			}
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
