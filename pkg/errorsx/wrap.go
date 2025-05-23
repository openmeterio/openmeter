package errorsx

import (
	"errors"
	"fmt"
	"strings"
)

func Join(errs ...error) error {
	joined := errors.Join(errs...)
	return FormatJoinedError(joined)
}

// FormatJoinedError formats an already-joined error nicely while preserving error chains.
func FormatJoinedError(err error) error {
	if err == nil {
		return nil
	}

	// Check if it's a joined error with multiple errors
	var joinedErr interface{ Unwrap() []error }
	if !errors.As(err, &joinedErr) {
		// Single error, just return
		return err
	}

	allErrors := joinedErr.Unwrap()
	if len(allErrors) <= 1 {
		// Single or no errors, just return
		return err
	}

	// Multiple errors: preserve chain for first, format the rest
	var additionalMsgs []string
	for i := 1; i < len(allErrors); i++ {
		additionalMsgs = append(additionalMsgs, allErrors[i].Error())
	}

	return fmt.Errorf("multiple errors: %w; %s",
		allErrors[0],
		strings.Join(additionalMsgs, "; "))
}
