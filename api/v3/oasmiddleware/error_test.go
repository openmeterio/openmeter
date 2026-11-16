package oasmiddleware_test

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/oasmiddleware"
)

func TestToAipError_AllOfInsideParameterReportsEachViolation(t *testing.T) {
	// given: a deep-object query parameter whose property breaks two allOf branches
	schema := openapi3.NewObjectSchema().WithProperty(
		"gt",
		openapi3.NewAllOfSchema(
			openapi3.NewStringSchema().WithPattern(`^[0-9]+$`),
			openapi3.NewStringSchema().WithMinLength(10),
		),
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

	// then: each violated rule is reported as a query error, not as the allOf wrapper
	require.Equal(t, []apierrors.InvalidParameter{
		{
			Field:  "gt",
			Rule:   "pattern",
			Reason: `string doesn't match the regular expression "^[0-9]+$"`,
			Source: apierrors.InvalidParamSourceQuery,
		},
		{
			Field:  "gt",
			Rule:   "min_length",
			Reason: "minimum string length is 10",
			Source: apierrors.InvalidParamSourceQuery,
		},
	}, invalidParams)
}
