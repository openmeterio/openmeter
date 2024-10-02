package errorsx

import (
	"errors"
	"fmt"
)

// WithPrefix annotates an error with a prefix.
func WithPrefix(err error, prefix string) error {
	if err == nil {
		return nil
	}

	type unwrapper interface {
		Unwrap() []error
	}

	// Deliberately checking for the unwrapper interface instead of the errors.Is function.
	// We only want to check the top-level error otherwise we may accidentally drop other wrappers from the error chain.
	e, ok := err.(unwrapper)
	if !ok {
		return fmt.Errorf("%s: %w", prefix, err)
	}

	errs := e.Unwrap()

	for i, err := range errs {
		errs[i] = WithPrefix(err, prefix)
	}

	return errors.Join(errs...)
}
