package models

import (
	"encoding/json"
	"testing"

	"github.com/bhmj/jsonslice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/treex"
)

func TestFieldSelector(t *testing.T) {
	tests := []struct {
		name             string
		selector         *FieldDescriptor
		expectedString   string
		expectedJSONPath string
	}{
		{
			name:             "field",
			selector:         NewFieldSelector("test"),
			expectedString:   `test`,
			expectedJSONPath: `$.test`,
		},
		{
			name: "with attribute",
			selector: NewFieldSelector("test").
				WithExpression(
					NewFieldAttrValue("key", "value"),
				),
			expectedString:   `test[key=value]`,
			expectedJSONPath: `$.test[?(@.key=='value')]`,
		},
		{
			name: "with array index",
			selector: NewFieldSelector("test").
				WithExpression(
					NewFieldArrIndex(1),
				),
			expectedString:   `test[1]`,
			expectedJSONPath: `$.test[1]`,
		},
		{
			name: "with multiple attributes",
			selector: NewFieldSelector("test").
				WithExpression(NewMultiFieldAttrValue(
					NewFieldAttrValue("key", "value"),
					NewFieldAttrValue("key2", "value2"),
				)),
			expectedString:   `test[key=value, key2=value2]`,
			expectedJSONPath: `$.test[?(@.key=='value' && @.key2=='value2')]`,
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
		selector         *FieldDescriptor
		expectNil        bool
		expectedString   string
		expectedJSONPath string
	}{
		{
			name:             "single",
			selector:         NewFieldSelectorGroup(NewFieldSelector("test")),
			expectedString:   `test`,
			expectedJSONPath: `$.test`,
		},
		{
			name: "multiple",
			selector: NewFieldSelectorGroup(
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
			selector: NewFieldSelectorGroup(
				NewFieldSelector("test3").
					WithExpression(
						NewMultiFieldAttrValue(
							NewFieldAttrValue("key", "value"),
							NewFieldAttrValue("key2", "value2"),
						),
					),
				NewFieldSelector("test4"),
			).WithPrefix(NewFieldSelectorGroup(
				NewFieldSelector("test1"),
				NewFieldSelector("test2").
					WithExpression(
						NewFieldAttrValue("key2", "value2"),
					),
			)),
			expectedString:   `test1.test2[key2=value2].test3[key=value, key2=value2].test4`,
			expectedJSONPath: `$.test1.test2[?(@.key2=='value2')].test3[?(@.key=='value' && @.key2=='value2')].test4`,
		},
		{
			name: "empty groupings",
			selector: NewFieldSelectorGroup(
				NewFieldSelectorGroup(
					NewFieldSelectorGroup(),
					NewFieldSelectorGroup(),
				),
				NewFieldSelectorGroup(
					NewFieldSelectorGroup(),
					NewFieldSelectorGroup(),
					NewFieldSelectorGroup(
						NewFieldSelectorGroup(),
						NewFieldSelectorGroup(),
					),
				),
			),
			expectNil:        true,
			expectedString:   "",
			expectedJSONPath: "",
		},
		{
			name: "multiple prefix",
			selector: NewFieldSelectorGroup(
				NewFieldSelector("test5").
					WithExpression(
						NewMultiFieldAttrValue(
							NewFieldAttrValue("key", "value"),
							NewFieldAttrValue("key2", "value2"),
						),
					),
				NewFieldSelector("test6"),
			).WithPrefix(NewFieldSelectorGroup(
				NewFieldSelector("test3"),
				NewFieldSelector("test4").
					WithExpression(
						NewFieldAttrValue("key2", "value2"),
					),
			)).WithPrefix(NewFieldSelectorGroup(
				NewFieldSelector("test1").WithExpression(WildCard),
				NewFieldSelector("test2"),
			).WithPrefix(
				NewFieldSelector("test0").WithExpression(NewFieldAttrValue("attr1", "value1")),
			)),
			expectedString:   `test0[attr1=value1].test1.test2.test3.test4[key2=value2].test5[key=value, key2=value2].test6`,
			expectedJSONPath: `$.test0[?(@.attr1=='value1')].test1[*].test2.test3.test4[?(@.key2=='value2')].test5[?(@.key=='value' && @.key2=='value2')].test6`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.expectNil {
				require.Nil(t, test.selector, "selector must be nil")
				return
			} else {
				require.NotNil(t, test.selector, "selector must not be nil")
			}

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
		selector       *FieldDescriptor
		expectedResult string
	}{
		{
			name: "single",
			selector: NewFieldSelectorGroup(
				NewFieldSelector("store"),
				NewFieldSelector("book").WithExpression(
					NewFieldAttrValue("category", "reference"),
				),
				NewFieldSelector("author"),
			),
			expectedResult: `["Nigel Rees"]`,
		},
		{
			name: "multi",
			selector: NewFieldSelectorGroup(
				NewFieldSelector("store"),
				NewFieldSelector("book").WithExpression(
					NewMultiFieldAttrValue(
						NewFieldAttrValue("category", "fiction"),
						NewFieldAttrValue("isbn", "0-395-19395-8"),
					),
				),
				NewFieldSelector("author"),
			),
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

func TestFieldSelectorGroup_WithAttributes(t *testing.T) {
	t.Run("Should add and retrieve new set of attributes", func(t *testing.T) {
		selector := NewFieldSelectorGroup(NewFieldSelector("test"))

		desc := selector.WithAttributes(Attributes{
			"attr1": "value1",
		})

		assert.Equal(t, Attributes{
			"attr1": "value1",
		}, desc.GetAttributes(), "attributes must match")
	})

	t.Run("Should merge with existing attributes", func(t *testing.T) {
		selector := NewFieldSelectorGroup(NewFieldSelector("test"))
		desc := selector.WithAttributes(Attributes{
			"attr1": "value1",
		})
		desc = desc.WithAttributes(Attributes{
			"attr2": "value2",
		})

		assert.Equal(t, Attributes{
			"attr1": "value1",
			"attr2": "value2",
		}, desc.GetAttributes(), "attributes must match")
	})

	t.Run("Should keep attributes on child when prefixing", func(t *testing.T) {
		sel := NewFieldSelector("someField").WithAttributes(Attributes{
			"attr1": "value1",
		})

		pref := NewFieldSelector("myPrefix").WithAttributes(Attributes{
			"attr2": "value2",
		})

		full := sel.WithPrefix(pref)

		// we'll compare by field as trees use cloned values
		visited := make(map[string]bool)

		require.NoError(t, full.Tree(func(tr *FieldDescriptorTree) error {
			return tr.DFS(func(n *treex.Node[*FieldDescriptor]) (bool, error) {
				desc := n.Value()

				if desc != nil {
					visited[desc.field] = true
				}

				if desc.field == sel.field {
					assert.Equal(t, Attributes{
						"attr1": "value1",
					}, desc.GetAttributes(), "attributes must match, %+v", desc)
				}

				if desc.field == pref.field {
					assert.Equal(t, Attributes{
						"attr2": "value2",
					}, desc.GetAttributes(), "attributes must match, %+v", desc)
				}

				return false, nil
			})
		}))

		assert.True(t, visited[sel.field], "sel must be visited")
		assert.True(t, visited[pref.field], "pref must be visited")
	})
}

func TestMethodArgumentsStayUnchanged(t *testing.T) {
	selector := NewFieldSelector("test")
	selector2 := NewFieldSelector("test2")
	selector3 := NewFieldSelector("test3")

	prefixed := selector.WithPrefix(selector2)
	prefixed2 := prefixed.WithPrefix(selector3)

	assert.Equal(t, "$.test", selector.JSONPath())
	assert.Equal(t, "$.test2", selector2.JSONPath())
	assert.Equal(t, "$.test2.test", prefixed.JSONPath())
	assert.Equal(t, "$.test3.test2.test", prefixed2.JSONPath())
}
