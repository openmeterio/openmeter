package productcatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/pkg/datex"
)

func TestPhaseSort(t *testing.T) {
	tests := []struct {
		Name string

		SortFunc SortPhasesFunc
		Phases   []Phase
		Expected []Phase
	}{
		{
			Name:     "ByStartAfter",
			SortFunc: SortPhasesByStartAfter,
			Phases: []Phase{
				{
					PhaseMeta: PhaseMeta{
						Key:        "pro",
						Name:       "Pro",
						StartAfter: datex.MustParse(t, "P3M"),
					},
				},
				{
					PhaseMeta: PhaseMeta{
						Key:        "trial",
						Name:       "Trial",
						StartAfter: datex.MustParse(t, "P0M"),
					},
				},
				{
					PhaseMeta: PhaseMeta{
						Key:        "pro-2",
						Name:       "Pro 2",
						StartAfter: datex.MustParse(t, "P1Y"),
					},
				},
			},
			Expected: []Phase{
				{
					PhaseMeta: PhaseMeta{
						Key:        "trial",
						Name:       "Trial",
						StartAfter: datex.MustParse(t, "P0M"),
					},
				},
				{
					PhaseMeta: PhaseMeta{
						Key:        "pro",
						Name:       "Pro",
						StartAfter: datex.MustParse(t, "P3M"),
					},
				},
				{
					PhaseMeta: PhaseMeta{
						Key:        "pro-2",
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
