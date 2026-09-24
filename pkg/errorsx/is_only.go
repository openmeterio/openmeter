package errorsx

import "errors"

// IsOnly reports whether every leaf in an error tree matches only errors in the allowed list.
func IsOnly(err error, allowed ...error) bool {
	if err == nil {
		// All errors are matching ;-)
		return true
	}

	if len(allowed) == 0 {
		return false
	}

	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		children := joined.Unwrap()
		if len(children) == 0 {
			return false
		}

		for _, child := range children {
			if !IsOnly(child, allowed...) {
				return false
			}
		}

		return true
	}

	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		if cause := wrapped.Unwrap(); cause != nil {
			return IsOnly(cause, allowed...)
		}
	}

	for _, candidate := range allowed {
		if candidate != nil && errors.Is(err, candidate) {
			return true
		}
	}

	return false
}
