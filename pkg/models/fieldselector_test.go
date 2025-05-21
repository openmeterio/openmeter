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
		name           string
		selector       FieldSelector
		expectedString string
	}{
		{
			name:           "empty",
			selector:       FieldSelector{},
			expectedString: "",
		},
		{
			name:           "field",
			selector:       NewFieldSelector("test"),
			expectedString: "test",
		},
		{
			name: "with attribute",
			selector: NewFieldSelector("test",
				FieldValue{"key", "value"},
			),
			expectedString: "test[?(@.key=='value')]",
		},
		{
			name: "with multiple attributes",
			selector: NewFieldSelector("test",
				FieldValue{"key", "value"},
				FieldValue{"key2", "value2"},
			),
			expectedString: "test[?(@.key=='value' && @.key2=='value2')]",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.selector.JSONPath()
			assert.Equalf(t, test.expectedString, actual, "selector must be equal")
		})
	}
}

func TestFieldSelectors(t *testing.T) {
	tests := []struct {
		name           string
		selector       FieldSelectors
		expectedString string
	}{
		{
			name:           "empty",
			selector:       FieldSelectors{},
			expectedString: "",
		},
		{
			name: "single",
			selector: FieldSelectors{
				NewFieldSelector("test"),
			},
			expectedString: "$.test",
		},
		{
			name: "multiple",
			selector: FieldSelectors{
				NewFieldSelector("test1"),
				NewFieldSelector("test2",
					FieldValue{"key", "value"},
				),
				NewFieldSelector("test3",
					FieldValue{"key", "value"},
					FieldValue{"key2", "value2"},
				),
				NewFieldSelector("test4"),
			},
			expectedString: "$.test1.test2[?(@.key=='value')].test3[?(@.key=='value' && @.key2=='value2')].test4",
		},
		{
			name: "merge",
			selector: FieldSelectors{
				NewFieldSelector("test3",
					FieldValue{"key", "value"},
					FieldValue{"key2", "value2"},
				),
				NewFieldSelector("test4"),
			}.WithPrefix(FieldSelectors{
				NewFieldSelector("test1"),
				NewFieldSelector("test2",
					FieldValue{"key", "value"},
				),
			}),
			expectedString: "$.test1.test2[?(@.key=='value')].test3[?(@.key=='value' && @.key2=='value2')].test4",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.selector.JSONPath()
			assert.Equalf(t, test.expectedString, actual, "selector must be equal")
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

func TestFieldSelectors_JSONPath(t *testing.T) {
	tests := []struct {
		name           string
		selector       FieldSelectors
		expectedResult string
	}{
		{
			name: "empty",
			selector: FieldSelectors{
				NewFieldSelector("store"),
				NewFieldSelector("book", FieldValue{"category", "reference"}),
				NewFieldSelector("author"),
			},
			expectedResult: `["Nigel Rees"]`,
		},
		{
			name: "multi",
			selector: FieldSelectors{
				NewFieldSelector("store"),
				NewFieldSelector("book",
					FieldValue{"category", "fiction"},
					FieldValue{"isbn", "0-395-19395-8"},
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

func TestFieldSelectors_String(t *testing.T) {
	tests := []struct {
		name           string
		selector       FieldSelectors
		expectedResult string
	}{
		{
			name: "empty",
			selector: FieldSelectors{
				NewFieldSelector("store"),
				NewFieldSelector("book", FieldValue{"category", "reference"}),
				NewFieldSelector("author"),
			},
			expectedResult: `/store/book[category='reference']/author`,
		},
		{
			name: "multi",
			selector: FieldSelectors{
				NewFieldSelector("store"),
				NewFieldSelector("book",
					FieldValue{"category", "fiction"},
					FieldValue{"isbn", "0-395-19395-8"},
				),
				NewFieldSelector("author"),
			},
			expectedResult: `/store/book[category='fiction', isbn='0-395-19395-8']/author`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q := test.selector.String()
			t.Logf("selector: %s", q)

			assert.Equalf(t, test.expectedResult, q, "must be equal")
		})
	}
}
