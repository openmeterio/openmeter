package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldPath(t *testing.T) {
	tests := []struct {
		name         string
		parts        []string
		expectedPath string
	}{
		{
			name:         "empty",
			parts:        []string{},
			expectedPath: "",
		},
		{
			name:         "single",
			parts:        []string{"part1"},
			expectedPath: "/part1",
		},
		{
			name:         "multiple",
			parts:        []string{"part1", "part2"},
			expectedPath: "/part1/part2",
		},
		{
			name:         "unclean",
			parts:        []string{"//part1/", "/./part2/"},
			expectedPath: "/part1/part2",
		},
		{
			name:         "traverse",
			parts:        []string{"/part1", "/part2", "/../part3"},
			expectedPath: "/part1/part3",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := FieldPathFromParts(test.parts...)
			assert.Equalf(t, test.expectedPath, actual, "path must be equal")
		})
	}
}
