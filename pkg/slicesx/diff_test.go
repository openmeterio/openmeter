package slicesx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiff(t *testing.T) {
	tests := []struct {
		Name string

		Base []string
		New  []string

		ExpectedAdditions []string
		ExpectedRemovals  []string
	}{
		{
			Name: "No change",
			Base: []string{
				"diff-1",
				"diff-2",
			},
			New: []string{
				"diff-2",
				"diff-1",
			},
		},
		{
			Name: "Add new item",
			Base: []string{
				"diff-1",
				"diff-2",
				"diff-3",
			},
			New: []string{
				"diff-2",
				"diff-1",
			},
			ExpectedAdditions: []string{
				"diff-3",
			},
		},
		{
			Name: "Remove old item",
			Base: []string{
				"diff-2",
			},
			New: []string{
				"diff-2",
				"diff-1",
			},
			ExpectedRemovals: []string{
				"diff-1",
			},
		},
		{
			Name: "Add and remove items",
			Base: []string{
				"diff-1",
				"diff-3",
			},
			New: []string{
				"diff-2",
				"diff-1",
			},
			ExpectedAdditions: []string{
				"diff-3",
			},
			ExpectedRemovals: []string{
				"diff-2",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			diff := NewDiff(test.Base, test.New)

			assert.ElementsMatch(t, test.ExpectedAdditions, diff.Additions())
			assert.ElementsMatch(t, test.ExpectedRemovals, diff.Removals())
		})
	}
}
