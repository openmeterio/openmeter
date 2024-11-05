package plan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetadataEqual(t *testing.T) {
	tests := []struct {
		Name string

		Left  map[string]string
		Right map[string]string

		ExpectedResult bool
	}{
		{
			Name: "Equal",
			Left: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			Right: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			ExpectedResult: true,
		},
		{
			Name: "Left diff",
			Left: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			Right: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			ExpectedResult: false,
		},
		{
			Name: "Right diff",
			Left: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			Right: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			ExpectedResult: false,
		},
		{
			Name: "Complete diff",
			Left: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			Right: map[string]string{
				"key4": "value4",
				"key5": "value5",
				"key6": "value6",
			},
			ExpectedResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			result := MetadataEqual(test.Left, test.Right)

			assert.Equal(t, test.ExpectedResult, result)
		})
	}
}
