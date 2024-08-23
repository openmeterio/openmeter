package notification

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChannelIDsDifference(t *testing.T) {
	tests := []struct {
		Name string

		New []string
		Old []string

		ExpectedAdditions []string
		ExpectedRemovals  []string
	}{
		{
			Name: "No change",
			New: []string{
				"channel-1",
				"channel-2",
			},
			Old: []string{
				"channel-2",
				"channel-1",
			},
		},
		{
			Name: "Add new channel",
			New: []string{
				"channel-1",
				"channel-2",
				"channel-3",
			},
			Old: []string{
				"channel-2",
				"channel-1",
			},
			ExpectedAdditions: []string{
				"channel-3",
			},
		},
		{
			Name: "Remove old channel",
			New: []string{
				"channel-2",
			},
			Old: []string{
				"channel-2",
				"channel-1",
			},
			ExpectedRemovals: []string{
				"channel-1",
			},
		},
		{
			Name: "Add and remove channels",
			New: []string{
				"channel-1",
				"channel-3",
			},
			Old: []string{
				"channel-2",
				"channel-1",
			},
			ExpectedAdditions: []string{
				"channel-3",
			},
			ExpectedRemovals: []string{
				"channel-2",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			diff := NewChannelIDsDifference(test.New, test.Old)

			assert.ElementsMatch(t, test.ExpectedAdditions, diff.Additions())
			assert.ElementsMatch(t, test.ExpectedRemovals, diff.Removals())
		})
	}
}
