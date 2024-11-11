package model

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/pkg/datex"
)

func TestPhaseSort(t *testing.T) {
	tests := []struct {
		Name string

		SortFunc SortFunc
		Phases   []Phase
		Expected []Phase
	}{
		{
			Name:     "ByStartAfter",
			SortFunc: SortPhasesByStartAfter,
			Phases: []Phase{
				{
					PhaseGeneric: PhaseGeneric{
						Name:       "Pro",
						StartAfter: datex.MustParse(t, "P3M"),
					},
				},
				{
					PhaseGeneric: PhaseGeneric{
						Name:       "Trial",
						StartAfter: datex.MustParse(t, "P0D"),
					},
				},
				{
					PhaseGeneric: PhaseGeneric{
						Name:       "Pro 2",
						StartAfter: datex.MustParse(t, "P1Y"),
					},
				},
			},
			Expected: []Phase{
				{
					PhaseGeneric: PhaseGeneric{
						Name:       "Trial",
						StartAfter: datex.MustParse(t, "P0D"),
					},
				},
				{
					PhaseGeneric: PhaseGeneric{
						Name:       "Pro",
						StartAfter: datex.MustParse(t, "P3M"),
					},
				},
				{
					PhaseGeneric: PhaseGeneric{
						Name:       "Pro 2",
						StartAfter: datex.MustParse(t, "P1Y"),
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			SortPhases(test.Phases, test.SortFunc)

			assert.ElementsMatch(t, test.Expected, test.Phases)
		})
	}
}
