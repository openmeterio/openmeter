package goblvalidation

import (
	"github.com/invopop/validation"
	"github.com/samber/lo"
)

// NewError creates a new validation error with the given code and message.
// The validation.NewError returns a struct implementing the validation.Error object, however
// that is not Comparable by errors.Is (see: https://github.com/golang/go/blob/master/src/errors/wrap.go#L49)
//
// This makes any testcase checking for specific errors useless, but pointers are comparable.
func NewError(code, message string) validation.Error {
	return lo.ToPtr(validation.NewError(code, message).(validation.ErrorObject))
}
