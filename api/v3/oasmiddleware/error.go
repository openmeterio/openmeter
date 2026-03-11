package oasmiddleware

import (
	"strings"

	validatorerrors "github.com/pb33f/libopenapi-validator/errors"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
)

var oasRuleToAip = map[string]string{
	"minLength": "min_length",
	"maxLength": "max_length",
	"minItems":  "min_items",
	"maxItems":  "max_items",
}

// ToAipErrorFromLibopenapi converts libopenapi ValidationErrors to AIP InvalidParameters.
func ToAipErrorFromLibopenapi(errs []*validatorerrors.ValidationError) []apierrors.InvalidParameter {
	var ipErrs []apierrors.InvalidParameter
	for _, ve := range errs {
		if ve == nil {
			continue
		}
		ip := apierrors.InvalidParameter{
			Field:  ve.ParameterName,
			Reason: ve.Reason,
			Rule:   ruleFromValidationError(ve),
			Source: sourceFromValidationError(ve),
		}
		if ip.Field == "" && len(ve.SchemaValidationErrors) > 0 {
			// Use field path from schema errors if no parameter name
			sve := ve.SchemaValidationErrors[0]
			if sve.FieldName != "" {
				ip.Field = sve.FieldName
			} else if sve.FieldPath != "" {
				ip.Field = strings.TrimPrefix(strings.TrimPrefix(sve.FieldPath, "$."), "body.")
			}
		}
		if len(ve.SchemaValidationErrors) > 0 {
			// Extract enum choices from schema validation errors if applicable
			for _, sve := range ve.SchemaValidationErrors {
				if sve.Reason != "" && ip.Reason == "" {
					ip.Reason = sve.Reason
				}
			}
		}
		ipErrs = append(ipErrs, ip)
	}
	return ipErrs
}

func ruleFromValidationError(ve *validatorerrors.ValidationError) string {
	if r, ok := oasRuleToAip[ve.ValidationSubType]; ok {
		return r
	}
	if ve.ValidationSubType != "" {
		return ve.ValidationSubType
	}
	if ve.ValidationType != "" {
		return ve.ValidationType
	}
	return ""
}

func sourceFromValidationError(ve *validatorerrors.ValidationError) apierrors.InvalidParameterSource {
	// libopenapi uses: path, query, header, cookie for parameter validation
	switch strings.ToLower(ve.ValidationType) {
	case "path":
		return apierrors.InvalidParamSourcePath
	case "query":
		return apierrors.InvalidParamSourceQuery
	case "header":
		return apierrors.InvalidParamSourceHeader
	case "requestbody", "schema":
		return apierrors.InvalidParamSourceBody
	}
	switch strings.ToLower(ve.ValidationSubType) {
	case "path":
		return apierrors.InvalidParamSourcePath
	case "query":
		return apierrors.InvalidParamSourceQuery
	case "header":
		return apierrors.InvalidParamSourceHeader
	case "requestbody", "schema":
		return apierrors.InvalidParamSourceBody
	}
	return apierrors.InvalidParamSourceBody
}
