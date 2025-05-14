package productcatalog

import (
	"errors"
	"fmt"
	"strings"
)

var _ error = (*InvalidResourceError)(nil)

type InvalidResourceError struct {
	Resource Resource `json:"resource"`
	Field    string   `json:"field"`
	Detail   string   `json:"detail"`
}

func (r InvalidResourceError) Error() string {
	attrs := []string{
		"key=" + r.Resource.Key,
	}

	for k, v := range r.Resource.Attributes {
		attrs = append(attrs, fmt.Sprintf("%s=%v", k, v))
	}

	return fmt.Sprintf("invalid %q field of %s [%s]: %+v", r.Field, r.Resource.Kind, strings.Join(attrs, " "), r.Detail)
}

func NewInvalidResourceError(resource Resource, field, detail string) error {
	return InvalidResourceError{
		Resource: resource,
		Field:    field,
		Detail:   detail,
	}
}

func UnwrapErrors[T InvalidResourceError](err error) []T {
	type wrappedError interface {
		Unwrap() error
	}

	if e, ok := err.(wrappedError); ok {
		return UnwrapErrors[T](e.Unwrap())
	}

	type wrappedErrors interface {
		Unwrap() []error
	}

	if e, ok := err.(wrappedErrors); ok {
		var targets []T

		for _, unwrappedErr := range e.Unwrap() {
			errs := UnwrapErrors[T](unwrappedErr)
			if len(errs) > 0 {
				targets = append(targets, errs...)
			}
		}

		return targets
	}

	target := new(T)

	if errors.As(err, target) {
		return []T{*target}
	}

	return nil
}
