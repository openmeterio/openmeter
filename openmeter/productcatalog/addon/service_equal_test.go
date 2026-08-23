package addon

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestUpdateAddonInputEqualDescriptionPresence(t *testing.T) {
	stored := Addon{}

	tests := []struct {
		name        string
		input       *string
		stored      *string
		expectEqual bool
	}{
		{name: "omitted equals absent", input: nil, stored: nil, expectEqual: true},
		{name: "omitted must clear a stored value", input: nil, stored: lo.ToPtr("desc"), expectEqual: false},
		{name: "omitted must clear a stored empty string", input: nil, stored: lo.ToPtr(""), expectEqual: false},
		{name: "matching values are equal", input: lo.ToPtr("desc"), stored: lo.ToPtr("desc"), expectEqual: true},
		{name: "differing values are not equal", input: lo.ToPtr("new"), stored: lo.ToPtr("old"), expectEqual: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := stored
			a.Description = tc.stored

			i := UpdateAddonInput{Description: tc.input}

			assert.Equal(t, tc.expectEqual, i.Equal(a))
		})
	}
}
