package gobldriver

import (
	"fmt"
	"sync"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/schema"
	"github.com/invopop/validation"
	"github.com/samber/lo"
)

type ValidationErrorJSON struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

// ValidationErrorsComplement is a struct that contains validation errors and will be appended to the
// Invoice's Complements field
type ValidationErrorsComplement struct {
	// Errors is a map of field names (JSON Path) to validation errors
	Fields map[string][]ValidationErrorJSON `json:"fields"`
	Global []ValidationErrorJSON            `json:"global"`
}

func (v ValidationErrorsComplement) HasErrors() bool {
	return len(v.Fields) > 0 || len(v.Global) > 0
}

const (
	openmeterSchemaBase = schema.ID("https://openmeter.io/schema/")
)

var goblSchemaRegistrationOnce sync.Once

func registerComplementSchema() {
	goblSchemaRegistrationOnce.Do(func() {
		schema.Register(openmeterSchemaBase, ValidationErrorsComplement{})
	})
}

func NewValidationErrorsComplement(vErrors []error) (ValidationErrorsComplement, error) {
	registerComplementSchema()

	if vErrors == nil {
		return ValidationErrorsComplement{}, nil
	}

	out := ValidationErrorsComplement{
		Fields: make(map[string][]ValidationErrorJSON),
	}

	for _, vErr := range vErrors {
		if vErr == nil {
			continue
		}

		if vFieldErrors, ok := lo.ErrorsAs[validation.Errors](vErr); ok {
			errorsToFieldList(vFieldErrors, out.Fields)
		} else if vGlobalError, ok := lo.ErrorsAs[validation.Error](vErr); ok {
			out.Global = append(out.Global, ValidationErrorJSON{
				Code:    vGlobalError.Code(),
				Message: vGlobalError.Message(),
			})
		} else {
			out.Global = append(out.Global, ValidationErrorJSON{
				Message: vErr.Error(),
			})
		}
	}

	return out, nil
}

func LookupValidationErrors(in *bill.Invoice) (ValidationErrorsComplement, error) {
	errorsSchemaID := schema.Lookup(ValidationErrorsComplement{})

	for _, schemaObject := range in.Complements {
		if schemaObject.Schema == errorsSchemaID {
			vErrors, ok := schemaObject.Instance().(ValidationErrorsComplement)
			if !ok {
				return ValidationErrorsComplement{}, fmt.Errorf("unexpected type for validation errors complement: %T", schemaObject.Instance())
			}

			return vErrors, nil
		}
	}

	return ValidationErrorsComplement{}, nil
}

func errorsToFieldList(rootErr validation.Errors, target map[string][]ValidationErrorJSON) {
	if rootErr == nil {
		return
	}

	errorsToFieldListImpl(rootErr, "", target)
}

func errorsToFieldListImpl(rootErr validation.Errors, currentPath string, target map[string][]ValidationErrorJSON) {
	for field, err := range rootErr {
		if err == nil {
			continue
		}

		fieldPath := fmt.Sprintf("%s.%s", currentPath, field)
		if currentPath == "" {
			fieldPath = field
		}

		if vErrors, ok := lo.ErrorsAs[validation.Errors](err); ok {
			errorsToFieldListImpl(vErrors, fieldPath, target)
		} else if vError, ok := lo.ErrorsAs[validation.Error](err); ok {
			target[fieldPath] = append(target[field], ValidationErrorJSON{
				Code:    vError.Code(),
				Message: vError.Message(),
			})
		} else {
			target[fieldPath] = append(target[field], ValidationErrorJSON{
				Message: err.Error(),
			})
		}
	}
}
