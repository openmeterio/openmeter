package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Annotations(t *testing.T) {
	t.Run("Clone", func(t *testing.T) {
		src := Annotations{
			"str":   "string",
			"slice": []int{1, 2, 3},
			"map": map[string]string{
				"str": "string",
			},
			"struct": struct {
				name string
				age  int
			}{
				name: "John",
				age:  30,
			},
			"ptrStruct": &struct {
				name string
				age  int
			}{
				name: "John",
				age:  30,
			},
		}

		dst, err := src.Clone()
		require.NoErrorf(t, err, "cloning annotations must not fail")
		assert.Equalf(t, src, dst, "cloned annotations must be equal")

		src["str"] = "new string"
		assert.Equalf(t, "string", dst["str"], "cloned annotations must be equal")

		src["ptrStruct"] = nil
		assert.NotNilf(t, dst["ptrStruct"], "cloned annotations must be equal")
	})

	t.Run("Merge", func(t *testing.T) {
		left := Annotations{
			"str":   "string",
			"slice": []int{1, 2, 3},
			"map": map[string]string{
				"str": "string",
			},
			"struct": struct {
				name string
				age  int
			}{
				name: "John",
				age:  30,
			},
			"ptrStruct": &struct {
				name string
				age  int
			}{
				name: "John",
				age:  30,
			},
		}

		right := Annotations{
			"str":   "string2",
			"slice": []int{3, 2, 1},
			"map": map[string]string{
				"str": "string2",
			},
			"ptrStruct": &struct {
				name string
				age  int
			}{
				name: "John",
				age:  50,
			},
		}

		result, err := left.Merge(right)
		require.NoError(t, err, "merging annotations must not fail")
		require.NotNil(t, result, "result must not be nil")

		expected := Annotations{
			"str":   "string2",
			"slice": []int{3, 2, 1},
			"map": map[string]string{
				"str": "string2",
			},
			"struct": struct {
				name string
				age  int
			}{
				name: "John",
				age:  30,
			},
			"ptrStruct": &struct {
				name string
				age  int
			}{
				name: "John",
				age:  50,
			},
		}
		assert.Equalf(t, expected, result, "merged annotations must be equal")

		left["str"] = "new string"
		right["str"] = "new string2"

		assert.Equalf(t, expected["str"], result["str"], "updating source annotations must not affect the merged annotation")
	})

	t.Run("GetInt", func(t *testing.T) {
		const value = "value"

		tests := []struct {
			name          string
			annotations   Annotations
			expectedValue int
			expectedOk    bool
		}{
			{
				name: "Int",
				annotations: Annotations{
					value: 1,
				},
				expectedValue: 1,
				expectedOk:    true,
			},
			{
				name: "Float64 - whole number",
				annotations: Annotations{
					value: 1.0,
				},
				expectedValue: 1,
				expectedOk:    true,
			},
			{
				name: "Float64",
				annotations: Annotations{
					value: 1.5,
				},
				expectedValue: 0,
				expectedOk:    false,
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				val, ok := test.annotations.GetInt(value)
				assert.Equalf(t, test.expectedValue, val, "value must match")
				assert.Equalf(t, test.expectedOk, ok, "ok must match")
			})
		}
	})
}
