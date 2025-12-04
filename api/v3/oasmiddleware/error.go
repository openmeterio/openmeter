package oasmiddleware

import (
	"errors"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
)

var oasRuleToAip = map[string]string{
	"minLength": "min_length",
	"maxLength": "max_length",
	"minItems":  "min_items",
	"maxItems":  "max_items",
}

func ToAipError(me openapi3.MultiError) []apierrors.InvalidParameter {
	return aipMapper(me, nil)
}

func aipMapper(me openapi3.MultiError, parent *apierrors.InvalidParameter) []apierrors.InvalidParameter {
	var ipErrs []apierrors.InvalidParameter
	for _, err := range me {
		var i *apierrors.InvalidParameter
		if parent != nil {
			i = parent
		} else {
			i = &apierrors.InvalidParameter{}
		}
		switch err := err.(type) {
		case *openapi3.SchemaError:
			i.Reason = err.Reason
			ipErrs = append(ipErrs, invalidParamFromSchemaError(err, i))
		case *openapi3filter.RequestError:
			if err.Parameter != nil {
				if err.Parameter.Name != "" {
					i.Field = err.Parameter.Name
				}
				if err.Parameter.In != "" {
					i.Source = apierrors.ToInvalid(err.Parameter.In)
				}
				if err.Parameter.Required {
					i.Rule = "required"
				}
			}
			i.Reason = err.Reason
			if err.Reason == "" || err.RequestBody != nil {
				i.Reason = err.Error()
			}

			if err, ok := err.Err.(openapi3.MultiError); ok {
				ipErrs = append(ipErrs, aipMapper(err, i)...)
				continue
			}

			if err, ok := err.Err.(*openapi3.SchemaError); ok {
				i.Choices = make([]string, 0)
				if err.SchemaField == "enum" {
					i.Rule = "enum"
					for _, v := range err.Schema.Enum {
						i.Choices = append(i.Choices, fmt.Sprintf("%v", v))
					}
					i.Reason = fmt.Sprintf("must be one of: [%s]", strings.Join(i.Choices, ","))
				} else if err.SchemaField == "oneOf" {
					ipErrs = append(ipErrs, collectFromSchemaError(err)...)
					continue
				}
			}
			ipErrs = append(ipErrs, *i)
		}
	}
	return ipErrs
}

// collectFromSchemaError looks at schemaErr.Origin. If there are deeper
// child errors (via unwrapOriginError), it returns those. Otherwise, it
// returns a single InvalidParameter built from schemaErr itself.
func collectFromSchemaError(se *openapi3.SchemaError) []apierrors.InvalidParameter {
	childParams := unwrapOriginError(se)
	if len(childParams) == 0 {
		return []apierrors.InvalidParameter{
			invalidParamFromSchemaError(se, nil),
		}
	}
	return childParams
}

// unwrapOriginError traverses schemaErr.Origin (which may be a wrapped multiErrorForOneOf)
// and returns a flat slice of InvalidParameter entries for each underlying *SchemaError.
func unwrapOriginError(schemaErr *openapi3.SchemaError) []apierrors.InvalidParameter {
	if schemaErr == nil || schemaErr.Origin == nil {
		return nil
	}

	// 1) First, try to pull out a MultiError (or multiErrorForOneOf) from the wrapper chain.
	var me openapi3.MultiError
	if errors.As(schemaErr.Origin, &me) {
		var result []apierrors.InvalidParameter
		for _, subErr := range me {
			var subSE *openapi3.SchemaError
			if errors.As(subErr, &subSE) {
				result = append(result, collectFromSchemaError(subSE)...)
			}
		}
		return result
	}

	// 2) If there are no multi-errors and Origin wraps another *SchemaError somewhere in its chain, dive into that.
	var innerSE *openapi3.SchemaError
	if errors.As(schemaErr.Origin, &innerSE) {
		return collectFromSchemaError(innerSE)
	}

	// 3) If we reach here, Origin was neither a nested *SchemaError nor a MultiError.
	return nil
}

func invalidParamFromSchemaError(
	schemaErr *openapi3.SchemaError,
	parent *apierrors.InvalidParameter,
) apierrors.InvalidParameter {
	var ip *apierrors.InvalidParameter
	if parent != nil {
		ip = parent
	} else {
		ip = &apierrors.InvalidParameter{
			Reason: schemaErr.Reason,
		}
	}
	if rule, ok := oasRuleToAip[schemaErr.SchemaField]; ok {
		ip.Rule = rule
	} else {
		ip.Rule = schemaErr.SchemaField
	}
	if path := schemaErr.JSONPointer(); len(path) > 0 {
		ip.Field = strings.Join(path, ".")
	}
	return *ip
}
