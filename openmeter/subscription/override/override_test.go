package override_test

import (
	"testing"

	"github.com/openmeterio/openmeter/openmeter/subscription/override"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

type data struct {
	name      string
	otherProp *int
}

var _ override.UniquelyComparable = data{}

func (d data) UniqBy() string {
	return d.name
}

func TestOverrides(t *testing.T) {
	tt := []struct {
		name      string
		base      []data
		overrides []override.Override[data]
		expected  []data
	}{
		{
			name:      "Should output nothing if both inputs are nothing",
			base:      []data{},
			overrides: []override.Override[data]{},
			expected:  []data{},
		},
		{
			name:      "Should output nothing if both inputs are nil",
			base:      nil,
			overrides: nil,
			expected:  []data{},
		},
		{
			name: "Should output nothing if only remove overrides are present",
			base: []data{},
			overrides: []override.Override[data]{
				{
					Action: override.OverrideActionRemove,
					Value:  data{name: "a"},
				},
			},
			expected: []data{},
		},
		{
			name: "Should delete the value if remove override is present",
			base: []data{
				{name: "a"},
			},
			overrides: []override.Override[data]{
				{
					Action: override.OverrideActionRemove,
					Value:  data{name: "a"},
				},
			},
			expected: []data{},
		},
		{
			name: "Should output add override",
			base: []data{},
			overrides: []override.Override[data]{
				{
					Action: override.OverrideActionAdd,
					Value:  data{name: "a"},
				},
			},
			expected: []data{
				{name: "a"},
			},
		},
		{
			name: "Should output add override in addition to base values",
			base: []data{
				{name: "b"},
			},
			overrides: []override.Override[data]{
				{
					Action: override.OverrideActionAdd,
					Value:  data{name: "a"},
				},
			},
			expected: []data{
				// Base value is present first
				{name: "b"},
				{name: "a"},
			},
		},
		{
			name: "Should override value",
			base: []data{
				{name: "a", otherProp: lo.ToPtr(2)},
				{name: "b"},
			},
			overrides: []override.Override[data]{
				{
					Action: override.OverrideActionAdd,
					Value:  data{name: "a", otherProp: lo.ToPtr(3)},
				},
			},
			expected: []data{
				{name: "a", otherProp: lo.ToPtr(3)},
				{name: "b"},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			res := override.ApplyOverrides(tc.base, tc.overrides)
			assert.Equal(t, tc.expected, res)
		})
	}
}
