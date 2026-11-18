package oasmiddleware_test

import (
	"math"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/samber/lo"
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
			Field:  "filter[gt]",
			Rule:   "pattern",
			Reason: `string doesn't match the regular expression "^[0-9]+$"`,
			Source: apierrors.InvalidParamSourceQuery,
		},
		{
			Field:   "filter[gt]",
			Rule:    "min_length",
			Reason:  "minimum string length is 10",
			Source:  apierrors.InvalidParamSourceQuery,
			Minimum: lo.ToPtr(10),
		},
	}, invalidParams)
}

func TestToAipError_NumericBoundsInsideParameter(t *testing.T) {
	// given: a deep-object page parameter whose size and number are out of range
	schema := openapi3.NewObjectSchema().
		WithProperty("size", openapi3.NewIntegerSchema().WithMin(1).WithMax(1000)).
		WithProperty("number", openapi3.NewIntegerSchema().WithMin(1))
	schemaErr := schema.VisitJSON(map[string]any{"size": 1001.0, "number": 0.0}, openapi3.MultiErrors())
	require.IsType(t, openapi3.MultiError{}, schemaErr)

	// when: the validator reports the parameter failure
	invalidParams := oasmiddleware.ToAipError(openapi3.MultiError{
		&openapi3filter.RequestError{
			Parameter: &openapi3.Parameter{Name: "page", In: openapi3.ParameterInQuery},
			Err:       schemaErr,
		},
	})

	// then: each error names the query key and carries the published rule and limit
	require.ElementsMatch(t, []apierrors.InvalidParameter{
		{
			Field:   "page[size]",
			Rule:    "max",
			Reason:  "number must be at most 1000",
			Source:  apierrors.InvalidParamSourceQuery,
			Maximum: lo.ToPtr(1000),
		},
		{
			Field:   "page[number]",
			Rule:    "min",
			Reason:  "number must be at least 1",
			Source:  apierrors.InvalidParamSourceQuery,
			Minimum: lo.ToPtr(1),
		},
	}, invalidParams)
}

func TestToAipError_BoundOutsideIntRangeIsOmitted(t *testing.T) {
	// given: a schema maximum of 2^63, which does not fit in an int
	schema := openapi3.NewObjectSchema().
		WithProperty("size", openapi3.NewFloat64Schema().WithMax(math.Exp2(63)))
	schemaErr := schema.VisitJSON(map[string]any{"size": math.Exp2(64)}, openapi3.MultiErrors())
	require.IsType(t, openapi3.MultiError{}, schemaErr)

	// when: the validator reports the parameter failure
	invalidParams := oasmiddleware.ToAipError(openapi3.MultiError{
		&openapi3filter.RequestError{
			Parameter: &openapi3.Parameter{Name: "page", In: openapi3.ParameterInQuery},
			Err:       schemaErr,
		},
	})

	// then: the rule is reported without a wrapped-around limit
	require.Len(t, invalidParams, 1)
	require.Equal(t, "page[size]", invalidParams[0].Field)
	require.Equal(t, "max", invalidParams[0].Rule)
	require.Nil(t, invalidParams[0].Maximum)
}
