package models

import (
	"encoding/json"
	"testing"

	"github.com/bhmj/jsonslice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldSelector(t *testing.T) {
	tests := []struct {
		name             string
		selector         FieldSelector
		expectedString   string
		expectedJSONPath string
	}{
		{
			name:             "empty",
			selector:         FieldSelector{},
			expectedString:   "",
			expectedJSONPath: "",
		},
		{
			name:             "field",
			selector:         NewFieldSelector("test"),
			expectedString:   `test`,
			expectedJSONPath: `test`,
		},
		{
			name: "with attribute",
			selector: NewFieldSelector("test").
				WithExpression(
					NewFieldAttrValue("key", "value"),
				),
			expectedString:   `test[key=value]`,
			expectedJSONPath: `test[?(@.key=='value')]`,
		},
		{
			name: "with array index",
			selector: NewFieldSelector("test").
				WithExpression(
					NewFieldArrIndex(1),
				),
			expectedString:   `test[1]`,
			expectedJSONPath: `test[1]`,
		},
		{
			name: "with multiple attributes",
			selector: NewFieldSelector("test").
				WithExpression(NewMultiFieldAttrValue(
					NewFieldAttrValue("key", "value"),
					NewFieldAttrValue("key2", "value2"),
				)),
			expectedString:   `test[key=value, key2=value2]`,
			expectedJSONPath: `test[?(@.key=='value' && @.key2=='value2')]`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualString := test.selector.String()
			assert.Equalf(t, test.expectedString, actualString, "string must be equal")

			actualJSONPath := test.selector.JSONPath()
			assert.Equalf(t, test.expectedJSONPath, actualJSONPath, "JSONPath must be equal")
		})
	}
}

func TestFieldSelectors(t *testing.T) {
	tests := []struct {
		name             string
		selector         FieldSelectors
		expectedString   string
		expectedJSONPath string
	}{
		{
			name:     "empty",
			selector: FieldSelectors{},
		},
		{
			name:             "single",
			selector:         NewFieldSelectors(NewFieldSelector("test")),
			expectedString:   `test`,
			expectedJSONPath: `$.test`,
		},
		{
			name: "multiple",
			selector: NewFieldSelectors(
				NewFieldSelector("test1").WithExpression(WildCard),
				NewFieldSelector("test2").
					WithExpression(
						NewFieldAttrValue("key", "value"),
					),
				NewFieldSelector("test3").
					WithExpression(
						NewMultiFieldAttrValue(
							NewFieldAttrValue("key", "value"),
							NewFieldAttrValue("key2", "value2"),
						),
					),
				NewFieldSelector("test4").WithExpression(
					NewFieldArrIndex(0),
				),
			),
			expectedString:   `test1.test2[key=value].test3[key=value, key2=value2].test4[0]`,
			expectedJSONPath: `$.test1[*].test2[?(@.key=='value')].test3[?(@.key=='value' && @.key2=='value2')].test4[0]`,
		},
		{
			name: "prefix",
			selector: NewFieldSelectors(
				NewFieldSelector("test3").
					WithExpression(
						NewMultiFieldAttrValue(
							NewFieldAttrValue("key", "value"),
							NewFieldAttrValue("key2", "value2"),
						),
					),
				NewFieldSelector("test4"),
			).WithPrefix(NewFieldSelectors(
				NewFieldSelector("test1"),
				NewFieldSelector("test2").
					WithExpression(
						NewFieldAttrValue("key2", "value2"),
					),
			)),
			expectedString:   `test1.test2[key2=value2].test3[key=value, key2=value2].test4`,
			expectedJSONPath: `$.test1.test2[?(@.key2=='value2')].test3[?(@.key=='value' && @.key2=='value2')].test4`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualString := test.selector.String()
			assert.Equalf(t, test.expectedString, actualString, "string must be equal")

			actualJSONPath := test.selector.JSONPath()
			assert.Equalf(t, test.expectedJSONPath, actualJSONPath, "JSONPath must be equal")
		})
	}
}

// See: https://www.rfc-editor.org/rfc/rfc9535.html
//
//nolint:misspell
var testJSON = `
{
	"store": {
		"book": [
			  { "category": "reference",
				"author": "Nigel Rees",
				"title": "Sayings of the Century",
				"price": 8.95
			  },
			  { "category": "fiction",
				"author": "Evelyn Waugh",
				"title": "Sword of Honour",
				"price": 12.99
			  },
			  { "category": "fiction",
				"author": "Herman Melville",
				"title": "Moby Dick",
				"isbn": "0-553-21311-3",
				"price": 8.99
			  },
			  { "category": "fiction",
				"author": "J. R. R. Tolkien",
				"title": "The Lord of the Rings",
				"isbn": "0-395-19395-8",
				"price": 22.99
			  }
		],
		"bicycle": {
			  "color": "red",
			  "price": 399
		}
  	}
}
`

func TestFieldSelectors_JSONPathQuery(t *testing.T) {
	tests := []struct {
		name           string
		selector       FieldSelectors
		expectedResult string
	}{
		{
			name: "single",
			selector: FieldSelectors{
				NewFieldSelector("store"),
				NewFieldSelector("book").WithExpression(
					NewFieldAttrValue("category", "reference"),
				),
				NewFieldSelector("author"),
			},
			expectedResult: `["Nigel Rees"]`,
		},
		{
			name: "multi",
			selector: FieldSelectors{
				NewFieldSelector("store"),
				NewFieldSelector("book").WithExpression(
					NewMultiFieldAttrValue(
						NewFieldAttrValue("category", "fiction"),
						NewFieldAttrValue("isbn", "0-395-19395-8"),
					),
				),
				NewFieldSelector("author"),
			},
			expectedResult: `["J. R. R. Tolkien"]`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ok := json.Valid([]byte(testJSON))
			require.Truef(t, ok, "JSON must be valid")

			q := test.selector.JSONPath()
			t.Logf("selector: %s", q)

			v, err := jsonslice.Get([]byte(testJSON), q)
			require.NoError(t, err, "querying JSON must not fail")

			t.Logf("result: %s", v)

			assert.Equalf(t, test.expectedResult, string(v), "must be equal")
		})
	}
}
