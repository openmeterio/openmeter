package models

import "errors"

type Validator interface {
	// Validate returns an error if the instance of the Validator is invalid.
	Validate() error
}

type ValidatorFunc[T any] func(T) error

type CustomValidator[T any] interface {
	ValidateWith(...ValidatorFunc[T]) error
}

func Validate[T any](v T, validators ...ValidatorFunc[T]) error {
	var errs []error

	for _, validator := range validators {
		if err := validator(v); err != nil {
			errs = append(errs, err)
		}
	}

	return NewNillableGenericValidationError(errors.Join(errs...))
}
