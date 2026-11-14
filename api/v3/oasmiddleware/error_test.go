package oasmiddleware_test

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/oasmiddleware"
)

func TestToAipError_AllOfInsideParameterKeepsSource(t *testing.T) {
	// given: a deep-object query parameter with a property that wraps a pattern in allOf
	schema := openapi3.NewObjectSchema().WithProperty(
		"gt",
		openapi3.NewAllOfSchema(openapi3.NewStringSchema().WithPattern(`^[0-9]+$`)),
	)
	schemaErr := schema.VisitJSON(map[string]any{"gt": "100."}, openapi3.MultiErrors())
	require.IsType(t, openapi3.MultiError{}, schemaErr)

	// when: the validator reports the parameter failure
	invalidParams := oasmiddleware.ToAipError(openapi3.MultiError{
		&openapi3filter.RequestError{
			Parameter: &openapi3.Parameter{Name: "filter", In: openapi3.ParameterInQuery},
			Err:       schemaErr,
		},
	})

	// then: the violated rule is reported as a query error, not as the allOf wrapper
	require.Len(t, invalidParams, 1)
	require.Equal(t, "gt", invalidParams[0].Field)
	require.Equal(t, apierrors.InvalidParamSourceQuery, invalidParams[0].Source)
	require.Equal(t, "pattern", invalidParams[0].Rule)
}
