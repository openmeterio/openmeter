package plan

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
					Name:       "Pro",
					StartAfter: datex.MustParse(t, "P3M"),
				},
				{
					Name:       "Trial",
					StartAfter: datex.MustParse(t, "P0D"),
				},
				{
					Name:       "Pro 2",
					StartAfter: datex.MustParse(t, "P1Y"),
				},
			},
			Expected: []Phase{
				{
					Name:       "Trial",
					StartAfter: datex.MustParse(t, "P0D"),
				},
				{
					Name:       "Pro",
					StartAfter: datex.MustParse(t, "P3M"),
				},
				{
					Name:       "Pro 2",
					StartAfter: datex.MustParse(t, "P1Y"),
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
